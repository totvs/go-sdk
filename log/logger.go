package log

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// zerolog-backed implementation of the fluent Event interface declared in facade.go.
type zerologEvent struct{ e *zerolog.Event }

func newZerologEvent(e *zerolog.Event) LogEvent { return &zerologEvent{e: e} }

func (z *zerologEvent) Str(k, v string) LogEvent             { z.e = z.e.Str(k, v); return z }
func (z *zerologEvent) Int(k string, v int) LogEvent         { z.e = z.e.Int(k, v); return z }
func (z *zerologEvent) Int64(k string, v int64) LogEvent     { z.e = z.e.Int64(k, v); return z }
func (z *zerologEvent) Uint(k string, v uint) LogEvent       { z.e = z.e.Uint(k, v); return z }
func (z *zerologEvent) Uint64(k string, v uint64) LogEvent   { z.e = z.e.Uint64(k, v); return z }
func (z *zerologEvent) Bool(k string, v bool) LogEvent       { z.e = z.e.Bool(k, v); return z }
func (z *zerologEvent) Float32(k string, v float32) LogEvent { z.e = z.e.Float32(k, v); return z }
func (z *zerologEvent) Float64(k string, v float64) LogEvent { z.e = z.e.Float64(k, v); return z }
func (z *zerologEvent) Interface(k string, v interface{}) LogEvent {
	z.e = z.e.Interface(k, v)
	return z
}
func (z *zerologEvent) Err(err error) LogEvent { z.e = z.e.Err(err); return z }
func (z *zerologEvent) Msg(msg string)         { z.e.Msg(msg) }
func (z *zerologEvent) Msgf(format string, args ...interface{}) {
	z.e.Msg(fmt.Sprintf(format, args...))
}

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

// loggerImpl is the concrete logger implementation based on zerolog.
// It is unexported; callers should use the `LoggerFacade` abstraction.
type loggerImpl struct{ l zerolog.Logger }

// newLogger creates a logger that writes JSON to the provided writer and uses the given level.
func newLogger(w io.Writer, level Level) loggerImpl {
	zerolog.TimeFieldFormat = time.RFC3339
	lg := zerolog.New(w).With().Timestamp().Logger().Level(level.toZerolog())
	return loggerImpl{l: lg}
}

// newDefaultLogger returns a JSON logger writing to stdout with level taken from LOG_LEVEL or Info.
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

// WithTraceFromContext returns a loggerImpl that includes the `trace_id` field when a trace id exists in the context.
func WithTraceFromContext(ctx context.Context, l loggerImpl) loggerImpl {
	if tid := TraceIDFromContext(ctx); tid != "" {
		return loggerImpl{l: l.l.With().Str(TraceIDField, tid).Logger()}
	}
	return l
}

// NOTE: context-based logger storage is handled via the facade helpers in `facade.go`.
// Callers should use the LoggerFacade abstraction for cross-package consistency.

// withFields returns a loggerImpl augmented with the provided fields.
// It uses type-specific setters when possible, falling back to `Interface`.
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

// withField returns a new loggerImpl with a single additional field.
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

// GenerateTraceID returns a new trace id. Exported for use by integration
// helpers (e.g., HTTP middleware) that need to generate or fallback a trace id.
func GenerateTraceID() string { return generateTraceID() }
