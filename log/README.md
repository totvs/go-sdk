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

    "github.com/rs/zerolog"
    logger "github.com/totvs/go-sdk/log"
)

func main() {
    l := logger.New(os.Stdout, zerolog.InfoLevel)
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
    l = logger.WithTraceFromContext(ctx, l)
    l.Info().Msg("aplicação iniciada")
}
```

Injecting logger into context:

```go
// create a logger and store it in the context so library code can use it
lg := logger.New(os.Stdout, zerolog.DebugLevel)
ctx := logger.ContextWithLogger(context.Background(), lg)

// later, library code can get the logger or fall back to default
lg2 := logger.FromContext(ctx)
lg2.Info().Msg("using injected logger")

// adding multiple fields conveniently
lg3 := logger.WithFields(lg2, map[string]interface{}{"service": "orders", "version": 3})
lg3.Info().Msg("request processed")
```

<!-- exemplo executável removido -->

Dicas:
- Ajuste o nível de log via `LOG_LEVEL` (ex.: `DEBUG`, `INFO`).
- Publique tags para versionamento do repositório: `git tag v0.1.0` e `git push origin v0.1.0`.
