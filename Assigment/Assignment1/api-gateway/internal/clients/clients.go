// Package clients holds the gateway's gRPC clients to Service A and Service B.
//
// The gateway has NO database; it just proxies REST to gRPC.
package clients

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	membersv1 "github.com/sangay/web303/proto/members/v1"
	checkinsv1 "github.com/sangay/web303/proto/checkins/v1"
)

// Clients bundles the two downstream gRPC clients.
type Clients struct {
	Members  membersv1.MemberServiceClient
	CheckIns checkinsv1.CheckInServiceClient

	membersConn  *grpc.ClientConn
	checkinsConn *grpc.ClientConn
}

// New dials both services. Dials are lazy — Services can be down at startup.
func New(membersAddr, checkinsAddr string) (*Clients, error) {
	membersConn, err := grpc.NewClient(membersAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial members: %w", err)
	}
	checkinsConn, err := grpc.NewClient(checkinsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = membersConn.Close()
		return nil, fmt.Errorf("dial checkins: %w", err)
	}

	return &Clients{
		Members:      membersv1.NewMemberServiceClient(membersConn),
		CheckIns:     checkinsv1.NewCheckInServiceClient(checkinsConn),
		membersConn:  membersConn,
		checkinsConn: checkinsConn,
	}, nil
}

func (c *Clients) Close() {
	if c.membersConn != nil {
		_ = c.membersConn.Close()
	}
	if c.checkinsConn != nil {
		_ = c.checkinsConn.Close()
	}
}
