package transaction

import "context"

type contextTraceIDKeyType string

const (
	contextTraceIDKey contextTraceIDKeyType = "trace-id"
)

// TraceIDFromContext extracts the trace id from the context, if present.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(contextTraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ContextWithTrace returns a new context containing the provided trace id.
func ContextWithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, contextTraceIDKey, traceID)
}
