# Adapters (log/adapter)

Esta pasta contém *adapters* que convertem bibliotecas de logging externas
para a fachada pública do pacote `log` (`LoggerFacade`). A intenção é isolar
dependências externas (por exemplo `zerolog`, `logr`) de modo que o resto da
base de código dependa apenas da API do `log`.

Boas práticas

- Nomeie arquivos como `<lib>_adapter.go` (por exemplo `zerolog_adapter.go`).
- Cada adapter deve implementar `github.com/totvs/go-sdk/log.LoggerFacade`.
- Exponha um construtor claro, por exemplo `NewLog(w io.Writer, level Level) LoggerFacade` e
  opcionalmente `NewDefaultLog()` para conveniência em exemplos/tests.
- Mantenha a dependência externa apenas dentro deste package (`log/adapter`) —
  não a exponha para consumidores do pacote `log`.
- Adicione testes ao package `log` que validem a integração através do adapter
  (use `bytes.Buffer` para capturar saída quando aplicável).

Por que usar adapters?

- Eles implementam o *adapter pattern*: convertem a API do provider para a
  interface que sua aplicação usa internamente (`LoggerFacade`).
- Permitem trocar implementações ou suportar múltiplas bibliotecas sem alterar
  o código que emite logs.

Casos de uso: `zerolog` vs `logr`

- `zerolog_adapter` (direção: zerolog → LoggerFacade)
  - Objetivo: construir uma `LoggerFacade` a partir do backend `zerolog`.
  - Uso típico: quando sua aplicação quer *usar* `zerolog` como backend
    concreto para a fachada (`LoggerFacade`).
  - Observação: com a organização recomendada, esse adapter pode ser um
    thin‑shim público que delega para uma implementação interna (`internal/backends/zerolog`).

- `logr_adapter` (direção: LoggerFacade → `logr`)
  - Objetivo: expor a `LoggerFacade` como um `logr.Logger` (implementando um
    `logr.LogSink`).
  - Uso típico: quando bibliotecas (ex.: controller‑runtime, componentes k8s)
    exigem um `logr.Logger` — este adapter permite que essas bibliotecas
    emitam logs que acabem na nossa fachada.

Por que ambos podem ser necessários

- Eles resolvem problemas diferentes: um fornece o backend; o outro permite
  integrar consumidores que esperam outra API.
- Manter os dois permite isolar dependências (não expor `zerolog` no core)
  enquanto ainda integra bem com o ecossistema (`logr`/k8s).

Recomendação

- Mantenha `zerolog` como implementação interna (em `internal/backends`) e
  ofereça um adapter público (`adapter.NewLog`/`adapter.NewDefaultLog`) como
  conveniência.
- Mantenha o `logr` adapter público para integração com bibliotecas que o
  exigem. Isso preserva compatibilidade com o ecossistema Kubernetes.

