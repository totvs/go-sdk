package util

import (
	"github.com/go-logr/logr"
	lg "github.com/totvs/go-sdk/log"
	"github.com/totvs/go-sdk/log/adapter"
)

// NewLogrAdapter cria um `logr.Logger` que delega para a LoggerFacade
// fornecida.
func NewLogrAdapter(lf lg.LoggerFacade) logr.Logger {
	return adapter.NewLogrAdapter(lf)
}

// NewGlobalLogr retorna um logr.Logger que usa o logger global do pacote `log`.
func NewGlobalLogr() logr.Logger { return adapter.NewGlobalLogr() }
