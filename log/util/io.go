package util

import (
	"io"
	"strings"

	lg "github.com/totvs/go-sdk/log"
)

// LogIOWriter encaminha escritas (io.Writer) do Gin para uma LoggerFacade.
// Cada linha escrita é transformada em um evento de log (Info ou Error).
type LogIOWriter struct {
	lf      lg.LoggerFacade
	isError bool
}

// NewLogIOWriter cria um io.Writer que encaminha mensagens para o LoggerFacade.
// Quando isError for true, as mensagens são logadas com nível Error, caso contrário Info.
func NewLogIOWriter(lf lg.LoggerFacade, isError bool) io.Writer {
	return &LogIOWriter{lf: lf, isError: isError}
}

func (g *LogIOWriter) Write(p []byte) (n int, err error) {
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
			g.lf.Error(nil).Msg(msg)
		} else {
			g.lf.Info().Msg(msg)
		}
	}
	return len(p), nil
}
