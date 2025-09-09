package logger

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

// LoggerFacade é a abstração pública para logging usada pela aplicação.
// Implementações podem usar zerolog (via o adaptador abaixo) ou qualquer
// outra biblioteca no futuro.
type LoggerFacade interface {
	WithField(k string, v interface{}) LoggerFacade
	WithFields(fields map[string]interface{}) LoggerFacade
	WithTraceFromContext(ctx context.Context) LoggerFacade

	Info(msg string)
	Debug(msg string)
	Warn(msg string)
	// Error logs a message and accepts an optional error (pass nil if none).
	Error(msg string, err error)

	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	// Errf logs a formatted message and an error.
	Errf(format string, err error, args ...interface{})
	// Errorw logs a message with an error and additional structured fields.
	Errorw(msg string, err error, fields map[string]interface{})
}

// zerologAdapter adapta o Logger concreto (baseado em zerolog) para a
// interface LoggerFacade.
type zerologAdapter struct{ l loggerImpl }

func (z zerologAdapter) WithField(k string, v interface{}) LoggerFacade {
	return zerologAdapter{l: z.l.withField(k, v)}
}

func (z zerologAdapter) WithFields(fields map[string]interface{}) LoggerFacade {
	return zerologAdapter{l: z.l.withFields(fields)}
}

func (z zerologAdapter) WithTraceFromContext(ctx context.Context) LoggerFacade {
	return zerologAdapter{l: WithTraceFromContext(ctx, z.l)}
}

func (z zerologAdapter) Info(msg string)  { z.l.InfoMsg(msg) }
func (z zerologAdapter) Debug(msg string) { z.l.DebugMsg(msg) }
func (z zerologAdapter) Warn(msg string)  { z.l.WarnMsg(msg) }
func (z zerologAdapter) Error(msg string, err error) {
	if err != nil {
		z.l.l.Error().Err(err).Msg(msg)
		return
	}
	z.l.ErrorMsg(msg)
}

func (z zerologAdapter) Infof(format string, args ...interface{}) {
	z.Info(fmt.Sprintf(format, args...))
}
func (z zerologAdapter) Debugf(format string, args ...interface{}) {
	z.Debug(fmt.Sprintf(format, args...))
}
func (z zerologAdapter) Warnf(format string, args ...interface{}) {
	z.Warn(fmt.Sprintf(format, args...))
}
func (z zerologAdapter) Errorf(format string, args ...interface{}) {
	z.Error(fmt.Sprintf(format, args...), nil)
}

func (z zerologAdapter) Errf(format string, err error, args ...interface{}) {
	z.Errorw(fmt.Sprintf(format, args...), err, nil)
}

func (z zerologAdapter) Errorw(msg string, err error, fields map[string]interface{}) {
	if fields == nil {
		// simple case: just log with error
		z.Error(msg, err)
		return
	}
	// apply fields then log error or message
	tmp := zerologAdapter{l: z.l.withFields(fields)}
	if err != nil {
		tmp.l.l.Error().Err(err).Msg(msg)
		return
	}
	tmp.l.ErrorMsg(msg)
}

// NewFacade cria uma nova instância de LoggerFacade baseada no zerolog
// existente. Use esta função para obter uma instância específica.
func NewFacade(w io.Writer, level Level) LoggerFacade { return zerologAdapter{l: newLogger(w, level)} }

// NewDefaultFacade creates a LoggerFacade with default settings (stdout/LOG_LEVEL).
func NewDefaultFacade() LoggerFacade { return zerologAdapter{l: newDefaultLogger()} }

// globalLogger stores the package-level logger in an atomic.Value to make
// reads/writes safe for concurrent access. Use SetGlobal/GetGlobal to access.
var globalLogger atomic.Value

func init() {
	globalLogger.Store(NewDefaultFacade())
}

// SetGlobal replaces the package-level global logger.
func SetGlobal(l LoggerFacade) { globalLogger.Store(l) }

// GetGlobal returns the package-level global logger.
func GetGlobal() LoggerFacade {
	if v := globalLogger.Load(); v != nil {
		if lf, ok := v.(LoggerFacade); ok {
			return lf
		}
	}
	// fallback: store and return a default facade
	def := NewDefaultFacade()
	globalLogger.Store(def)
	return def
}

// Atalho de pacote para usar o logger global.
func Info(msg string)             { GetGlobal().Info(msg) }
func Debug(msg string)            { GetGlobal().Debug(msg) }
func Warn(msg string)             { GetGlobal().Warn(msg) }
func Error(msg string, err error) { GetGlobal().Error(msg, err) }

func Infof(format string, args ...interface{})           { GetGlobal().Infof(format, args...) }
func Debugf(format string, args ...interface{})          { GetGlobal().Debugf(format, args...) }
func Warnf(format string, args ...interface{})           { GetGlobal().Warnf(format, args...) }
func Errorf(format string, args ...interface{})          { GetGlobal().Errorf(format, args...) }
func Errf(format string, err error, args ...interface{}) { GetGlobal().Errf(format, err, args...) }
func Errorw(msg string, err error, fields map[string]interface{}) {
	GetGlobal().Errorw(msg, err, fields)
}

// NOTE: prefer using `GetGlobal()` to access the global facade and call
// `WithField`/`WithFields` on it. Deliberately avoid adding global
// `WithField`/`WithFields` shortcuts to keep the API explicit.

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

// FromContextFacade returns a LoggerFacade extracted from the context if present; otherwise returns the global logger.
func FromContextFacade(ctx context.Context) LoggerFacade {
	if lf, ok := LoggerFromContext(ctx); ok {
		return lf
	}
	return GetGlobal()
}

// GetLoggerFromRequest is a convenience helper for HTTP handlers.
// It returns a LoggerFacade extracted from the request context when present,
// otherwise returns the global logger. The second return value indicates
// whether the middleware already logged the request (so handlers can avoid
// duplicating the same request-level message).
func GetLoggerFromRequest(r *http.Request) (LoggerFacade, bool) {
	if r == nil {
		return GetGlobal(), false
	}
	if lf, ok := LoggerFromContext(r.Context()); ok {
		return lf, LoggedFromContext(r.Context())
	}
	return GetGlobal(), false
}
