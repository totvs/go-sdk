# Logging (pkg/log)

Este documento descreve como usar o utilitário de logging fornecido em `pkg/log`.

Características principais:

- Baseado em `zerolog` e produz logs em JSON por padrão.
- Suporte a `trace_id` via `context.Context` para rastreabilidade distribuída.
- Middleware HTTP que injeta `trace_id` a partir dos headers `X-Request-Id` ou `X-Correlation-Id`.
- Funções utilitárias: `New`, `NewDefault`, `ContextWithTrace`, `TraceIDFromContext`, `WithTraceFromContext`.

Exemplo básico (uso direto) — coloque este código no seu pacote ou nos readmes dos seus componentes:

```go
import (
    "context"
    "os"

    "github.com/rs/zerolog"
    logger "github.com/totvs/go-sdk/pkg/log"
)

func example() {
    // cria logger JSON para stdout
    l := logger.New(os.Stdout, zerolog.InfoLevel)

    // anexa trace id ao contexto
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")

    // obtém logger com o campo `trace_id` quando presente no contexto
    l = logger.WithTraceFromContext(ctx, l)
    l.Info().Msg("aplicação iniciada")
}
```


Exemplo de middleware HTTP (documentação):

```go
// no seu servidor HTTP, use:
mux := http.NewServeMux()
// ... registre handlers ...
http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))

// O middleware procura `X-Request-Id` ou `X-Correlation-Id` e injeta no contexto.
```

Exemplo executável:

Um exemplo executável também está disponível em `pkg/log/cmd/example`. Execute com:

```bash
go run ./pkg/log/cmd/example
```


Dicas:

- Para ajustar o nível de log via variável de ambiente, defina `LOG_LEVEL` (por exemplo, `DEBUG`, `INFO`).
- Nome do campo `trace_id` segue o padrão do repositório para facilitar correlação entre serviços.
