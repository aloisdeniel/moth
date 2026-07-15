package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/aloisdeniel/moth/internal/token"
)

type requestIDKey struct{}

// requestIDHeader carries the request ID back to clients.
const requestIDHeader = "X-Request-Id"

// RequestIDFromContext returns the request ID assigned by the interceptor.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func newRequestIDInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			id := token.Random(8)
			ctx = context.WithValue(ctx, requestIDKey{}, id)
			resp, err := next(ctx, req)
			if resp != nil {
				resp.Header().Set(requestIDHeader, id)
			}
			return resp, err
		}
	}
}

func newLoggingInterceptor(log *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			attrs := []any{
				"procedure", req.Spec().Procedure,
				"duration", time.Since(start).Round(time.Microsecond).String(),
				"request_id", RequestIDFromContext(ctx),
			}
			if err != nil {
				attrs = append(attrs, "code", connect.CodeOf(err).String(), "error", err.Error())
				log.WarnContext(ctx, "rpc", attrs...)
				return resp, err
			}
			log.InfoContext(ctx, "rpc", attrs...)
			return resp, nil
		}
	}
}

func newRecoveryInterceptor(log *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			defer func() {
				if r := recover(); r != nil {
					log.ErrorContext(ctx, "rpc panic",
						"procedure", req.Spec().Procedure,
						"request_id", RequestIDFromContext(ctx),
						"panic", fmt.Sprint(r))
					err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
				}
			}()
			return next(ctx, req)
		}
	}
}
