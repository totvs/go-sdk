package util

import (
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	lg "github.com/totvs/go-sdk/log"
)

// GinWriter encaminha escritas (io.Writer) do Gin para uma LoggerFacade.
// Cada linha escrita é transformada em um evento de log (Info ou Error).
type GinWriter struct {
	lf      lg.LoggerFacade
	isError bool
}

// NewGinWriter cria um io.Writer que encaminha mensagens para o LoggerFacade.
// Quando isError for true, as mensagens são logadas com nível Error, caso contrário Info.
func NewGinWriter(lf lg.LoggerFacade, isError bool) io.Writer {
	return &GinWriter{lf: lf, isError: isError}
}

func (g *GinWriter) Write(p []byte) (n int, err error) {
	if g == nil || g.lf == nil {
		return len(p), nil
	}
	s := string(p)
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return len(p), nil
	}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		msg := strings.TrimRight(line, "\r")
		if msg == "" {
			continue
		}
		if g.isError {
			g.lf.WithField("component", "gin").WithField("stream", "stderr").Error(nil).Msg(msg)
		} else {
			g.lf.WithField("component", "gin").WithField("stream", "stdout").Info().Msg(msg)
		}
	}
	return len(p), nil
}

// ConfigureGinDefaultWriters substitui os writers padrão do Gin
// (`gin.DefaultWriter` e `gin.DefaultErrorWriter`) por writers que
// encaminham as mensagens para a LoggerFacade fornecida. Retorna os
// writers antigos para que possam ser restaurados.
func ConfigureGinDefaultWriters(lf lg.LoggerFacade) (oldOut io.Writer, oldErr io.Writer) {
	oldOut = gin.DefaultWriter
	oldErr = gin.DefaultErrorWriter
	gin.DefaultWriter = NewGinWriter(lf, false)
	gin.DefaultErrorWriter = NewGinWriter(lf, true)
	return oldOut, oldErr
}

// RestoreGinDefaultWriters restaura os writers anteriores do Gin.
func RestoreGinDefaultWriters(oldOut io.Writer, oldErr io.Writer) {
	if oldOut != nil {
		gin.DefaultWriter = oldOut
	}
	if oldErr != nil {
		gin.DefaultErrorWriter = oldErr
	}
}
