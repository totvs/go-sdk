# Kubebuilder utilities

Este módulo contém utilitários, dicas e exemplos para iniciar projetos de Operators usando o `kubebuilder`.
Não há código de operator implementado aqui; apenas instruções e helpers que servem como blueprint.

Passos típicos:
- Inicializar projeto: `kubebuilder init --domain <your.domain> --repo github.com/totvs/<repo>`
- Criar API e Controller: `kubebuilder create api --group <group> --version <version> --kind <Kind>`
- Testar localmente com `make` gerado pelo kubebuilder (verifique o Makefile gerado pelo scaffold).