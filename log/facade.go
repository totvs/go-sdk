// Package log provides a logging abstraction (facade) to decouple application code
// from specific logging implementations like zerolog.
//
// Quick start:
//
//	import (
//	    "github.com/totvs/go-sdk/log"
//	    "github.com/totvs/go-sdk/log/adapter"
//	)
//
//	lg := adapter.NewDefaultLog()
//	log.SetGlobal(lg)
//	log.Info().Str("key", "value").Msg("hello world")
//
// Key concepts:
//   - LoggerFacade: Main interface for creating loggers with fields
//   - LogEvent: Fluent interface for building individual log entries
//   - Global logger: Thread-safe package-level logger via SetGlobal/GetGlobal
//   - Context injection: Store/retrieve loggers via ContextWithLogger/FromContext
//
// Thread safety: All operations are safe for concurrent use. The global logger
// uses sync/atomic.Value internally.
package log

import (
	"context"
	"sync/atomic"
)

// LogEvent is the fluent interface for building chained log entries.
// It mirrors zerolog.Event without exposing zerolog directly to consumers.
type LogEvent interface {
	// Str adds a string field.
	Str(k, v string) LogEvent
	// Int adds an int field.
	Int(k string, v int) LogEvent
	// Int64 adds an int64 field.
	Int64(k string, v int64) LogEvent
	// Uint adds a uint field.
	Uint(k string, v uint) LogEvent
	// Uint64 adds a uint64 field.
	Uint64(k string, v uint64) LogEvent
	// Bool adds a boolean field.
	Bool(k string, v bool) LogEvent
	// Float32 adds a float32 field.
	Float32(k string, v float32) LogEvent
	// Float64 adds a float64 field.
	Float64(k string, v float64) LogEvent
	// Interface adds a field with any type (uses reflection).
	Interface(k string, v interface{}) LogEvent
	// Err adds an error field.
	Err(err error) LogEvent
	// Msg emits the log entry with the given message.
	Msg(msg string)
	// Msgf emits the log entry with a formatted message.
	Msgf(format string, args ...interface{})
	// Write implements io.Writer for use with standard log package.
	Write(p []byte) (int, error)
}

// LoggerFacade is the public abstraction for logging used by applications.
// Implementations can use zerolog (via adapter) or other libraries.
// All methods are safe for concurrent use.
type LoggerFacade interface {
	// WithField returns a new logger with an additional field.
	WithField(k string, v interface{}) LoggerFacade
	// WithFields returns a new logger with multiple additional fields.
	WithFields(fields map[string]interface{}) LoggerFacade
	// WithTraceFromContext returns a new logger with trace_id from context.
	WithTraceFromContext(ctx context.Context) LoggerFacade

	// Debug returns a LogEvent for debug-level logging.
	Debug() LogEvent
	// Info returns a LogEvent for info-level logging.
	Info() LogEvent
	// Warn returns a LogEvent for warn-level logging.
	Warn() LogEvent
	// Error returns a LogEvent for error-level logging.
	// If err is non-nil, it will be included as the "error" field.
	Error(err error) LogEvent
}

// Note: concrete implementations live in log/adapter and log/internal/backend.
// The package provides a no-op fallback when no global logger is configured.

// Level represents a logging level independent from concrete implementations
// so callers don't need to import zerolog directly.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// ctxKey is used for storing values in context without colliding with other packages.
type ctxKey string

const loggerKey ctxKey = "logger"

// Note: trace-related helpers (ContextWithTrace, TraceIDFromContext,
// GenerateTraceID, etc.) live in the `trace` package to keep tracing and
// propagation concerns separated from the logging facade.

// globalLogger stores the package-level logger in an atomic.Value to make
// reads/writes safe for concurrent access. Use SetGlobal/GetGlobal to access.
var globalLogger atomic.Value

// storedLogger is a stable concrete type stored in the atomic.Value. We keep
// a pointer to this type so we always store the same concrete type, avoiding
// atomic.Value panics when swapping implementations at runtime.
type storedLogger struct{ lf LoggerFacade }

// SetGlobal replaces the package-level global logger.
func SetGlobal(l LoggerFacade) { globalLogger.Store(&storedLogger{lf: l}) }

// nop implementations used as safe fallbacks when no global logger is set.
type nopEvent struct{}

func (nopEvent) Str(k, v string) LogEvent                   { return nopEvent{} }
func (nopEvent) Int(k string, v int) LogEvent               { return nopEvent{} }
func (nopEvent) Int64(k string, v int64) LogEvent           { return nopEvent{} }
func (nopEvent) Uint(k string, v uint) LogEvent             { return nopEvent{} }
func (nopEvent) Uint64(k string, v uint64) LogEvent         { return nopEvent{} }
func (nopEvent) Bool(k string, v bool) LogEvent             { return nopEvent{} }
func (nopEvent) Float32(k string, v float32) LogEvent       { return nopEvent{} }
func (nopEvent) Float64(k string, v float64) LogEvent       { return nopEvent{} }
func (nopEvent) Interface(k string, v interface{}) LogEvent { return nopEvent{} }
func (nopEvent) Err(err error) LogEvent                     { return nopEvent{} }
func (nopEvent) Msg(msg string)                             {}
func (nopEvent) Msgf(format string, args ...interface{})    {}
func (nopEvent) Write(p []byte) (n int, err error)          { return 0, nil }

type nopLogger struct{}

func (nopLogger) WithField(k string, v interface{}) LoggerFacade        { return nopLogger{} }
func (nopLogger) WithFields(fields map[string]interface{}) LoggerFacade { return nopLogger{} }
func (nopLogger) WithTraceFromContext(ctx context.Context) LoggerFacade { return nopLogger{} }
func (nopLogger) Debug() LogEvent                                       { return nopEvent{} }
func (nopLogger) Info() LogEvent                                        { return nopEvent{} }
func (nopLogger) Warn() LogEvent                                        { return nopEvent{} }
func (nopLogger) Error(err error) LogEvent                              { return nopEvent{} }

// GetGlobal returns the package-level global logger. If none is configured
// yet, it stores and returns a no-op logger to avoid nil panics.
func GetGlobal() LoggerFacade {
	if v := globalLogger.Load(); v != nil {
		if sl, ok := v.(*storedLogger); ok {
			if sl.lf != nil {
				return sl.lf
			}
		}
	}
	def := &storedLogger{lf: nopLogger{}}
	globalLogger.Store(def)
	return def.lf
}

// Package-level helpers for fluent events
func Debug() LogEvent          { return GetGlobal().Debug() }
func Info() LogEvent           { return GetGlobal().Info() }
func Warn() LogEvent           { return GetGlobal().Warn() }
func Error(err error) LogEvent { return GetGlobal().Error(err) }

// ContextWithLogger stores a LoggerFacade in the context so callers can inject
// a logger instance (facade) that will be used by library code.
func ContextWithLogger(ctx context.Context, l LoggerFacade) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFromContext extracts a LoggerFacade from the context. The boolean indicates whether a logger was present.
// The context value must be a LoggerFacade
func LoggerFromContext(ctx context.Context) (LoggerFacade, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(loggerKey); v != nil {
		if lf, ok := v.(LoggerFacade); ok {
			return lf, true
		}
	}
	return nil, false
}

// FromContext returns a LoggerFacade extracted from the context if present; otherwise returns the global logger.
func FromContext(ctx context.Context) LoggerFacade {
	if lf, ok := LoggerFromContext(ctx); ok {
		return lf
	}
	return GetGlobal()
}
