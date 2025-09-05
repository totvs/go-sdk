package logger

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type ctxKey string

const traceIDKey ctxKey = "trace-id"
const loggerKey ctxKey = "logger"

// Level represents a logging level independent from zerolog so callers don't
// need to import zerolog directly.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l Level) toZerolog() zerolog.Level {
	switch l {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// Logger is a lightweight wrapper around zerolog.Logger that keeps the
// concrete zerolog type internal to the package. Consumers can use the
// returned Logger without importing zerolog.
type Logger struct{ l zerolog.Logger }

// New creates a logger that writes JSON to the provided writer and uses the given level.
func New(w io.Writer, level Level) Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	lg := zerolog.New(w).With().Timestamp().Logger().Level(level.toZerolog())
	return Logger{l: lg}
}

// NewDefault returns a JSON logger writing to stdout with level taken from LOG_LEVEL or Info.
func NewDefault() Logger {
	lvl := InfoLevel
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		switch s {
		case "DEBUG", "debug":
			lvl = DebugLevel
		case "INFO", "info":
			lvl = InfoLevel
		case "WARN", "warn", "WARNING", "warning":
			lvl = WarnLevel
		case "ERROR", "error":
			lvl = ErrorLevel
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
func WithTraceFromContext(ctx context.Context, l Logger) Logger {
	if tid := TraceIDFromContext(ctx); tid != "" {
		return Logger{l: l.l.With().Str("trace_id", tid).Logger()}
	}
	return l
}

// ContextWithLogger stores a Logger in the context so callers can inject a logger that will be used by library code.
func ContextWithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFromContext extracts a Logger from the context. The boolean indicates whether a logger was present.
func LoggerFromContext(ctx context.Context) (Logger, bool) {
	if ctx == nil {
		return Logger{}, false
	}
	if v := ctx.Value(loggerKey); v != nil {
		if lg, ok := v.(Logger); ok {
			return lg, true
		}
	}
	return Logger{}, false
}

// FromContext returns the logger stored in the context or a sensible default created by NewDefault when none is present.
func FromContext(ctx context.Context) Logger {
	if l, ok := LoggerFromContext(ctx); ok {
		return l
	}
	return NewDefault()
}

// WithFields returns a logger augmented with the provided fields. It accepts a map of string->interface{} and will use
// type-specific setters when possible, falling back to `Interface` for unknown types.
func WithFields(l Logger, fields map[string]interface{}) Logger {
	c := l.l.With()
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			c = c.Str(k, val)
		case int:
			c = c.Int(k, val)
		case int64:
			c = c.Int64(k, val)
		case uint:
			c = c.Uint(k, val)
		case uint64:
			c = c.Uint64(k, val)
		case bool:
			c = c.Bool(k, val)
		case float32:
			c = c.Float32(k, val)
		case float64:
			c = c.Float64(k, val)
		default:
			c = c.Interface(k, val)
		}
	}
	return Logger{l: c.Logger()}
}

// WithField returns a new logger with a single additional field.
func (l Logger) WithField(k string, v interface{}) Logger {
	return Logger{l: l.l.With().Interface(k, v).Logger()}
}

// WithFields method mirrors the package function for convenience.
func (l Logger) WithFields(fields map[string]interface{}) Logger { return WithFields(l, fields) }

// Convenience message helpers so callers don't need to call the zerolog event methods directly.
func (l Logger) InfoMsg(msg string)  { l.Info().Msg(msg) }
func (l Logger) DebugMsg(msg string) { l.Debug().Msg(msg) }
func (l Logger) WarnMsg(msg string)  { l.Warn().Msg(msg) }
func (l Logger) ErrorMsg(msg string) { l.Error().Msg(msg) }

// generateTraceID creates a 16-byte hex id (UUID-like) for request tracing.
func generateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based id (very unlikely)
		return time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	// encode as hex without external deps
	const hextable = "0123456789abcdef"
	dst := make([]byte, 32)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

// Info/Debug/Warn/Error expose the underlying zerolog event so callers can use the familiar API
// without importing zerolog themselves.
func (l Logger) Info() *zerolog.Event  { tmp := l.l; return (&tmp).Info() }
func (l Logger) Debug() *zerolog.Event { tmp := l.l; return (&tmp).Debug() }
func (l Logger) Warn() *zerolog.Event  { tmp := l.l; return (&tmp).Warn() }
func (l Logger) Error() *zerolog.Event { tmp := l.l; return (&tmp).Error() }

// HTTPMiddlewareWithLogger returns a middleware using the provided base logger.
func HTTPMiddlewareWithLogger(base Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace := r.Header.Get("X-Request-Id")
			if trace == "" {
				trace = r.Header.Get("X-Correlation-Id")
			}
			if trace == "" {
				trace = generateTraceID()
			}
			ctx := ContextWithTrace(r.Context(), trace)
			// include trace and basic request info
			l := WithTraceFromContext(ctx, base)
			tmp := l.l.With().Str("method", r.Method).Str("path", r.URL.Path).Str("trace_id", trace).Logger()
			(&tmp).Info().Msg("http request received")
			// ensure response contains the trace id so callers can correlate
			if w.Header().Get("X-Request-Id") == "" {
				w.Header().Set("X-Request-Id", trace)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HTTPMiddleware is a convenience wrapper that uses the default logger.
func HTTPMiddleware(next http.Handler) http.Handler {
	return HTTPMiddlewareWithLogger(NewDefault())(next)
}
