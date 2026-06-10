# go-sdk

SDK Go com utilitários reutilizáveis para logging, tracing e integrações.

Resumo
- Fachada de logging pública em `log/` para desacoplar consumidores de implementações
  concretas (por exemplo `zerolog`).
- Implementações concretas ficam em `log/internal` (não exportadas).
- Helpers de trace e propagation em `trace/`.
- Helpers Kubernetes em `kubernetes/status/` (status padronizado baseado em KStatus) e `utils/kubernetes/transmitter/` (transmitter genérico de CRDs).
- Exemplos em `examples/` e alvos úteis no `Makefile`.

## Setup inicial

- Após clonar o repositório, execute:

  ```bash
  make setup          # Instala lefthook via go install
  # ou
  make setup ASDF=true  # Instala lefthook via asdf
  ```

- Isso configura git hooks automáticos definidos em [`lefthook.yml`](lefthook.yml)
- Para desabilitar hooks temporariamente: `LEFTHOOK=0 git commit -m "message"`
- Para pular um hook específico: `LEFTHOOK_EXCLUDE=test git push`

## Estrutura principal

- `log/` — pacote de fachada: `facade.go`, testes e documentação (`log/README.md`).
  - `log/adapter/` — adaptadores públicos que retornam `LoggerFacade` (ex.: `NewLog`, `NewDefaultLog`).
  - `log/internal/` — implementações concretas (por exemplo `internal/backend/zerolog.go`).
  - `log/middleware/`, `log/util/` — middlewares e helpers relacionados a logging.
- `trace/` — helpers de propagation (`ContextWithTrace`, `TraceIDFromContext`, `GenerateTraceID`).
- `kubernetes/status/` — helpers para leitura/escrita de `metav1.Condition`,
  integração com KStatus oficial e geração de `Summary` padronizado.
- `utils/kubernetes/transmitter/` — transmitter genérico para publicar status de CRDs.
- `examples/` — exemplos executáveis (ex.: `examples/logger/main.go`).
- `Makefile` — targets comuns: `test`, `test-v`, `test-race`, `cover`, `cover-html`, `fmt`, `vet`, `build`, `tidy`, `ci`, `setup`, `run-example`.

Logging: API rápida
- Construtores (adapter):
  - `adapter.NewLog(w io.Writer, level)` — cria um `LoggerFacade` que escreve para `w`.
  - `adapter.NewDefaultLog()` — cria um logger com configurações padrão.
- Context helpers (em `trace` e `log`):
  - `trace.ContextWithTrace`, `trace.TraceIDFromContext`, `trace.GenerateTraceID` — propagation de trace id.
  - `log.ContextWithLogger(ctx, l)`, `log.LoggerFromContext(ctx)`, `log.FromContext(ctx)` — injeção/recuperação de `LoggerFacade`.
- Helpers de campos: `WithField`, `WithFields` (disponíveis no `LoggerFacade`).
- Erros: use `LoggerFacade.Error(err)` seguido de `Msg`/`Msgf` para incluir o campo `error` no payload. Ex.: `f.Error(err).Msg("failed")` ou `f.WithFields(...).Error(err).Msgf("failed %s", name)`.
- Globais/atalhos: `log.SetGlobal(l)`, `log.GetGlobal()` e helpers de nível `log.Debug()/Info()/Warn()/Error(err)` que retornam um `LogEvent` fluente.

Kubernetes status: API rápida

- Escrita de conditions:
  - `status.MarkReady(&conditions, gen, status.Reasons.Reconciled, "ready")`
  - `status.MarkReconciling(&conditions, gen, status.Reasons.Reconciling, "installing...")`
  - `status.MarkWaiting(&conditions, gen, status.Reasons.PreconditionNotMet, "awaiting dependency")`
  - `status.MarkStalled(&conditions, gen, status.Reasons.DependencyNotFound, "CRD not found")`
  - `status.MarkTerminating(&conditions, gen, "terminating")`
- Leitura/normalização:
  - `status.SummaryFromObject(obj)` — computa `Summary{KStatus, State, Severity, Reason, Message}` on-the-fly.
  - `status.SummaryFromUnstructured(u)` — variante para dynamic client / Unstructured.
  - `status.SummaryFromObject(obj, status.WithSummaryMapping(domainMapping))` — injeta mapeamento de domínio.
  - `status.NotFoundSummary(reason, message)` — Summary para recurso ausente no cluster.
  - `status.Compute(obj)` / `status.ComputeFromUnstructured(u)` — KStatus puro sem Summary.
- Reasons embutidos:
  - Core: `status.Reasons.Reconciled`, `.Reconciling`, `.Terminating`, `.Unknown`
  - Common: `status.Reasons.DependencyNotFound`, `.DependencyUnavailable`, `.InvalidConfiguration`, `.PermissionDenied`, `.Conflict`, `.Timeout`, `.PreconditionNotMet`
  - Domain reasons (ex: `"PendingApproval"`) são definidos no consumer, injetados via `WithSummaryMapping`.
- Documentação detalhada: `kubernetes/status/README.md`

Adicionando um adapter
- Para suportar outra biblioteca, adicione um adaptador em `log/adapter/` que construa/retorne um `log.LoggerFacade`.
- Mantenha a dependência concreta dentro de `log/internal` quando for necessário usar bibliotecas externas.

Testes e desenvolvimento
- Coloque testes ao lado do código (`*_test.go`). Use `bytes.Buffer` e `httptest` para capturar saída e comportamento HTTP.
- Se um teste alterar o logger global (`log.SetGlobal`), restaure o valor anterior com `defer log.SetGlobal(prev)`.
- Alvos úteis:
  - `make test` — roda todos os testes.
  - `make test-v` — testes em modo verbose.
  - `make test-race` — com detector de race e cobertura.
  - `make run-example` — executa `examples/logger` (use `LOG_LEVEL` para alterar o nível).

Formatação e análise estática
- Rode `make fmt` (gofmt) e `make vet` (go vet) antes de submeter mudanças.

Build / CI
- `make build` — compila os pacotes.
- `make ci` — target para CI que executa `fmt`, `vet` e `test`.

Boas práticas
- Prefira usar a abstração `log.LoggerFacade` nas bibliotecas para não acoplar
  consumidores a uma implementação concreta.
- Mantenha implementações concretas em `log/internal` para evitar vazamento de dependências.
- O logger global é armazenado com `sync/atomic.Value`: definir o global uma vez no
  startup é a prática recomendada; swaps em runtime são suportados mas use com cuidado.

Exemplos e documentação adicional
- Veja `examples/logger` para um exemplo de uso.
- Consulte `log/README.md` para documentação detalhada da fachada e exemplos de adapters.

Licença e contato
- Ver `LICENSE` (se presente) e abra issues/pull requests para contribuições.
