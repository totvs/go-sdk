# Logging (log)

Este módulo fornece um utilitário de logging baseado em `zerolog`, com saída em JSON e suporte a `trace_id`.

Estrutura:
- `logger.go` — implementação pública do logger no pacote `logger`.
- `internal/` — (opcional) helpers privados.
- `cmd/example` — exemplo executável.

Como usar:

1. No repositório que consome o módulo, adicione a dependência:
   `go get github.com/totvs/go-sdk/log@v0.0.0` (ou use `replace` localmente durante desenvolvimento)

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

Exemplo executável:

Execute o exemplo localmente:
```
go run ./cmd/example
```

Dicas:
- Ajuste o nível de log via `LOG_LEVEL` (ex.: `DEBUG`, `INFO`).
- Publique tags para versionamento do módulo: `git tag log/v0.1.0` e `git push origin log/v0.1.0`.

