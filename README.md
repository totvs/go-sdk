# go-sdk

Projeto base (SDK) em Go com utilitários reaproveitáveis de logging, tracing
e integrações com frameworks comuns.

Visão geral
- Fachada de logging: `github.com/totvs/go-sdk/log` — API pública usada pela aplicação.
- Backend zerolog implementado internamente e exposto via `log/adapter`.
- Helpers de propagation em `github.com/totvs/go-sdk/trace`.

Estrutura principal
- `log/` — fachada pública: interfaces (`LoggerFacade`, `LogEvent`), helpers de contexto e middleware genérico.
- `log/adapter/` — adaptadores e construtores públicos (ex.: `adapter.NewLog`, `adapter.NewDefaultLog`) que retornam `LoggerFacade`.
- `log/util/` — integrações e helpers (por exemplo escritores para Gin, wrappers para klog/logr).
- `log/middleware/` — middlewares HTTP/Gin que usam a fachada e propagam `trace_id`.
- `trace/` — helpers de propagation (`ContextWithTrace`, `TraceIDFromContext`, `GenerateTraceID`).
- `internal/backends/zerolog/` — implementação concreta baseada em `zerolog` (não exportada publicamente).
- `examples/` — exemplos executáveis (ex.: `examples/logger`).

Instalação e dependências
- O módulo Go está na raiz: `module github.com/totvs/go-sdk`.
- Instale dependências e atualize o `go.mod` com:

```bash
go mod tidy
```

Uso rápido

Crie um logger (adapter) e registre como global para usar os atalhos do pacote `log`:

```go
package main

import (
    "os"

    logger "github.com/totvs/go-sdk/log"
    adapter "github.com/totvs/go-sdk/log/adapter"
)

func main() {
    lg := adapter.NewDefaultLog() // cria um LoggerFacade (zerolog por baixo)
    logger.SetGlobal(lg)
    logger.Info().Msg("aplicação iniciada")
}
```

Para construir com `io.Writer` e nível customizado:

```go
lg := adapter.NewLog(os.Stdout, logger.InfoLevel)
```

Trace / propagation
- Use o pacote `trace` para gerar/propagar `trace_id` entre handlers/middlewares:

```go
import tr "github.com/totvs/go-sdk/trace"

ctx := tr.ContextWithTrace(r.Context(), "trace-1234")
```

Middleware
- Use os middlewares em `log/middleware` para gerar `trace_id`, injetar o logger no contexto e adicionar o header `X-Request-Id` na resposta.

Exemplo (net/http):

```go
import (
    "net/http"
    middleware "github.com/totvs/go-sdk/log/middleware/http"
    adapter "github.com/totvs/go-sdk/log/adapter"
)

http.ListenAndServe(":8080", middleware.HTTPMiddlewareWithLogger(adapter.NewDefaultLog())(mux))
```

Executar exemplos
- Exemplo principal:

```bash
cd examples/logger
LOG_LEVEL=DEBUG go run .
```

Testes e CI
- Rode todos os testes com:

```bash
go test ./...
```

- CI deve executar `gofmt -w .`, `go vet ./...` e `go test ./...`.

Contribuindo
- Mantenha implementações concretas em `internal/` para não expô-las a consumidores.
- Adicione adapters em `log/adapter` para integrar bibliotecas externas sem poluir a fachada.

Mais informações
- Veja `log/README.md`, `log/adapter/README.md` e `log/util/README.md` para detalhes e exemplos.

