# Logging (log)

Este módulo fornece uma fachada de logging (abstração) com implementação
baseada em `zerolog`. Os consumidores usam a interface pública do pacote
(`LoggerFacade`) sem depender diretamente de `zerolog`.

## Constantes públicas

- `TraceIDHeader` — nome do header HTTP para trace id (`X-Request-Id`).
- `TraceIDCorrelationHeader` — nome alternativo para correlação (`X-Correlation-Id`).
- `TraceIDField` — nome do campo JSON adicionado aos logs (`trace_id`).

## Estrutura

- `logger.go` — implementação concreta (interno) baseada em `zerolog`.
- `facade.go` — a fachada pública `LoggerFacade` e helpers de contexto.
- `logr_adapter.go` — adaptador para `logr` (usado por `controller-runtime`).
- `middleware/` — helpers e middleware HTTP (ex.: injeção de trace id).

## Como usar

1. Adicione a dependência no seu módulo (ou use `replace` localmente durante o desenvolvimento).

2. Use a fachada pública para emitir logs sem conhecer a implementação concreta:

```go
import (
    "os"
    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // cria uma fachada que escreve em stdout com nível Info
    lg := logger.NewLog(os.Stdout, logger.InfoLevel)

    // registra como logger global (opcional)
    logger.SetGlobal(lg)

    // atalhos de pacote usam o logger global (via Event fluente)
    logger.Info().Msg("aplicação iniciada")
}
```

### API rápida

- Construtores: `NewLog(w io.Writer, level Level) LoggerFacade`, `NewDefaultLog()`.
- Context helpers: `ContextWithTrace`, `TraceIDFromContext`, `ContextWithLogger`, `LoggerFromContext`, `FromContext`.
- Fields: `WithField`, `WithFields`.
- Erros: use a API fluente: `Error(err).Msg("message")` ou encadeie campos antes de chamar `Msg`.
- Globals: `SetGlobal`, `GetGlobal` e atalhos `logger.Info/...`.

## Novos helpers e middleware

- `WithField(key, value)` — adiciona um campo ao logger (encadeável).
- `WithFields(map[string]interface{})` — adiciona múltiplos campos.
- Middlewares HTTP fornecem geração/propagação de `trace_id` e injeção do logger no contexto.

## Middleware HTTP

O middleware disponível aceita uma `LoggerFacade` e gera um `trace id` seguro
quando o cliente não fornece `X-Request-Id` ou `X-Correlation-Id`.

Comportamento principal:

- Insere o `trace_id` no contexto com `ContextWithTrace`.
- Adiciona `trace_id` ao log emitido no nível de request.
- Define o header `X-Request-Id` na resposta quando ausente.

Configuração via `MiddlewareOptions`:

- `LogRequest bool` — emitir log de request (padrão: true).
- `InjectLogger bool` — injetar a fachada no contexto (padrão: true).
- `AddTraceHeader bool` — adicionar `X-Request-Id` na resposta (padrão: true).

Exemplo de uso:

```go
opts := middleware.MiddlewareOptions{
    LogRequest:    false,
    InjectLogger:  true,
    AddTraceHeader: true,
}
http.ListenAndServe(":8080", middleware.HTTPMiddlewareWithOptions(myLogger, opts)(mux))
```

Nota: o middleware marca o contexto quando já emitiu o log de request. Handlers
que também logam devem checar `logger.LoggedFromContext(r.Context())` para
evitar duplicação.

## Adapters

Para integrar a fachada com bibliotecas que exigem uma API diferente,
existem adaptadores dentro do pacote. Atualmente há um adaptador para
`logr` (usado por `controller-runtime`).

- Arquivo: `log/logr_adapter.go`.
- Principais helpers:
  - `log.NewLogrAdapter(l LoggerFacade) logr.Logger` — cria um `logr.Logger` que delega para `l`.
  - `log.NewGlobalLogr()` — atalho que usa `log.GetGlobal()`.

Exemplo (usar com controller-runtime):

```go
import (
    "os"

    logger "github.com/totvs/go-sdk/log"
    ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
    logger.SetGlobal(logger.NewLog(os.Stdout, logger.InfoLevel))

    // converte a LoggerFacade em logr.Logger para o controller-runtime
    ctrl.SetLogger(logger.NewLogrAdapter(logger.GetGlobal()))

    // iniciar manager, controllers, etc.
}
```

### Integração com klog / component-base logs

Você pode redirecionar as chamadas de `klog` para a fachada deste pacote
convertendo a `LoggerFacade` para um `logr.Logger` e registrando-o em `klog`.
Exemplo:

```go
import (
    "github.com/totvs/go-sdk/log"
    "k8s.io/klog/v2"
)

// liga o klog ao logger global do pacote
log.InstallGlobalKlog()

// ou explicitamente com uma fachada criada
lg := log.NewLog(os.Stdout, log.InfoLevel)
log.InstallKlogLogger(lg)

// quando usar component-base/logs, chame logs.InitLogs() conforme recomendado
// pelo Kubernetes e chame log.InstallGlobalKlog() durante a inicialização.
```

#### Helper: `InstallKlogWithComponentBase`

O helper `InstallKlogWithComponentBase` inicializa o subsystem de logs do
`k8s.io/component-base/logs` e instala a `LoggerFacade` em `klog`. Ele
devolve uma função de cleanup (que chama `logs.FlushLogs`) que deve ser
invocada no final do `main` (por exemplo via `defer`).

Exemplo:

```go
package main

import (
    "flag"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    flag.Parse() // necessário antes de InitLogs

    lg := logger.NewLog(os.Stdout, logger.InfoLevel)

    // Instala o klog e obtém a função de cleanup (flush)
    cleanup := logger.InstallKlogWithComponentBase(lg)
    defer cleanup()

    // restante da inicialização e execução
}
```

Notas sobre o adaptador `logr`:

- Mapeamento de verbosidade: `V(0)` → `Info`, `V(n>0)` → `Debug`.
 - Use a API fluente: `Error(err).Msg(...)`; quando houver campos adicionais, encadeie `WithFields(...).Error(err).Msg(...)`.
- `Enabled()` do sink retorna `true` (o filtro final fica a cargo do logger subjacente).

## Inserindo o logger no contexto (facade)

```go
lg := logger.NewLog(os.Stdout, logger.DebugLevel)
ctx := logger.ContextWithLogger(context.Background(), lg)

f := logger.FromContext(ctx)
f.Info().Msg("using injected logger via facade")

f3 := f.WithFields(map[string]interface{}{"service": "orders", "version": 3})
f3.Info().Msg("request processed")

err := errors.New("boom")
f.Error(err).Msg("operation failed")
f.Error(err).Msgf("failed to %s", "start")
f.WithFields(map[string]interface{}{"service": "orders"}).Error(err).Msg("failed to start")
```

## Handler helper

```go
func handler(w http.ResponseWriter, r *http.Request) {
    lg, logged := middleware.GetLoggerFromRequest(r)
    if !logged {
        lg.Info().Msg("handler received request")
    }
    // lógica do handler
}
```

## Exemplo completo e saída esperada

Exemplo simplificado de servidor HTTP com middleware que gera `trace_id`:

```go
package main

import (
    "net/http"
    "os"

    logger "github.com/totvs/go-sdk/log"
    middleware "github.com/totvs/go-sdk/log/middleware/http"
)

func main() {
    l := logger.NewLog(os.Stdout, logger.InfoLevel)

    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    http.ListenAndServe(":8080", middleware.HTTPMiddlewareWithLogger(l)(mux))
}
```

Uma chamada GET para `/ping` sem `X-Request-Id` pode gerar uma linha JSON como:

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

## Dicas

- Ajuste o nível de log via `LOG_LEVEL`. Valores aceitos (case-insensitive): `DEBUG`, `INFO` (padrão), `WARN` / `WARNING`, `ERROR`.
- Para builds locais com módulo substituído, use `replace` no `go.mod`.
