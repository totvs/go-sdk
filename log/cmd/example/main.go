package main

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"

    "github.com/rs/zerolog"
    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // cria logger JSON para stdout
    l := logger.New(os.Stdout, zerolog.InfoLevel)

    // anexa trace id ao contexto
    ctx := logger.ContextWithTrace(context.Background(), "example-trace-0001")

    // obtém logger com o campo `trace_id` quando presente no contexto
    l = logger.WithTraceFromContext(ctx, l)
    l.Info().Str("service", "log-example").Msg("starting example logger")

    // demonstra uso em um handler sem iniciar um servidor de verdade
    req := httptest.NewRequest("GET", "http://example.local/", nil)
    req = req.WithContext(ctx)

    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.WithTraceFromContext(r.Context(), l).Info().Msg("handler invoked")
        w.WriteHeader(200)
        w.Write([]byte("ok"))
    })

    handler.ServeHTTP(httptest.NewRecorder(), req)
    fmt.Fprintln(os.Stdout, "example run completed")
}

