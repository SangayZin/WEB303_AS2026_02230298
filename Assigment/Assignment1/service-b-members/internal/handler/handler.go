package handler

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	membersv1 "github.com/sangay/web303/proto/members/v1"
	"github.com/sangay/web303/members/internal/db"
)

// MemberHandler implements membersv1.MemberServiceServer.
type MemberHandler struct {
	membersv1.UnimplementedMemberServiceServer
	store *db.Store
}

func New(store *db.Store) *MemberHandler {
	return &MemberHandler{store: store}
}

// ---------- validation helpers ----------
var (
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	phoneRe = regexp.MustCompile(`^[0-9]{7,15}$`)
	validPlans = map[string]bool{"basic": true, "premium": true, "vip": true}
)

// validateMember checks all required fields and formats.
// Returns nil on success, or a gRPC INVALID_ARGUMENT error.
func validateMember(name, email, phone, plan string) error {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	plan = strings.TrimSpace(plan)

	if name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if len(name) > 100 {
		return status.Error(codes.InvalidArgument, "name must be ≤ 100 characters")
	}
	if email == "" {
		return status.Error(codes.InvalidArgument, "email is required")
	}
	if !emailRe.MatchString(email) {
		return status.Error(codes.InvalidArgument, "email format is invalid")
	}
	if phone == "" {
		return status.Error(codes.InvalidArgument, "phone is required")
	}
	if !phoneRe.MatchString(phone) {
		return status.Error(codes.InvalidArgument, "phone must be 7–15 digits")
	}
	if plan == "" {
		return status.Error(codes.InvalidArgument, "plan is required")
	}
	if !validPlans[plan] {
		return status.Error(codes.InvalidArgument, "plan must be one of: basic, premium, vip")
	}
	return nil
}

// mapDBError converts internal DB errors to gRPC status errors.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, db.ErrNotFound) {
		return status.Error(codes.NotFound, "member not found")
	}
	return status.Errorf(codes.Internal, "database error: %v", err)
}

// ---------- RPC implementations ----------

func (h *MemberHandler) CreateMember(ctx context.Context, req *membersv1.CreateMemberRequest) (*membersv1.Member, error) {
	if err := validateMember(req.Name, req.Email, req.Phone, req.Plan); err != nil {
		return nil, err
	}
	m := &membersv1.Member{
		Name:  strings.TrimSpace(req.Name),
		Email: strings.ToLower(strings.TrimSpace(req.Email)),
		Phone: strings.TrimSpace(req.Phone),
		Plan:  strings.TrimSpace(req.Plan),
	}
	created, err := h.store.Create(ctx, m)
	if err != nil {
		// unique email violation from Postgres
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, status.Error(codes.AlreadyExists, "a member with this email already exists")
		}
		return nil, mapDBError(err)
	}
	return created, nil
}

func (h *MemberHandler) GetMember(ctx context.Context, req *membersv1.GetMemberRequest) (*membersv1.Member, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	m, err := h.store.Get(ctx, req.Id)
	if err != nil {
		return nil, mapDBError(err)
	}
	return m, nil
}

func (h *MemberHandler) UpdateMember(ctx context.Context, req *membersv1.UpdateMemberRequest) (*membersv1.Member, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := validateMember(req.Name, req.Email, req.Phone, req.Plan); err != nil {
		return nil, err
	}
	m := &membersv1.Member{
		Id:    req.Id,
		Name:  strings.TrimSpace(req.Name),
		Email: strings.ToLower(strings.TrimSpace(req.Email)),
		Phone: strings.TrimSpace(req.Phone),
		Plan:  strings.TrimSpace(req.Plan),
	}
	updated, err := h.store.Update(ctx, m)
	if err != nil {
		return nil, mapDBError(err)
	}
	return updated, nil
}

func (h *MemberHandler) DeleteMember(ctx context.Context, req *membersv1.DeleteMemberRequest) (*emptypb.Empty, error) {
	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	if err := h.store.Delete(ctx, req.Id); err != nil {
		return nil, mapDBError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *MemberHandler) ListMembers(ctx context.Context, req *membersv1.ListMembersRequest) (*membersv1.ListMembersResponse, error) {
	// simple offset-based pagination; page_token == offset as string
	offset := 0
	if req.PageToken != "" {
		var n int
		if _, err := fmtSscan(req.PageToken, &n); err == nil && n >= 0 {
			offset = n
		}
	}
	list, err := h.store.List(ctx, req.PageSize, offset)
	if err != nil {
		return nil, mapDBError(err)
	}
	resp := &membersv1.ListMembersResponse{Members: list}
	if int32(len(list)) == req.PageSize && req.PageSize > 0 {
		resp.NextPageToken = itoa(offset + int(req.PageSize))
	}
	return resp, nil
}

// tiny helpers to avoid extra imports in this file
func fmtSscan(s string, n *int) (int, error) {
	// strconv.Atoi equivalent without importing strconv at top
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
	}
	if s == "" {
		return 0, errors.New("empty")
	}
	v := 0
	for _, c := range s {
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

// keep time import used (used indirectly by proto types)
var _ = time.Now
