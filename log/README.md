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
the client does not provide `X-Request-Id` or `X-Correlation-Id`. The generated id is:

- Inserido no contexto da requisição (`ContextWithTrace`).
- Adicionado ao log como campo `trace_id`.
- Inserido no header de resposta `X-Request-Id` (se o header ainda estiver ausente), facilitando correlação.

Exemplo rápido:

```go
mux := http.NewServeMux()
// ... registre handlers ...
http.ListenAndServe(":8080", logger.HTTPMiddleware(mux)) // usa logger padrão
// ou com logger customizado:
// http.ListenAndServe(":8080", logger.HTTPMiddlewareWithLogger(myLogger)(mux))
```

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
```

<!-- exemplo executável removido -->

Dicas:
- Ajuste o nível de log via `LOG_LEVEL` (ex.: `DEBUG`, `INFO`).
- Publique tags para versionamento do repositório: `git tag v0.1.0` e `git push origin v0.1.0`.
