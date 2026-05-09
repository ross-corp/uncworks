package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

// CreateAgentRunRateLimiter is a ConnectRPC interceptor that applies rate limiting
// specifically to the CreateAgentRun method.
type CreateAgentRunRateLimiter struct {
	limiter *rate.Limiter
	enabled bool
}

// NewCreateAgentRunRateLimiter creates a new interceptor with the given RPS and burst.
// If RPS <= 0 or burst <= 0, rate limiting is disabled.
func NewCreateAgentRunRateLimiter(rps float64, burst int) *CreateAgentRunRateLimiter {
	if rps <= 0 || burst <= 0 {
		slog.Debug("CreateAgentRun rate limiting disabled", "rps", rps, "burst", burst)
		return &CreateAgentRunRateLimiter{enabled: false}
	}
	
	slog.Info("CreateAgentRun rate limiting enabled", "rps", rps, "burst", burst)
	return &CreateAgentRunRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		enabled: true,
	}
}

// WrapUnary implements connect.UnaryInterceptorFunc.
func (rl *CreateAgentRunRateLimiter) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Only apply rate limiting to CreateAgentRun method
		if req.Spec().Procedure == "/aot.api.v1.AOTService/CreateAgentRun" {
			if rl.enabled && !rl.limiter.Allow() {
				slog.Warn("CreateAgentRun rate limit exceeded", "procedure", req.Spec().Procedure)
				return nil, connect.NewError(connect.CodeResourceExhausted, 
					fmt.Errorf("rate limit exceeded for CreateAgentRun"))
			}
		}
		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.StreamingClientInterceptorFunc.
func (rl *CreateAgentRunRateLimiter) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	// No rate limiting for client streaming
	return next
}

// WrapStreamingHandler implements connect.StreamingHandlerInterceptorFunc.
func (rl *CreateAgentRunRateLimiter) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	// No rate limiting for handler streaming (WatchAgentRun is a server-streaming method)
	return next
}