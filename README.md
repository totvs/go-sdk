# go-sdk

Projeto base (SDK) em Go que serve como blueprint para outros projetos da TOTVS.

## Visão geral
- Contém utilitários reusáveis para logging (pacote `log`).
- Este repositório fornece helpers, guias e exemplos — não implementa operators nem addons prontos.

### Principais características
- Logging: formato JSON com suporte a `trace_id` para rastreabilidade.
- Estrutura modular: os utilitários ficam em pastas como `log/`.

### Estrutura do repositório

1. Módulos/pacotes incluídos

- `log/` — utilitários de logging (pacote `log`).

```text
module github.com/totvs/go-sdk

go 1.25
```

2. Instale dependências:

```bash
# o `go.mod` agora está na raiz do repositório
go mod tidy
```

3. Se preferir desenvolver consumindo o repositório localmente a partir de outro repositório, use `replace` no `go.mod` do consumidor:

```mod
replace github.com/totvs/go-sdk => /caminho/para/repositorio
```

Uso do logger (exemplo)

Importe o pacote e use as funções do pacote `log` (ex.: `logger "github.com/totvs/go-sdk/log"`):

```go
import (
    "context"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // use the API to create a logger instance
    f := logger.NewLog(os.Stdout, logger.InfoLevel)

    // Alternatively use the default constructor which writes to stdout and
    // reads the log level from the `LOG_LEVEL` environment variable:
    // f := logger.NewDefaultLog()
    // For per-request trace ids prefer using the HTTP middleware which
    // will inject the trace id from the request into the logger context.
    f.Info().Msg("aplicação iniciada")

    // tornar este logger a instância global utilizada por atalhos do pacote
    logger.SetGlobal(f)
    logger.Info().Msg("usando logger global")
}
```

Configuração via ambiente:

- Ajuste o nível de log via `LOG_LEVEL`. Valores aceitos (case-insensitive): `DEBUG`, `INFO` (padrão), `WARN` / `WARNING`, `ERROR`.
- Exemplo: `export LOG_LEVEL=DEBUG` antes de iniciar a aplicação.

Middleware HTTP (exemplo rápido)

```go
import (
    "net/http"
    middleware "github.com/totvs/go-sdk/log/middleware/http"
)

mux := http.NewServeMux()
// ... registre handlers ...
http.ListenAndServe(":8080", middleware.HTTPMiddleware(mux))
```

## Versionamento e publicação
- O repositório usa um único módulo Go na raiz: `module github.com/totvs/go-sdk`.
- Para consumir um pacote deste repositório use a import path, por exemplo: `github.com/totvs/go-sdk/log`.
- Consumidor: `go get github.com/totvs/go-sdk@v0.1.0`.

## CI e testes
- Um script simples para rodar `go test` em todo o repositório:

```bash
go test ./...
```

## Boas práticas
- Coloque código público reutilizável em pacotes dentro de suas pastas (`log/`, etc.).
- Coloque código que não deve ser importado externamente em `internal/` dentro do respectivo pacote.
- Use `go.work` para desenvolvimento local e `replace` para casos pontuais.
- Documente cada pacote com `README.md` e exemplos; adicione `Example` tests para gerar documentação automática.

## Exemplos rápidos

Uso básico:

```go
import (
    "context"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // criando uma instância de logger (a implementação concreta é ocultada)
    f := logger.NewLog(os.Stdout, logger.InfoLevel)

    // per-request trace ids are normally added by the HTTP middleware;
    f.Info().Msg("aplicação iniciada")

    // opcional: definir como logger global para usar atalhos como `logger.Info()`
    logger.SetGlobal(f)
    logger.Info().Msg("mensagem via logger global")

    // note: error logging uses an explicit error parameter via the fluent API:
    // `f.Error(err).Msg("failed to start")` — you can chain fields before calling `Msg`.
}
```

Adicionar campos e usar helpers:

```go
f := logger.NewLog(os.Stdout, logger.DebugLevel)
f = f.WithField("service", "orders")
f = f.WithFields(map[string]interface{}{"version": 3, "region": "eu"})
    f.Debug().Msg("config carregada")
```

HTTP middleware (gera `trace id` automaticamente se estiver ausente):

```go
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("ok"))
})

// usa logger default
http.ListenAndServe(":8080", middleware.HTTPMiddleware(mux))

// or with a custom logger
// myLogger := logger.NewLog(os.Stdout, logger.DebugLevel)
// http.ListenAndServe(":8080", middleware.HTTPMiddlewareWithLogger(myLogger)(mux))
```

Ao usar o middleware, se o cliente não enviar `X-Request-Id` ou `X-Correlation-Id`, o middleware gera um id seguro,
o coloca no contexto, adiciona ao log como `trace_id` e também inclui o header `X-Request-Id` na resposta para facilitar
correlação entre cliente e servidor.

## Contribuindo
- Siga as políticas internas da empresa para licenciamento e contribution guidelines.

## Mais informações
- Verifique o README em `log/README.md` para exemplos e orientações específicas.
