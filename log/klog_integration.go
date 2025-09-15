package log

import (
    logs "k8s.io/component-base/logs"
)

// InstallKlogWithComponentBase inicializa o suporte de logging do
// `k8s.io/component-base/logs` e registra a `LoggerFacade` em `klog`.
//
// Deve ser invocado após o processamento das flags do programa (por
// exemplo, após `flag.Parse()`). Retorna uma função de cleanup que deve
// ser chamada (por exemplo via `defer`) para garantir flush dos logs.
func InstallKlogWithComponentBase(l LoggerFacade) func() {
    logs.InitLogs()
    InstallKlogLogger(l)
    return logs.FlushLogs
}

