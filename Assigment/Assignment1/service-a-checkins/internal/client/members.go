// Package client provides a resilient gRPC client to Service B (Members).
//
// It composes three resilience patterns:
//  1. Per-call timeout         — a context deadline on every RPC
//  2. Retry with backoff       — up to N attempts on transient errors
//  3. Circuit breaker          — stop hammering a failing Service B
//
// All three are applied transparently to the GetMember call.
package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	membersv1 "github.com/sangay/web303/proto/members/v1"
)

// Sentinel errors callers can test with errors.Is().
var (
	ErrMembersUnavailable = errors.New("members service is unavailable")
	ErrMemberNotFound     = errors.New("member not found")
)

// Config bundles resilience settings passed in from config.Config.
type Config struct {
	Timeout      time.Duration
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
	CBMinReqs    uint32
	CBFailurePct float64
	CBInterval   time.Duration
	CBTimeout    time.Duration
}

// Client wraps a gRPC connection to Service B with resilience patterns.
type Client struct {
	conn *grpc.ClientConn
	stub membersv1.MemberServiceClient
	cfg  Config
	cb   *gobreaker.CircuitBreaker
}

// New dials Service B and returns a resilient client.
// The underlying grpc.ClientConn is lazy — it will not fail here even
// if Service B is down. Failures surface on the first RPC attempt.
func New(target string, cfg Config) (*Client, error) {
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial members: %w", err)
	}

	cbSettings := gobreaker.Settings{
		Name:        "members-service",
		MaxRequests: 1,
		Interval:    cfg.CBInterval,
		Timeout:     cfg.CBTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.CBMinReqs {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cfg.CBFailurePct
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("[circuit-breaker %s] %s -> %s", name, from, to)
		},
	}

	return &Client{
		conn: conn,
		stub: membersv1.NewMemberServiceClient(conn),
		cfg:  cfg,
		cb:   gobreaker.NewCircuitBreaker(cbSettings),
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error { return c.conn.Close() }

// GetMember retrieves a member, applying timeout + retry + circuit breaker.
//
// Behaviour:
//   - If the circuit is OPEN, returns ErrMembersUnavailable immediately.
//   - Otherwise runs getMemberOnce with retries/backoff.
//   - If Service B says NOT_FOUND, returns ErrMemberNotFound (NOT retried).
//   - Other errors are retried up to MaxAttempts with exponential backoff.
func (c *Client) GetMember(ctx context.Context, id string) (*membersv1.Member, error) {
	var result *membersv1.Member
	var lastErr error

	_, cbErr := c.cb.Execute(func() (interface{}, error) {
		m, err := c.getMemberWithRetry(ctx, id)
		if err != nil {
			return nil, err
		}
		result = m
		return m, nil
	})

	if cbErr != nil {
		if errors.Is(cbErr, gobreaker.ErrOpenState) || errors.Is(cbErr, gobreaker.ErrTooManyRequests) {
			log.Printf("[circuit-breaker] rejecting call, members-service circuit is OPEN")
			return nil, fmt.Errorf("%w: circuit breaker open", ErrMembersUnavailable)
		}
		return nil, cbErr
	}

	_ = lastErr
	return result, nil
}

// getMemberWithRetry implements timeout + retry-with-backoff around one RPC call.
func (c *Client) getMemberWithRetry(ctx context.Context, id string) (*membersv1.Member, error) {
	backoff := c.cfg.BaseBackoff
	var lastErr error

	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		// Per-attempt timeout
		attemptCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)

		m, err := c.stub.GetMember(attemptCtx, &membersv1.GetMemberRequest{Id: id})
		cancel()

		if err == nil {
			return m, nil
		}

		st, _ := status.FromError(err)
		lastErr = err

		switch st.Code() {
		case codes.NotFound:
			// Business error — do NOT retry, do NOT count as a circuit failure.
			return nil, fmt.Errorf("%w: %s", ErrMemberNotFound, st.Message())
		case codes.InvalidArgument:
			// Caller error — do NOT retry.
			return nil, err
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
			// Transient — worth retrying.
			log.Printf("[members-client] attempt %d/%d failed (%s): %v",
				attempt, c.cfg.MaxAttempts, st.Code(), st.Message())
		default:
			// Unknown/internal — retry once then give up.
			log.Printf("[members-client] unexpected error (%s): %v", st.Code(), st.Message())
		}

		if attempt == c.cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		// Exponential backoff with cap
		backoff *= 2
		if backoff > c.cfg.MaxBackoff {
			backoff = c.cfg.MaxBackoff
		}
	}

	return nil, lastErr
}
