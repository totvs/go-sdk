go-sdk

Projeto base (SDK) em Go que serve como blueprint para outros projetos da TOTVS.

Visão geral:
- Contém utilitários reusáveis para logging, scaffolding de operators com kubebuilder e integração com Open Cluster Management (OCM).
- Este repositório não implementa operators nem addons; fornece helpers e convenções.

Principais características:
- Logging: `zerolog` em formato JSON com suporte a `trace_id` para rastreabilidade.
- Estrutura de pastas em inglês para facilitar integração com outros repositórios.

Estrutura:
- `log/` : utilitário de logging (zerolog + trace_id).
- `kubebuilder/` : dicas e helpers para kubebuilder.
- `ocm/` : dicas e helpers para Open Cluster Management.

Como começar:
1. Ajuste o módulo Go se necessário: `module github.com/totvs/<your-repo>`
2. Baixe dependências: `go mod tidy` (requer acesso à internet).
3. Rodar exemplo de logging: `go run ./log/cmd/example`

Logging:
- O logger escreve em JSON por padrão.
- Para incluir `trace_id` use `pkg/log.ContextWithTrace(ctx, traceID)` e depois `pkg/log.WithTraceFromContext(ctx, logger)`.


Exemplos de uso e trechos de código estão nos READMEs dos respectivos pacotes:

- `log/README.md` — exemplos de uso do logger (inclui middleware HTTP e injeção de `trace_id`).
- `pkg/kubebuilder/README.md` — exemplos e comandos para scaffolding de operators.
- `pkg/ocm/README.md` — exemplos e recomendações para addons OCM.

Próximos passos:
- Adicionar templates, scripts e integrações conforme necessário.
- Adotar este repositório como base para novos projetos Go.

Licença e contribuições:
- Siga as políticas internas da empresa para licença e contribution guidelines.
