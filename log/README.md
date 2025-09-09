# Logging (log)

Este módulo fornece um utilitário de logging baseado em `zerolog`, com saída em JSON e suporte a `trace_id`.

Estrutura:
- `logger.go` — implementação pública do logger no pacote `logger`.
- `internal/` — (opcional) helpers privados.


Como usar:

1. No repositório que consome o pacote, adicione a dependência:
   `go get github.com/totvs/go-sdk@v0.0.0` (ou use `replace github.com/totvs/go-sdk => /caminho/para/repositorio` localmente durante desenvolvimento)

2. Exemplo de uso básico:

```go
import (
    "context"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // use the facade API (decoupled from the concrete implementation)
    f := logger.NewFacade(os.Stdout, logger.InfoLevel)
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
    f = f.WithTraceFromContext(ctx)
    f.Info("aplicação iniciada")

    // definir logger global para usar atalhos do pacote
    logger.SetGlobal(f)
    logger.Info("mensagem via logger global")
}
```

Novos helpers e middleware

- `WithField(key, value)` — adiciona um único campo ao logger de forma conveniente.
- `WithFields(map[string]interface{})` — método equivalente à função de pacote para adicionar múltiplos campos.
-- `Info`, `Debug`, `Warn`, `Error` on the facade — helpers that emit a simple message (`f.Info("...")`).

Middleware HTTP

`HTTPMiddlewareWithLogger(base LoggerFacade)` accepts a logger facade and will generate a secure `trace id` when
the client does not provide `X-Request-Id` or `X-Correlation-Id`.

Behavior

- The middleware inserts the trace id into the request context via `ContextWithTrace`.
- The middleware adds `trace_id` to the emitted request-level log entry.
- The middleware adds the `X-Request-Id` header to the response when absent (default behavior).

Configuration

The middleware is configurable via `MiddlewareOptions`:

- `LogRequest bool` — when true the middleware emits a request-level log (default: true).
- `InjectLogger bool` — when true the middleware injects the facade logger into the request context so handlers can reuse it (default: true).
- `AddTraceHeader bool` — when true the middleware sets `X-Request-Id` on the response when absent (default: true).

Use `HTTPMiddlewareWithOptions(base, opts)` to customize behavior. The convenience
`HTTPMiddlewareWithLogger(base)` uses sensible defaults (all `true`).

Example: disable request logging but still inject the logger

```go
opts := logger.MiddlewareOptions{LogRequest: false, InjectLogger: true, AddTraceHeader: true}
http.ListenAndServe(":8080", logger.HTTPMiddlewareWithOptions(myLogger, opts)(mux))
```

Note: the middleware will mark the request context when it already logged the request. Handlers that also log
should check `logger.LoggedFromContext(r.Context())` to avoid duplicating the same request-level message.

Exemplo completo e saída esperada

```go
package main

import (
    "net/http"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    l := logger.NewFacade(os.Stdout, logger.InfoLevel)

    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    // start server with middleware (will generate trace id if missing)
    http.ListenAndServe(":8080", logger.HTTPMiddlewareWithLogger(l)(mux))
}
```

Uma chamada GET para `/ping` sem `X-Request-Id` pode gerar uma linha de log JSON parecida com:

```json
{
  "level": "info",
  "time": "2025-09-05T12:00:00Z",
  "trace_id": "9f3b2c1a4d5e6f708192a3b4c5d6e7f8",
  "method": "GET",
  "path": "/ping",
  "message": "http request received"
}
```

Injecting logger into context (facade)

```go
// create a facade and store it in the context so library code can use it
lg := logger.NewFacade(os.Stdout, zerolog.DebugLevel)
ctx := logger.ContextWithLogger(context.Background(), lg)

// later, library code extracts a facade from the context and uses it
f := logger.FromContextFacade(ctx)
f.Info("using injected logger via facade")

// adding multiple fields conveniently via the facade
f3 := f.WithFields(map[string]interface{}{"service": "orders", "version": 3})
f3.Info("request processed")

// Error logging examples
f.Error("operation failed", nil) // simple errorless message
f.Error("operation failed", errors.New("boom")) // include an error
f.Errf("failed to %s", errors.New("boom"), "start") // formatted message with error
f.Errorw("failed to start", errors.New("boom"), map[string]interface{}{"service": "orders"}) // error + fields
```

<!-- exemplo executável removido -->

Dicas:
- Ajuste o nível de log via `LOG_LEVEL` (ex.: `DEBUG`, `INFO`).
- Publique tags para versionamento do repositório: `git tag v0.1.0` e `git push origin v0.1.0`.
