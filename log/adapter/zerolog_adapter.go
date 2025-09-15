package adapter

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	lg "github.com/totvs/go-sdk/log"
)

// zerolog-backed adapter in its own package. It implements lg.LoggerFacade
// and returns lg.LogEvent instances so callers interact with the facade types.
type zerologAdapter struct{ l zerolog.Logger }

func (z zerologAdapter) WithField(k string, v interface{}) lg.LoggerFacade {
	return zerologAdapter{l: z.l.With().Interface(k, v).Logger()}
}

func (z zerologAdapter) WithFields(fields map[string]interface{}) lg.LoggerFacade {
	c := z.l.With()
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
	return zerologAdapter{l: c.Logger()}
}

func (z zerologAdapter) WithTraceFromContext(ctx context.Context) lg.LoggerFacade {
	if tid := lg.TraceIDFromContext(ctx); tid != "" {
		return zerologAdapter{l: z.l.With().Str(lg.TraceIDField, tid).Logger()}
	}
	return z
}

// adapter LogEvent backed by zerolog.Event
type zerologEvent struct{ e *zerolog.Event }

func newZerologEvent(e *zerolog.Event) lg.LogEvent { return &zerologEvent{e: e} }

func (z *zerologEvent) Str(k, v string) lg.LogEvent             { z.e = z.e.Str(k, v); return z }
func (z *zerologEvent) Int(k string, v int) lg.LogEvent         { z.e = z.e.Int(k, v); return z }
func (z *zerologEvent) Int64(k string, v int64) lg.LogEvent     { z.e = z.e.Int64(k, v); return z }
func (z *zerologEvent) Uint(k string, v uint) lg.LogEvent       { z.e = z.e.Uint(k, v); return z }
func (z *zerologEvent) Uint64(k string, v uint64) lg.LogEvent   { z.e = z.e.Uint64(k, v); return z }
func (z *zerologEvent) Bool(k string, v bool) lg.LogEvent       { z.e = z.e.Bool(k, v); return z }
func (z *zerologEvent) Float32(k string, v float32) lg.LogEvent { z.e = z.e.Float32(k, v); return z }
func (z *zerologEvent) Float64(k string, v float64) lg.LogEvent { z.e = z.e.Float64(k, v); return z }
func (z *zerologEvent) Interface(k string, v interface{}) lg.LogEvent {
	z.e = z.e.Interface(k, v)
	return z
}
func (z *zerologEvent) Err(err error) lg.LogEvent { z.e = z.e.Err(err); return z }
func (z *zerologEvent) Msg(msg string)            { z.e.Msg(msg) }
func (z *zerologEvent) Msgf(format string, args ...interface{}) {
	z.e.Msgf(format, args...)
}

func (z zerologAdapter) Debug() lg.LogEvent { return newZerologEvent(z.l.Debug()) }
func (z zerologAdapter) Info() lg.LogEvent  { return newZerologEvent(z.l.Info()) }
func (z zerologAdapter) Warn() lg.LogEvent  { return newZerologEvent(z.l.Warn()) }
func (z zerologAdapter) Error(err error) lg.LogEvent {
	ev := z.l.Error()
	if err != nil {
		ev = ev.Err(err)
	}
	return newZerologEvent(ev)
}

// NewLog cria um LoggerFacade baseado em zerolog que escreve em `w` com o nível informado.
func NewLog(w io.Writer, level lg.Level) lg.LoggerFacade {
	zerolog.TimeFieldFormat = time.RFC3339
	var zlvl zerolog.Level
	switch level {
	case lg.DebugLevel:
		zlvl = zerolog.DebugLevel
	case lg.InfoLevel:
		zlvl = zerolog.InfoLevel
	case lg.WarnLevel:
		zlvl = zerolog.WarnLevel
	case lg.ErrorLevel:
		zlvl = zerolog.ErrorLevel
	default:
		zlvl = zerolog.InfoLevel
	}
	lgz := zerolog.New(w).With().Timestamp().Logger().Level(zlvl)
	return zerologAdapter{l: lgz}
}

// NewDefaultLog cria um adapter zerolog com configurações padrão (stdout, LOG_LEVEL).
func NewDefaultLog() lg.LoggerFacade {
	lvl := lg.InfoLevel
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		switch s {
		case "DEBUG", "debug":
			lvl = lg.DebugLevel
		case "INFO", "info":
			lvl = lg.InfoLevel
		case "WARN", "warn", "WARNING", "warning":
			lvl = lg.WarnLevel
		case "ERROR", "error":
			lvl = lg.ErrorLevel
		}
	}
	return NewLog(os.Stdout, lvl)
}
