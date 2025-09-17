# Utilitários de integração (log/util)

O package `log/util` centraliza helpers e integrações que operam *sobre* a
fachada `LoggerFacade`. Ele não substitui adapters ou a implementação
concreta do logger — fornece utilitários que facilitam integrar outras
bibliotecas ou frameworks (por exemplo Gin, klog, logr) com a fachada.

Exemplos de utilitários neste pacote

- `gin.go` — helpers para redirecionar `gin.DefaultWriter`/`DefaultErrorWriter`
  para uma `LoggerFacade` e um `io.Writer` que transforma linhas em eventos de log.
- `klog.go` — wrapper estreito para instalar klog com a `LoggerFacade`.
- `logr.go` — helpers que retornam `logr.Logger` baseados na fachada.

Quando adicionar novos utilitários

- Coloque cada integração em seu próprio arquivo para clareza (`<target>.go`).
- Escreva testes que validem apenas a lógica do utilitário (use buffers/httptest).
- Documente o comportamento (por exemplo, se um utilitário substitui writers
  globais, documente como restaurá-los).

