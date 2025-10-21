# Metrics SDK

SDK para instrumentação de métricas usando OpenTelemetry + Prometheus

## 🚀 Setup Rápido

### 1. Instalação
```go
go get github.com/totvs/go-sdk/metrics
```

### 2. Setup Básico (3 linhas)
```go
package main

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/totvs/go-sdk/metrics/util"
)

func main() {
    // 1. Setup completo com labels TOTVS automáticos
    setup, _ := util.SetupPrometheusMetrics("meu-servico")

    // 2. Expor endpoint de métricas
    http.Handle("/metrics", promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{}))

    // 3. Servidor HTTP normal
    http.ListenAndServe(":8080", nil)
}
```

**Todas as métricas terão automaticamente os labels TOTVS de aplicação:**
- `platform="totvs.apps"`

**Nota:** `metric_type` e `metric_class` são **obrigatórios** na criação de cada métrica (veja seção TOTVS abaixo)

## 📊 O que você ganha automaticamente

### Endpoint `/metrics`
Acesse `http://localhost:8080/metrics` e encontre:

```prometheus
# Métricas do sistema
target_info{service_name="meu-servico"} 1

# Suas métricas customizadas (quando criadas)
my_counter_total{label="value"} 42
my_histogram_bucket{le="1.0"} 10
```

### Instrumentação Customizada
```go
// Criar métricas específicas (já têm labels TOTVS)
counter := setup.Metrics.Counter("requests_total")
histogram := setup.Metrics.Histogram("duration_seconds")

// Adicionar labels customizados (além dos TOTVS)
counter.Inc(ctx, metrics.Attr("endpoint", "/api/users"))
histogram.Record(ctx, duration.Seconds(), metrics.Attr("method", "GET"))
```

## 🔧 Recursos Inclusos

### ✅ **Automático**
- **Shutdown graceful** - Captura SIGINT/SIGTERM automaticamente
- **Registry customizado** - Isolado do Prometheus global
- **Labels semânticos** - Seguindo convenções OpenTelemetry

### ✅ **Tipos de Métricas**
- **Counter** - Valores que só aumentam (requests, errors)
- **Gauge** - Valores que sobem/descem (memory, connections)
- **Histogram** - Distribuições (latency, sizes)

### ✅ **Integração**
- **Prometheus** - Scraping via `/metrics`
- **Grafana** - Dashboards automáticos
- **OpenTelemetry** - Padrão da indústria

## 🎯 Exemplo com Middleware HTTP

```go
// Middleware simples para HTTP metrics
func metricsMiddleware(metrics metrics.MetricsFacade) func(http.Handler) http.Handler {
    counter := metrics.Counter("http_requests_total")
    duration := metrics.Histogram("http_duration_seconds")

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)

            attrs := []metrics.Attribute{
                metrics.Attr("method", r.Method),
                metrics.Attr("path", r.URL.Path),
            }

            counter.Inc(r.Context(), attrs...)
            duration.Record(r.Context(), time.Since(start).Seconds(), attrs...)
        })
    }
}

// Usar o middleware
mux := http.NewServeMux()
handler := metricsMiddleware(setup.Metrics)(mux)
http.ListenAndServe(":8080", handler)
```

## 🚨 Controle Manual (Opcional)

```go
// Shutdown manual se necessário
setup.Shutdown() // Safe para múltiplas chamadas
```

## 🔗 Conceitos

- **MetricsFacade** - Interface para criar métricas
- **Registry** - Registro isolado do Prometheus
- **Attributes** - Labels/tags para suas métricas
- **Shutdown** - Cleanup automático de recursos

---

## 🏢 Customizar Labels TOTVS

Por padrão, `SetupPrometheusMetrics()` usa labels TOTVS sensatos. Para customizar, use `SetupTOTVSMetrics()`:

```go
import "github.com/totvs/go-sdk/metrics/util"

setup, err := util.SetupTOTVSMetrics(util.TOTVSMetricsConfig{
    ServiceName:  "meu-servico",
    Platform:     "totvs.apps",                 // Ex: erp.protheus, fluig.apps, carol.apps
})
if err != nil {
    log.Fatal(err)
}
defer setup.Shutdown()
```

### Labels TOTVS

**Escopo da Aplicação** (aplicados automaticamente no setup):

| Label | Descrição | Valores |
|-------|-----------|---------|
| `platform` | Plataforma de origem | `erp.protheus`, `totvs.apps`, `fluig.apps`, etc |

**Escopo da Métrica** (adicionados por métrica via `Attr()`):

| Label | Descrição | Constantes | Valores |
|-------|-----------|-----------|---------|
| `metric_type` | Destino da métrica | `MetricTypeTech`<br>`MetricTypeBusiness` | `tech` (somente Prometheus)<br>`bus` (enviada para Carol) |
| `metric_class` | Escopo da métrica | `MetricClassService`<br>`MetricClassInstance` | `service` (agregado)<br>`instance` (por instância) |

### Exemplo Completo TOTVS

```go
// Setup com labels TOTVS (application-level)
setup, _ := util.SetupTOTVSMetrics(util.TOTVSMetricsConfig{
    ServiceName:  "pedidos-api",
    Platform:     "totvs.apps",
})

// Métrica técnica (Prometheus apenas)
// metric_type e metric_class são obrigatórios na criação
httpCounter := setup.Metrics.Counter("http_requests_total",
    util.MetricTypeTech,
    util.MetricClassService,
)
httpCounter.Inc(ctx,
    metrics.Attr("endpoint", "/api/orders"),
)

// Métrica de negócio (enviada para Carol)
ordersCounter := setup.Metrics.Counter("business_orders_total",
    util.MetricTypeBusiness,
    util.MetricClassService,
)
ordersCounter.Inc(ctx,
    metrics.Attr("status", "completed"),
)

// Métrica de instância (CPU de um pod específico)
cpuGauge := setup.Metrics.Gauge("process_cpu_percent",
    util.MetricTypeTech,
    util.MetricClassInstance,
)
cpuGauge.Set(ctx, 42.5)

// Resultado no Prometheus:
// http_requests_total{platform="totvs.apps",metric_type="tech",metric_class="service",endpoint="/api/orders"} 1
// business_orders_total{platform="totvs.apps",metric_type="bus",metric_class="service",status="completed"} 1
// process_cpu_percent{platform="totvs.apps",metric_type="tech",metric_class="instance"} 42.5
```

### Nomenclatura de Métricas (RFC)

Siga o padrão: `<namespace>_<name>_<base-unity>`

```go
// ✅ Correto
"http_request_duration_seconds"
"amqp_inbound_message_count"
"process_memory_usage_bytes"
"business_orders_total"

// ❌ Evite
"myMetric"
"request-time"
"OrderCount"
```

---

**Pronto!** Com 3 linhas você tem métricas completas e endpoint Prometheus funcionando. 🎉