package main

import (
	"context"
	"net/http"
	"os"

	logger "github.com/totvs/go-sdk/log"
)

func main() {
	// usando a fachada (abstração)
	myAppInstanceLogger1 := logger.NewFacade(os.Stdout, logger.InfoLevel)
	ctx1 := logger.ContextWithTrace(context.Background(), "trace-1234")
	myAppInstanceLogger1 = myAppInstanceLogger1.WithTraceFromContext(ctx1)
	myAppInstanceLogger1.Info("application started (facade)")

	// definir como logger global para usar atalhos do pacote
	logger.SetGlobal(myAppInstanceLogger1)
	logger.Info("using global logger")

	// ainda é possível injetar uma facade no contexto e extrair ela depois
	myAppInstanceLogger2 := logger.NewFacade(os.Stdout, logger.DebugLevel)
	ctx2 := logger.ContextWithLogger(context.Background(), myAppInstanceLogger2)
	myAppInstanceLogger3 := logger.FromContextFacade(ctx2)
	myAppInstanceLogger3.Info("using injected logger via facade")

	// adicionar campos via facade
	f3 := myAppInstanceLogger3.WithFields(map[string]interface{}{"service": "orders", "version": 3})
	f3.Info("request processed1")
	f3.Info("request processed2")

	// HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if lf, ok := logger.LoggerFromContext(r.Context()); ok {
			lf.Info("fazendo alguma coisa no handler")
		}

		w.Write([]byte("ok"))
	})

	// listen on :8080 (ctrl-c to stop) — pass the same app logger instance to the middleware
	http.ListenAndServe(":8080", logger.HTTPMiddlewareWithLogger(myAppInstanceLogger1)(mux))
}
