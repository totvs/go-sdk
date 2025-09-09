package main

import (
	"context"
	"net/http"
	"os"

	logger "github.com/totvs/go-sdk/log"
)

func main() {
	// usando a fachada (abstração)
	f := logger.NewFacade(os.Stdout, logger.InfoLevel)
	ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
	f = f.WithTraceFromContext(ctx)
	f.Info("application started (facade)")

	// definir como logger global para usar atalhos do pacote
	logger.SetGlobal(f)
	logger.Info("using global logger")

	// ainda é possível injetar o Logger concreto no contexto e extrair uma facade
	lg := logger.New(os.Stdout, logger.DebugLevel)
	ctx2 := logger.ContextWithLogger(context.Background(), lg)
	f2 := logger.FromContextFacade(ctx2)
	f2.Info("using injected logger via facade")

	// adicionar campos via facade
	f3 := f2.WithFields(map[string]interface{}{"service": "orders", "version": 3})
	f3.Info("request processed")

	// HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// listen on :8080 (ctrl-c to stop)
	http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))
}
