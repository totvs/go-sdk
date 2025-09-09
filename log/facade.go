package logger

import (
	"context"
	"fmt"
	"io"
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
    // Error accepts an optional error parameter. Pass an error to include it
    // in the log payload (as the `error` field).
    Error(msg string, err ...error)

	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
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
func (z zerologAdapter) Error(msg string, err ...error) {
    if len(err) > 0 && err[0] != nil {
        z.l.l.Error().Err(err[0]).Msg(msg)
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
	z.Error(fmt.Sprintf(format, args...))
}

// NewFacade cria uma nova instância de LoggerFacade baseada no zerolog
// existente. Use esta função para obter uma instância específica.
func NewFacade(w io.Writer, level Level) LoggerFacade { return zerologAdapter{l: newLogger(w, level)} }

// NewDefaultFacade creates a LoggerFacade with default settings (stdout/LOG_LEVEL).
func NewDefaultFacade() LoggerFacade { return zerologAdapter{l: newDefaultLogger()} }

// global é o logger usado pelas funções de atalho do pacote. Pode ser
// substituído com SetGlobal para usar outra implementação ou uma instância.
var global LoggerFacade = NewDefaultFacade()

// SetGlobal substitui o logger global usado pelas funções de atalho.
func SetGlobal(l LoggerFacade) { global = l }

// GetGlobal retorna o logger global atual.
func GetGlobal() LoggerFacade { return global }

// Atalho de pacote para usar o logger global.
func Info(msg string)  { global.Info(msg) }
func Debug(msg string) { global.Debug(msg) }
func Warn(msg string)  { global.Warn(msg) }
func Error(msg string, err ...error) { global.Error(msg, err...) }

func Infof(format string, args ...interface{})  { global.Infof(format, args...) }
func Debugf(format string, args ...interface{}) { global.Debugf(format, args...) }
func Warnf(format string, args ...interface{})  { global.Warnf(format, args...) }
func Errorf(format string, args ...interface{}) { global.Errorf(format, args...) }

func WithField(k string, v interface{}) LoggerFacade { return global.WithField(k, v) }

// WithFieldsGlobal é um atalho que aplica os campos ao logger global e
// retorna um novo LoggerFacade.
func WithFieldsGlobal(fields map[string]interface{}) LoggerFacade { return global.WithFields(fields) }

// ContextWithLogger stores a LoggerFacade in the context so callers can inject
// a logger instance (facade) that will be used by library code.
func ContextWithLogger(ctx context.Context, l LoggerFacade) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// LoggerFromContext extracts a LoggerFacade from the context. The boolean indicates whether a logger was present.
// For backward compatibility, if a concrete Logger is stored in the context it will be wrapped.
func LoggerFromContext(ctx context.Context) (LoggerFacade, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(loggerKey); v != nil {
		if lf, ok := v.(LoggerFacade); ok {
			return lf, true
		}
		if l, ok := v.(loggerImpl); ok {
			return zerologAdapter{l: l}, true
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
