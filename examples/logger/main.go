package main

import (
	"context"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	logger "github.com/totvs/go-sdk/log"
)

func main() {
	// basic logger
	l := logger.New(os.Stdout, zerolog.InfoLevel)
	ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
	l = logger.WithTraceFromContext(ctx, l)
	l.Info().Msg("application started")

	// inject logger into context for library code
	lg := logger.New(os.Stdout, zerolog.DebugLevel)
	ctx2 := logger.ContextWithLogger(context.Background(), lg)
	lg2 := logger.FromContext(ctx2)
	lg2.Info().Msg("using injected logger")

	// add fields
	lg3 := logger.WithFields(lg2, map[string]interface{}{"service": "orders", "version": 3})
	lg3.Info().Msg("request processed")

	// HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// listen on :8080 (ctrl-c to stop)
	http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))
}
