package main

import (
	"context"
	"net/http"
	"os"

	logger "github.com/totvs/go-sdk/log"
	adapter "github.com/totvs/go-sdk/log/adapter"
	middleware "github.com/totvs/go-sdk/log/middleware/http"
	"github.com/totvs/go-sdk/transaction"
)

// startAppLogger cria um logger de aplicação com trace e emite a mensagem de inicialização.
func startAppLogger() {
	l := adapter.NewDefaultLog()
	ctx := transaction.ContextWithTrace(context.Background(), "trace-1234")
	l = l.WithTraceFromContext(ctx)
	l.Info().Msg("application started (facade)")
}

// setGlobalLogger registra o logger como global e escreve uma mensagem via atalhos do pacote.
func setGlobalLogger() {
	l := adapter.NewDefaultLog()
	logger.SetGlobal(l)
	logger.Info().Msg("using global logger")
}

// injectedLoggerExample demonstra injeção/extracao de logger via contexto.
func injectedLoggerExample() {
	l := adapter.NewLog(os.Stdout, logger.DebugLevel)
	ctx := logger.ContextWithLogger(context.Background(), l)
	lg := logger.FromContext(ctx)
	lg.Info().Msg("using injected logger via facade")
}

// withFieldsExample adiciona campos e emite algumas mensagens de exemplo.
func withFieldsExample() {
	l := adapter.NewLog(os.Stdout, logger.InfoLevel)
	f := l.WithFields(map[string]interface{}{"service": "orders", "version": 3})
	f.Info().Msg("request processed1")
	f.Info().Msg("request processed2")
}

// packageLevelFieldsExamples demonstra uso via logger global.
func packageLevelFieldsExamples() {
	l := adapter.NewDefaultLog()
	logger.SetGlobal(l)
	logger.GetGlobal().WithFields(map[string]interface{}{"app": "example", "uptime": "1m"}).Info().Msg("global infow example")
	logger.GetGlobal().WithFields(map[string]interface{}{"detail": "verbose info"}).Debug().Msg("global debugw example")
	logger.GetGlobal().WithFields(map[string]interface{}{"disk": "low"}).Warn().Msg("global warnw example")
}

// chainedFluentExample demonstra o estilo encadeado (fluente).
func chainedFluentExample() {
	l := adapter.NewLog(os.Stdout, logger.DebugLevel)
	l.Debug().Str("Scale", "833 cents").Float64("Interval", 833.09).Msg("Fibonacci is everywhere 1")
	// também via helper de pacote
	logger.Debug().Str("Scale", "833 cents").Float64("Interval", 833.09).Msg("Fibonacci is everywhere 2")
}

// httpServerExample inicia um servidor HTTP simples que usa o middleware de logging.
func httpServerExample() {
	appLogger := adapter.NewDefaultLog()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		lf, logged := middleware.GetLoggerFromRequest(r)
		if !logged {
			lf.Info().Msg("handler received request")
		}
		w.Write([]byte("ok"))
	})

	// listen on :8080 — passa o logger da aplicação para o middleware
	err := http.ListenAndServe(":8080", middleware.HTTPMiddlewareWithLogger(appLogger)(mux))
	if err != nil {
		appLogger.Error(err).Msg("failed to start server")
	}
}

func main() {
	startAppLogger()
	setGlobalLogger()
	injectedLoggerExample()
	withFieldsExample()
	packageLevelFieldsExamples()
	chainedFluentExample()
	httpServerExample()
}
