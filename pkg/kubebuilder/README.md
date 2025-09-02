## Kubebuilder utilities

Este diretório contém utilitários e dicas para iniciar projetos de Operators usando o `kubebuilder`.
Não há código de operator implementado aqui; apenas instruções e helpers que servem como blueprint.

Passos típicos:
- Inicializar projeto: `kubebuilder init --domain <your.domain> --repo github.com/totvs/<repo>`
- Criar API e Controller: `kubebuilder create api --group <group> --version <version> --kind <Kind>`
- Testar localmente com `make` gerado pelo kubebuilder (verifique o Makefile gerado pelo scaffold).

Recomendações:
- Mantenha controllers leves e delegue lógica para pacotes em `pkg/`.
- Versionamento de CRDs e migrações: use ferramentas de migração quando necessário.
- Integração com CI/CD: automatize `make generate`, `make manifests` e `make test`.

Este arquivo é apenas referência; adicione templates ou scripts conforme necessário.

