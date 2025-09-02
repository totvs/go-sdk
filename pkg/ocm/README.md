## Open Cluster Management (OCM) utilities

Este diretório tem utilitários e documentação para trabalhar com addons e integrações do Open Cluster Management.
Não há implementações de integração aqui; apenas helpers e instruções.

Dicas rápidas:
- Instalar OCM: siga a documentação oficial do Open Cluster Management.
- Addons: utilize `oc`/`kubectl` e `kustomize`/`helm` para empacotar e aplicar addons.
- Recomenda-se criar um diretório `manifests/` ou `helm/` dentro de cada addon para facilitar o CICD.

Exemplo de comando (referência):
`kubectl apply -f ./manifests/addon.yaml`

Adapte e expanda este diretório com templates e helpers conforme sua necessidade.

