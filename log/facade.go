package log

import (
	"context"
	"sync/atomic"
)

// LogEvent é a interface fluente para construir logs encadeados (similar a zerolog.Event).
// Mantemos essa interface aqui para não expor zerolog diretamente aos consumidores.
type LogEvent interface {
	Str(k, v string) LogEvent
	Int(k string, v int) LogEvent
	Int64(k string, v int64) LogEvent
	Uint(k string, v uint) LogEvent
	Uint64(k string, v uint64) LogEvent
	Bool(k string, v bool) LogEvent
	Float32(k string, v float32) LogEvent
	Float64(k string, v float64) LogEvent
	Interface(k string, v interface{}) LogEvent
	Err(err error) LogEvent
	Msg(msg string)
	Msgf(format string, args ...interface{})
	Write(p []byte) (int, error)
}

// LoggerFacade é a abstração pública para logging usada pela aplicação.
// Implementações podem usar zerolog (via o adaptador abaixo) ou qualquer
// outra biblioteca no futuro.
type LoggerFacade interface {
	WithField(k string, v interface{}) LoggerFacade
	WithFields(fields map[string]interface{}) LoggerFacade
	WithTraceFromContext(ctx context.Context) LoggerFacade

	// Event builders for fluent logs. These return an Event that can be
	// chained (Str/Float64/Interface/Err/Msg...). Use these when you prefer
	// the zerolog-like fluent API.
	Debug() LogEvent
	Info() LogEvent
	Warn() LogEvent
	// Error accepts an optional error that will be attached to the event and
	// returns a LogEvent for chaining.
	Error(err error) LogEvent
}

// defaultAdapter is a minimal in-package adapter used as the package default.
// It wraps the internal `loggerImpl` so the package can initialize a sane
// default logger without depending on external adapter packages.
// NOTE: concrete implementations live in separate packages (for example
// `log/impl` or `log/adapter`). The package provides a small no-op default
// implementation used as a safe fallback when no global logger is configured.

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
