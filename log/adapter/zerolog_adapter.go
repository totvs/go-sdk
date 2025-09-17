package adapter

import (
	"io"

	lg "github.com/totvs/go-sdk/log"
	backend "github.com/totvs/go-sdk/log/internal/backend"
)

// NewLog delegates to the internal zerolog backend.
func NewLog(w io.Writer, level lg.Level) lg.LoggerFacade { return backend.NewLog(w, level) }

// NewDefaultLog delegates to the internal zerolog backend.
func NewDefaultLog() lg.LoggerFacade { return backend.NewDefaultLog() }
