# go-sdk

Projeto base (SDK) em Go que serve como blueprint para outros projetos da TOTVS.

## Visão geral
- Contém utilitários reusáveis para logging (pacote `log`).
- Este repositório fornece helpers, guias e exemplos — não implementa operators nem addons prontos.

### Principais características
- Logging: `zerolog` em formato JSON com suporte a `trace_id` para rastreabilidade.
- Estrutura modular: os utilitários ficam em pastas como `log/`.

### Estrutura do repositório

1. Módulos/pacotes incluídos

- `log/` — utilitários de logging (pacote `logger`).

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

Importe o pacote e use as funções do pacote `logger`:

```go
import (
    "context"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    l := logger.New(os.Stdout, logger.InfoLevel)
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
    l = logger.WithTraceFromContext(ctx, l)
    l.Info().Msg("aplicação iniciada")
}
```

Middleware HTTP (exemplo rápido)

```go
mux := http.NewServeMux()
// ... registre handlers ...
http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))
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

Uso básico — sem precisar importar `zerolog`:

```go
import (
    "context"
    "os"

    logger "github.com/totvs/go-sdk/log"
)

func main() {
    // cria logger que escreve em stdout com nível Info
    l := logger.New(os.Stdout, logger.InfoLevel)

    // adiciona trace no contexto e aplica ao logger
    ctx := logger.ContextWithTrace(context.Background(), "trace-1234")
    l = logger.WithTraceFromContext(ctx, l)
    l.InfoMsg("aplicação iniciada")
}
```

Adicionar campos e usar helpers:

```go
lg := logger.New(os.Stdout, logger.DebugLevel)
lg = lg.WithField("service", "orders")
lg = lg.WithFields(map[string]interface{}{"version": 3, "region": "eu"})
lg.DebugMsg("config carregada")
```

HTTP middleware (gera `trace id` automaticamente se estiver ausente):

```go
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("ok"))
})

// usa logger default
http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))

// ou com logger customizado
// myLogger := logger.New(os.Stdout, logger.DebugLevel)
// http.ListenAndServe(":8080", logger.HTTPMiddlewareWithLogger(myLogger)(mux))
```

Ao usar o middleware, se o cliente não enviar `X-Request-Id` ou `X-Correlation-Id`, o middleware gera um id seguro,
o coloca no contexto, adiciona ao log como `trace_id` e também inclui o header `X-Request-Id` na resposta para facilitar
correlação entre cliente e servidor.

## Contribuindo
- Siga as políticas internas da empresa para licenciamento e contribution guidelines.

## Mais informações
- Verifique o README em `log/README.md` para exemplos e orientações específicas.
