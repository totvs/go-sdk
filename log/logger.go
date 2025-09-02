package logger

import (
    "context"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/rs/zerolog"
)

type ctxKey string

const traceIDKey ctxKey = "trace-id"

// New creates a zerolog logger that writes JSON to the provided writer and uses the given level.
func New(w io.Writer, level zerolog.Level) zerolog.Logger {
    zerolog.TimeFieldFormat = time.RFC3339
    logger := zerolog.New(w).With().Timestamp().Logger().Level(level)
    return logger
}

// NewDefault returns a JSON logger writing to stdout with level taken from LOG_LEVEL or Info.
func NewDefault() zerolog.Logger {
    lvl := zerolog.InfoLevel
    if s := os.Getenv("LOG_LEVEL"); s != "" {
        if l, err := zerolog.ParseLevel(s); err == nil {
            lvl = l
        }
    }
    return New(os.Stdout, lvl)
}

// ContextWithTrace returns a new context containing the provided trace id.
func ContextWithTrace(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
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

// WithTraceFromContext returns a logger that includes the `trace_id` field when a trace id exists in the context.
func WithTraceFromContext(ctx context.Context, l zerolog.Logger) zerolog.Logger {
    if tid := TraceIDFromContext(ctx); tid != "" {
        return l.With().Str("trace_id", tid).Logger()
    }
    return l
}

// HTTPMiddleware adds trace id from headers into the request context and logs incoming requests with trace_id.
// It looks for `X-Request-Id` or `X-Correlation-Id` headers.
func HTTPMiddleware(next http.Handler) http.Handler {
    base := NewDefault()
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        trace := r.Header.Get("X-Request-Id")
        if trace == "" {
            trace = r.Header.Get("X-Correlation-Id")
        }
        ctx := ContextWithTrace(r.Context(), trace)
        l := WithTraceFromContext(ctx, base).With().Str("method", r.Method).Str("path", r.URL.Path).Logger()
        l.Info().Msg("http request received")
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

