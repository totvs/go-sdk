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
    // uso direto com o tipo Logger (compatível com zerolog)
    l := logger.New(os.Stdout, logger.InfoLevel)
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
    l = logger.WithTraceFromContext(ctx, l)
    l.Info().Msg("aplicação iniciada")

    // uso através da fachada (abstração) — código consumidor fica desacoplado
    f := logger.NewFacade(os.Stdout, logger.InfoLevel)
    f = f.WithTraceFromContext(ctx)
    f.Info("aplicação iniciada (facade)")

    // definir logger global para usar atalhos do pacote
    logger.SetGlobal(f)
    logger.Info("mensagem via logger global")
}
```

Novos helpers e middleware

- `WithField(key, value)` — adiciona um único campo ao logger de forma conveniente.
- `WithFields(map[string]interface{})` — método equivalente à função de pacote para adicionar múltiplos campos.
- `InfoMsg`, `DebugMsg`, `WarnMsg`, `ErrorMsg` — helpers que emitem uma mensagem simples (`l.InfoMsg("...")`).

Middleware HTTP

O `HTTPMiddlewareWithLogger(base Logger)` agora gera automaticamente um `trace id` seguro quando nenhum header
`X-Request-Id` ou `X-Correlation-Id` é fornecido pelo cliente. O id gerado é:

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
    l := logger.New(os.Stdout, logger.InfoLevel)

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

Injecting logger into context:

```go
// create a logger and store it in the context so library code can use it
lg := logger.New(os.Stdout, zerolog.DebugLevel)
ctx := logger.ContextWithLogger(context.Background(), lg)

// later, library code can get the concrete logger or a facade wrapper
lg2 := logger.FromContext(ctx) // returns Logger
lg2.Info().Msg("using injected logger")

// if you want a facade extracted from context (to use the abstract API):
f := FromContextFacade(ctx)
f.Info("using injected logger via facade")

// adding multiple fields conveniently (concrete or facade)
lg3 := logger.WithFields(lg2, map[string]interface{}{"service": "orders", "version": 3})
lg3.Info().Msg("request processed")
```

<!-- exemplo executável removido -->

Dicas:
- Ajuste o nível de log via `LOG_LEVEL` (ex.: `DEBUG`, `INFO`).
- Publique tags para versionamento do repositório: `git tag v0.1.0` e `git push origin v0.1.0`.
