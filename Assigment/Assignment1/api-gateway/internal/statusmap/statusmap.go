// Package statusmap converts gRPC status codes to HTTP status codes
// so the REST API can return correct HTTP semantics.
package statusmap

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToHTTP converts a gRPC error to (httpStatus, message).
// If err is nil, returns (200, "").
func ToHTTP(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, err.Error()
	}
	msg := st.Message()
	if msg == "" {
		msg = st.Code().String()
	}
	return codeToHTTP(st.Code()), msg
}

func codeToHTTP(c codes.Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Canceled:
		return 499 // client closed request
	default:
		return http.StatusInternalServerError
	}
}
