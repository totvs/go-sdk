# go-sdk

Projeto base (SDK) em Go que serve como blueprint para outros projetos da TOTVS.

## Visão geral
- Contém utilitários reusáveis para logging, scaffolding de operators com kubebuilder e integração com Open Cluster Management (OCM).
- Este repositório fornece helpers, guias e exemplos — não implementa operators nem addons prontos.

### Principais características
- Logging: `zerolog` em formato JSON com suporte a `trace_id` para rastreabilidade.
- Estrutura modular: cada utilitário pode ser um submódulo (ex.: `log/`) ou pacote interno conforme necessidade.

### Estrutura do repositório

1. Módulos incluídos

- `log/` — módulo independente com utilitários de logging (pacote `logger`).
- `kubebuilder/` — utilitários, dicas e exemplos para scaffolding com `kubebuilder`.
- `ocm/` — utilitários e guias para trabalhar com Open Cluster Management (OCM).

```text
go 1.20

use (
  ./log
  ./kubebuilder
  ./ocm
)
```

2. Instale dependências e rode os exemplos de cada módulo:

```bash
cd log && go mod tidy
cd ../kubebuilder && go mod tidy
cd ../ocm && go mod tidy
```

3. Se preferir desenvolver consumindo o módulo localmente a partir de outro repositório, use `replace` no `go.mod` do consumidor:

```mod
replace github.com/totvs/go-sdk/log => /caminho/para/repositorio/log
```

Uso do logger (exemplo)

Importe o módulo e use as funções do pacote `logger`:

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

Middleware HTTP (exemplo rápido)

```go
mux := http.NewServeMux()
// ... registre handlers ...
http.ListenAndServe(":8080", logger.HTTPMiddleware(mux))
```

## Versionamento e publicação
- Tags por submódulo: crie tags com prefixo do diretório, por exemplo:

```bash
git tag -a log/v0.1.0 -m "log v0.1.0"
git push origin log/v0.1.0
```

- Consumidor: `go get github.com/totvs/go-sdk/log@v0.1.0`.
- Para major >= 2, inclua o sufixo de versão no `module` (ex.: `module github.com/totvs/go-sdk/log/v2`).

### Tag por diretório

- **O que é:** usar tags Git prefixadas com o caminho do submódulo, por exemplo `log/v0.1.0`, para versionar um módulo que vive em um subdiretório do monorepo.
- **Quando usar:** quando um submódulo precisa de ciclo de versão/release independente (ex.: `log` consumido por muitos repositórios).
- **Como criar:** tag anotada e push da tag, por exemplo:

```bash
git tag -a log/v0.1.0 -m "log v0.1.0"
git push origin log/v0.1.0
```

- **Consumidor:** `go get github.com/totvs/go-sdk/log@v0.1.0`.
- **Nota sobre major >= 2:** se o `module` incluir `/v2`, mantenha o padrão (ex.: `module github.com/totvs/go-sdk/log/v2` e tag `log/v2.0.0`).
- **Boas práticas:** use tags anotadas, garanta que o commit da tag contenha o `go.mod` do submódulo, e automatize o processo via CI quando possível.

## CI e testes
- Um script simples para rodar `go test` em todos os módulos:

```bash
find . -name 'go.mod' -print0 | xargs -0 -n1 dirname | while read -r d; do
  (cd "$d" && go test ./...)
done
```

## Boas práticas
- Coloque código público reutilizável em módulos/pacotes dentro de suas pastas (`log/`, etc.).
- Coloque código que não deve ser importado externamente em `internal/` dentro do respectivo módulo.
- Use `go.work` para desenvolvimento local e `replace` para casos pontuais.
- Documente cada módulo com `README.md` e exemplos; adicione `Example` tests para gerar documentação automática.

## Contribuindo
- Siga as políticas internas da empresa para licenciamento e contribution guidelines.

## Mais informações
- Verifique os READMEs em cada submódulo (`log/README.md`, `kubebuilder/README.md`, `ocm/README.md`) para exemplos e orientações específicas.
