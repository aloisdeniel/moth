package metrics

import (
	"context"
	"time"

	"connectrpc.com/connect"
)

// Interceptor returns a connect unary interceptor that records every RPC in
// the registry: a request counter labelled by procedure and result code
// ("ok" on success, the connect.Code string otherwise) and a latency
// observation in the duration histogram. Place it in the shared observability
// chain so it covers every service.
func (r *Registry) Interceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			code := "ok"
			if err != nil {
				code = connect.CodeOf(err).String()
			}
			r.ObserveRPC(req.Spec().Procedure, code, time.Since(start).Seconds())
			return resp, err
		}
	}
}
