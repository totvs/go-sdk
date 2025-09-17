package trace

import (
	"context"
	"crypto/rand"
	"time"
)

// Public constants for trace header and field names used across projects.
const (
	TraceIDHTTPHeader            = "X-Request-Id"
	TraceIDHTTPCorrelationHeader = "X-Correlation-Id"
	TraceIDField                 = "trace_id"
)

type ctxKey string

const (
	traceIDKey ctxKey = "trace-id"
	loggedKey  ctxKey = "logged"
)

// ContextWithTrace returns a new context containing the provided trace id.
func ContextWithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// ContextWithLogged marks the context indicating the middleware already emitted
// a request-level log entry for this request.
func ContextWithLogged(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggedKey, true)
}

// LoggedFromContext returns true if the middleware logged the request already.
func LoggedFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if v := ctx.Value(loggedKey); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// TraceIDFromContext extracts the trace id from the context, if present.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GenerateTraceID returns a new 16-byte hex trace id.
func GenerateTraceID() string { return generateTraceID() }

func generateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	const hextable = "0123456789abcdef"
	dst := make([]byte, 32)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}
