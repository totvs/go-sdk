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
const loggerKey ctxKey = "logger"

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

// ContextWithLogger stores a zerolog.Logger in the context so callers can
// inject a logger that will be used by library code.
func ContextWithLogger(ctx context.Context, l zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFromContext extracts a zerolog.Logger from the context. The boolean
// indicates whether a logger was present.
func LoggerFromContext(ctx context.Context) (zerolog.Logger, bool) {
	if ctx == nil {
		return zerolog.Logger{}, false
	}
	if v := ctx.Value(loggerKey); v != nil {
		if lg, ok := v.(zerolog.Logger); ok {
			return lg, true
		}
	}
	return zerolog.Logger{}, false
}

// FromContext returns the logger stored in the context or a sensible default
// created by NewDefault when none is present.
func FromContext(ctx context.Context) zerolog.Logger {
	if l, ok := LoggerFromContext(ctx); ok {
		return l
	}
	return NewDefault()
}

// WithFields returns a logger augmented with the provided fields. It accepts
// a map of string->interface{} and will use type-specific setters when
// possible, falling back to `Interface` for unknown types.
func WithFields(l zerolog.Logger, fields map[string]interface{}) zerolog.Logger {
	c := l.With()
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			c = c.Str(k, val)
		case int:
			c = c.Int(k, val)
		case int64:
			c = c.Int64(k, val)
		case bool:
			c = c.Bool(k, val)
		case float64:
			c = c.Float64(k, val)
		default:
			c = c.Interface(k, val)
		}
	}
	return c.Logger()
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
