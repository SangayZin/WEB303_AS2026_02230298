package handler

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	checkinsv1 "github.com/sangay/web303/proto/checkins/v1"
	"github.com/sangay/web303/checkins/internal/client"
	"github.com/sangay/web303/checkins/internal/db"
)

// CheckInHandler implements checkinsv1.CheckInServiceServer.
type CheckInHandler struct {
	checkinsv1.UnimplementedCheckInServiceServer
	store   *db.Store
	members *client.Client   // resilient client to Service B
}

func New(store *db.Store, members *client.Client) *CheckInHandler {
	return &CheckInHandler{store: store, members: members}
}

// ---------- validation ----------

const maxNotesLen = 200

func validateCreate(memberID, notes string) error {
	if strings.TrimSpace(memberID) == "" {
		return status.Error(codes.InvalidArgument, "member_id is required")
	}
	if len(notes) > maxNotesLen {
		return status.Errorf(codes.InvalidArgument, "notes must be ≤ %d characters", maxNotesLen)
	}
	return nil
}

// mapDBError converts internal DB errors to gRPC status errors.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, db.ErrNotFound) {
		return status.Error(codes.NotFound, "check-in not found")
	}
	return status.Errorf(codes.Internal, "database error: %v", err)
}

// mapMembersError converts the resilient client's errors to gRPC status errors.
// This is where cross-service failure becomes a meaningful status for the caller.
func mapMembersError(err error) error {
	switch {
	case errors.Is(err, client.ErrMemberNotFound):
		return status.Error(codes.NotFound, "referenced member does not exist")
	case errors.Is(err, client.ErrMembersUnavailable):
		return status.Error(codes.Unavailable, "members service is unavailable, please retry shortly")
	default:
		// Preserve caller-caused errors (InvalidArgument etc.) that came back from B
		if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown && st.Code() != codes.Internal {
			return err
		}
		return status.Error(codes.Unavailable, "members service call failed")
	}
}

// ---------- RPC implementations ----------

// CreateCheckIn validates the input, verifies the member exists via Service B,
// then stores the check-in.
func (h *CheckInHandler) CreateCheckIn(ctx context.Context, req *checkinsv1.CreateCheckInRequest) (*checkinsv1.CheckIn, error) {
	if err := validateCreate(req.MemberId, req.Notes); err != nil {
		return nil, err
	}

	// Cross-service call — MUST succeed before we store anything.
	// This is where timeout + retry + circuit breaker apply.
	if _, err := h.members.GetMember(ctx, req.MemberId); err != nil {
		return nil, mapMembersError(err)
	}

	c := &checkinsv1.CheckIn{
		MemberId: strings.TrimSpace(req.MemberId),
		Notes:    strings.TrimSpace(req.Notes),
	}
	created, err := h.store.Create(ctx, c)
	if err != nil {
		return nil, mapDBError(err)
	}
	return created, nil
}

func (h *CheckInHandler) GetCheckIn(ctx context.Context, req *checkinsv1.GetCheckInRequest) (*checkinsv1.CheckIn, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	c, err := h.store.Get(ctx, req.Id)
	if err != nil {
		return nil, mapDBError(err)
	}

	// Enrich: fetch member so we can confirm it still exists.
	// If members-service is down, we still return the check-in but that's
	// the resilience fallback: the system degrades gracefully, doesn't hang.
	if _, err := h.members.GetMember(ctx, c.MemberId); err != nil {
		if errors.Is(err, client.ErrMembersUnavailable) {
			// fallback: log and continue (return the check-in without enrichment)
			// intentionally no error returned
			return c, nil
		}
		// Member truly not found — still return check-in; business decision.
		return c, nil
	}
	return c, nil
}

func (h *CheckInHandler) UpdateCheckIn(ctx context.Context, req *checkinsv1.UpdateCheckInRequest) (*checkinsv1.CheckIn, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if len(req.Notes) > maxNotesLen {
		return nil, status.Errorf(codes.InvalidArgument, "notes must be ≤ %d characters", maxNotesLen)
	}
	updated, err := h.store.Update(ctx, req.Id, req.Notes)
	if err != nil {
		return nil, mapDBError(err)
	}
	return updated, nil
}

func (h *CheckInHandler) DeleteCheckIn(ctx context.Context, req *checkinsv1.DeleteCheckInRequest) (*emptypb.Empty, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := h.store.Delete(ctx, req.Id); err != nil {
		return nil, mapDBError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *CheckInHandler) ListCheckIns(ctx context.Context, req *checkinsv1.ListCheckInsRequest) (*checkinsv1.ListCheckInsResponse, error) {
	offset := 0
	if req.PageToken != "" {
		var n int
		if _, err := fmtSscan(req.PageToken, &n); err == nil && n >= 0 {
			offset = n
		}
	}
	list, err := h.store.List(ctx, req.MemberId, req.PageSize, offset)
	if err != nil {
		return nil, mapDBError(err)
	}
	resp := &checkinsv1.ListCheckInsResponse{Checkins: list}
	if int32(len(list)) == req.PageSize && req.PageSize > 0 {
		resp.NextPageToken = itoa(offset + int(req.PageSize))
	}
	return resp, nil
}

// tiny integer parse/format helpers (avoid importing strconv in this file)
func fmtSscan(s string, n *int) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	v := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		v = v*10 + int(c-'0')
	}
	*n = v
	return 1, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
