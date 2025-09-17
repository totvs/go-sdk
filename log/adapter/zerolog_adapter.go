package adapter

import (
	"io"

	zimpl "github.com/totvs/go-sdk/internal/backends/zerolog"
	lg "github.com/totvs/go-sdk/log"
)

// NewLog delegates to the internal zerolog backend.
func NewLog(w io.Writer, level lg.Level) lg.LoggerFacade { return zimpl.NewLog(w, level) }

// NewDefaultLog delegates to the internal zerolog backend.
func NewDefaultLog() lg.LoggerFacade { return zimpl.NewDefaultLog() }
