package util

import (
	lg "github.com/totvs/go-sdk/log"
	"github.com/totvs/go-sdk/log/adapter"
	logs "k8s.io/component-base/logs"
)

// InstallKlogWithComponentBase inicializa o subsystem de logs do
// `k8s.io/component-base/logs` e registra a `LoggerFacade` em `klog`.
//
// Deve ser invocado após o processamento das flags do programa (por
// exemplo, após `flag.Parse()`). Retorna uma função de cleanup que deve
// ser chamada (por exemplo via `defer`) para garantir flush dos logs.
func InstallKlogWithComponentBase(lf lg.LoggerFacade) func() {
	logs.InitLogs()
	adapter.InstallKlogLogger(lf)
	return logs.FlushLogs
}

// InstallGlobalKlog é uma conveniência que instala o logger global do
// pacote `log` em klog.
func InstallGlobalKlog() func() { return InstallKlogWithComponentBase(lg.GetGlobal()) }
