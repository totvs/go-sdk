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
const loggedKey ctxKey = "logged"

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
// loggerImpl is the concrete logger implementation based on zerolog.
// It is unexported so callers are encouraged to use the LoggerFacade
// abstraction instead of depending on zerolog directly.
type loggerImpl struct{ l zerolog.Logger }

// New creates a logger that writes JSON to the provided writer and uses the given level.
func newLogger(w io.Writer, level Level) loggerImpl {
	zerolog.TimeFieldFormat = time.RFC3339
	lg := zerolog.New(w).With().Timestamp().Logger().Level(level.toZerolog())
	return loggerImpl{l: lg}
}

// NewDefault returns a JSON logger writing to stdout with level taken from LOG_LEVEL or Info.
func newDefaultLogger() loggerImpl {
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
	return newLogger(os.Stdout, lvl)
}

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

// WithTraceFromContext returns a logger that includes the `trace_id` field when a trace id exists in the context.
func WithTraceFromContext(ctx context.Context, l loggerImpl) loggerImpl {
	if tid := TraceIDFromContext(ctx); tid != "" {
		return loggerImpl{l: l.l.With().Str("trace_id", tid).Logger()}
	}
	return l
}

// NOTE: context-based logger storage is handled via the facade helpers in `facade.go`.
// The concrete Logger remains available for callers that need zerolog-specific features,
// but context helpers that store/retrieve loggers use the LoggerFacade abstraction.

// WithFields returns a logger augmented with the provided fields. It accepts a map of string->interface{} and will use
// type-specific setters when possible, falling back to `Interface` for unknown types.
func withFields(l loggerImpl, fields map[string]interface{}) loggerImpl {
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
	return loggerImpl{l: c.Logger()}
}

// WithField returns a new logger with a single additional field.
func (l loggerImpl) withField(k string, v interface{}) loggerImpl {
	return loggerImpl{l: l.l.With().Interface(k, v).Logger()}
}

// withFields method mirrors the package function for convenience.
func (l loggerImpl) withFields(fields map[string]interface{}) loggerImpl {
	return withFields(l, fields)
}

// Convenience message helpers so callers inside the package don't need to call the zerolog event methods directly.
func (l loggerImpl) InfoMsg(msg string)  { l.l.Info().Msg(msg) }
func (l loggerImpl) DebugMsg(msg string) { l.l.Debug().Msg(msg) }
func (l loggerImpl) WarnMsg(msg string)  { l.l.Warn().Msg(msg) }
func (l loggerImpl) ErrorMsg(msg string) { l.l.Error().Msg(msg) }

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
// Note: we intentionally do not expose methods that return zerolog-specific
// types. Callers should use the LoggerFacade abstraction instead.

// HTTPMiddlewareWithLogger returns a middleware using the provided base logger.
func HTTPMiddlewareWithLogger(base LoggerFacade) func(http.Handler) http.Handler {
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
			// include trace and basic request info via the facade API
			l := base.WithTraceFromContext(ctx)
			l2 := l.WithFields(map[string]interface{}{"method": r.Method, "path": r.URL.Path, "trace_id": trace})
			l2.Info("http request received")

			// mark that middleware already logged this request and inject the
			// (facade) logger into the request context so handlers can extract
			// and reuse the same logger instance without duplicating the same message.
			ctx = ContextWithLogged(ctx)
			ctx = ContextWithLogger(ctx, l2)
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
	return HTTPMiddlewareWithLogger(NewDefaultFacade())(next)
}
