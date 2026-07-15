package authrpc

import (
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// ErrorDomain identifies moth in google.rpc.ErrorInfo details.
const ErrorDomain = "moth.dev"

// Stable machine-readable error reasons carried as google.rpc.ErrorInfo
// details; the SDK maps them to typed errors.
const (
	ReasonInvalidCredentials  = "INVALID_CREDENTIALS"
	ReasonEmailNotVerified    = "EMAIL_NOT_VERIFIED"
	ReasonEmailAlreadyExists  = "EMAIL_ALREADY_EXISTS"
	ReasonSignupClosed        = "SIGNUP_CLOSED"
	ReasonWeakPassword        = "WEAK_PASSWORD"
	ReasonInvalidEmail        = "INVALID_EMAIL"
	ReasonInvalidToken        = "INVALID_TOKEN"
	ReasonInvalidRefreshToken = "INVALID_REFRESH_TOKEN"
	ReasonRefreshTokenReused  = "REFRESH_TOKEN_REUSED"
	ReasonInvalidAccessToken  = "INVALID_ACCESS_TOKEN"
	ReasonUserDisabled        = "USER_DISABLED"
	ReasonRateLimited         = "RATE_LIMITED"
)

// newError builds a connect error carrying a stable reason detail.
func newError(code connect.Code, reason, msg string) *connect.Error {
	err := connect.NewError(code, errors.New(msg))
	if detail, derr := connect.NewErrorDetail(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: ErrorDomain,
	}); derr == nil {
		err.AddDetail(detail)
	}
	return err
}

// ErrorReason extracts the moth reason from a connect error, or "".
func ErrorReason(err error) string {
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		return ""
	}
	for _, d := range cerr.Details() {
		msg, err := d.Value()
		if err != nil {
			continue
		}
		if info, ok := msg.(*errdetails.ErrorInfo); ok && info.Domain == ErrorDomain {
			return info.Reason
		}
	}
	return ""
}

func errInvalidCredentials() *connect.Error {
	return newError(connect.CodeUnauthenticated, ReasonInvalidCredentials, "invalid email or password")
}

func errInvalidAccessToken(msg string) *connect.Error {
	return newError(connect.CodeUnauthenticated, ReasonInvalidAccessToken, msg)
}

func errUserDisabled() *connect.Error {
	return newError(connect.CodePermissionDenied, ReasonUserDisabled, "account is disabled")
}

func errInvalidEmailToken() *connect.Error {
	return newError(connect.CodeInvalidArgument, ReasonInvalidToken, "invalid or expired token")
}

func errInternal(err error) *connect.Error {
	return connect.NewError(connect.CodeInternal, err)
}
