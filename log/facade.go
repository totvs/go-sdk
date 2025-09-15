package log

import (
	"context"
	"io"
	"sync/atomic"
)

// Public constants for trace header and field names used across projects.
const (
	// TraceIDHeader is the HTTP header used to carry the request trace id.
	TraceIDHeader = "X-Request-Id"
	// TraceIDCorrelationHeader is the alternate header name often used for correlation ids.
	TraceIDCorrelationHeader = "X-Correlation-Id"
	// TraceIDField is the JSON field name added to logs for trace ids.
	TraceIDField = "trace_id"
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
type defaultAdapter struct{ l loggerImpl }

func (d defaultAdapter) WithField(k string, v interface{}) LoggerFacade {
	return defaultAdapter{l: d.l.withField(k, v)}
}
func (d defaultAdapter) WithFields(fields map[string]interface{}) LoggerFacade {
	return defaultAdapter{l: d.l.withFields(fields)}
}
func (d defaultAdapter) WithTraceFromContext(ctx context.Context) LoggerFacade {
	return defaultAdapter{l: WithTraceFromContext(ctx, d.l)}
}
func (d defaultAdapter) Debug() LogEvent { return newZerologEvent(d.l.l.Debug()) }
func (d defaultAdapter) Info() LogEvent  { return newZerologEvent(d.l.l.Info()) }
func (d defaultAdapter) Warn() LogEvent  { return newZerologEvent(d.l.l.Warn()) }
func (d defaultAdapter) Error(err error) LogEvent {
	ev := d.l.l.Error()
	if err != nil {
		ev = ev.Err(err)
	}
	return newZerologEvent(ev)
}

// NOTE: adapter implementation (zerologAdapter) is provided in a separate
// file (zerolog_adapter.go) to keep the facade declaration independent of the
// concrete implementation.

// globalLogger stores the package-level logger in an atomic.Value to make
// reads/writes safe for concurrent access. Use SetGlobal/GetGlobal to access.
var globalLogger atomic.Value

// storedLogger is a stable concrete type stored in the atomic.Value. We keep
// a pointer to this type so we always store the same concrete type, avoiding
// atomic.Value panics when swapping implementations at runtime.
type storedLogger struct{ lf LoggerFacade }

func init() {
    // initialize with an in-package default adapter that wraps the internal loggerImpl
    globalLogger.Store(&storedLogger{lf: defaultAdapter{l: newDefaultLogger()}})
}

// SetGlobal replaces the package-level global logger.
func SetGlobal(l LoggerFacade) { globalLogger.Store(&storedLogger{lf: l}) }

// GetGlobal returns the package-level global logger.
func GetGlobal() LoggerFacade {
    if v := globalLogger.Load(); v != nil {
        if sl, ok := v.(*storedLogger); ok {
            if sl.lf != nil {
                return sl.lf
            }
        }
    }
    // fallback: store and return an in-package default facade
    def := &storedLogger{lf: defaultAdapter{l: newDefaultLogger()}}
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

// NewLog cria um LoggerFacade local (não depende do pacote adapter).
func NewLog(w io.Writer, level Level) LoggerFacade { return defaultAdapter{l: newLogger(w, level)} }

// NewDefaultLog cria uma LoggerFacade com as configurações padrão (stdout, LOG_LEVEL).
func NewDefaultLog() LoggerFacade { return defaultAdapter{l: newDefaultLogger()} }
