package main

import (
	"context"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/totvs/go-sdk/examples/metrics/handlers"
	"github.com/totvs/go-sdk/metrics"
	"github.com/totvs/go-sdk/metrics/adapter"
	"github.com/totvs/go-sdk/metrics/util"
)

// basicCounterExample demonstra uso básico de um counter com OTEL.
// NewDefaultMetrics já inclui Prometheus exporter e labels TOTVS automáticos.
func basicCounterExample() {
	setup, err := adapter.NewPrometheusMetrics("basic-example")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	ctx := context.Background()
	counter := setup.Metrics.GetOrCreateCounter("requests_total", metrics.MetricTypeTech, metrics.MetricClassService)
	counter.Inc(ctx) // Terá platform="totvs.apps"
	counter.Add(ctx, 5)

	log.Println("✓ Basic counter example completed")
	log.Println("  (Includes Prometheus exporter + TOTVS labels)")
}

// setGlobalMetricsExample registra o metrics como global e usa via atalhos do pacote.
func setGlobalMetricsExample() {
	setup, err := adapter.NewPrometheusMetrics("global-example")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}

	metrics.SetGlobal(setup.Metrics)

	// Usa via package-level helpers
	counter := metrics.NewCounter("global_requests", metrics.MetricTypeTech, metrics.MetricClassService)
	counter.Inc(context.Background())

	log.Println("✓ Global metrics example completed")
}

// injectedMetricsExample demonstra injeção/extração de metrics via contexto.
func injectedMetricsExample() {
	setup, err := adapter.NewPrometheusMetrics("injected-example")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}

	ctx := metrics.ContextWithMetrics(context.Background(), setup.Metrics)
	mf := metrics.FromContext(ctx)

	counter := mf.GetOrCreateCounter("injected_operations", metrics.MetricTypeTech, metrics.MetricClassService)
	counter.Inc(ctx)

	log.Println("✓ Injected metrics example completed")
}

// multipleMetricTypesExample demonstra uso de Counter, Gauge e Histogram.
func multipleMetricTypesExample() {
	setup, err := adapter.NewPrometheusMetrics("multi-types")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}

	ctx := context.Background()

	// Counter
	counter := setup.Metrics.GetOrCreateCounter("api_requests_total", metrics.MetricTypeTech, metrics.MetricClassService)
	counter.Add(ctx, 10)

	// Gauge
	gauge := setup.Metrics.GetOrCreateGauge("active_connections", metrics.MetricTypeTech, metrics.MetricClassService)
	gauge.Set(ctx, 42)

	// Histogram
	histogram := setup.Metrics.GetOrCreateHistogram("request_duration_seconds", metrics.MetricTypeTech, metrics.MetricClassService)
	histogram.Record(ctx, 0.125)
	histogram.Record(ctx, 0.250)

	log.Println("✓ Multiple metric types example completed")
}

// otelWithPrometheusExporterExample demonstra uso de OTEL adapter com Prometheus exporter manual.
func otelWithPrometheusExporterExample() {
	// Cria registry isolado do Prometheus
	registry := prometheus.NewRegistry()

	// Cria exporter do Prometheus para OTEL
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
	)
	if err != nil {
		log.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Cria MeterProvider com o exporter
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	// Usa adapter com provider customizado (sem TOTVS defaults)
	m := adapter.NewMetricsWithProvider(provider, "otel-prom-example")

	ctx := context.Background()

	// Usa métricas normalmente (sem labels TOTVS automáticos)
	counter := m.GetOrCreateCounter("custom_requests_total", metrics.MetricTypeTech, metrics.MetricClassService)
	gauge := m.GetOrCreateGauge("custom_memory_bytes", metrics.MetricTypeTech, metrics.MetricClassService)
	histogram := m.GetOrCreateHistogram("custom_duration_seconds", metrics.MetricTypeTech, metrics.MetricClassService)

	counter.Add(ctx, 42)
	gauge.Set(ctx, 1024*1024*256) // 256 MB
	histogram.Record(ctx, 0.075)

	log.Println("✓ OTEL with Prometheus exporter example completed")
	log.Println("  (Registry available:", registry != nil, ")")
	log.Println("  (Can be used with: promhttp.HandlerFor(registry, ...))")
}

// prometheusSetupExample demonstra setup completo com Prometheus para expor endpoint /metrics.
func prometheusSetupExample() {
	setup, err := adapter.NewPrometheusMetrics("prometheus-example")
	if err != nil {
		log.Fatalf("Failed to setup Prometheus metrics: %v", err)
	}
	defer setup.Shutdown()

	ctx := context.Background()

	// Cria e usa métricas
	counter := setup.Metrics.GetOrCreateCounter("http_requests_total", metrics.MetricTypeTech, metrics.MetricClassService)
	gauge := setup.Metrics.GetOrCreateGauge("memory_usage_bytes", metrics.MetricTypeTech, metrics.MetricClassService)
	histogram := setup.Metrics.GetOrCreateHistogram("http_duration_seconds", metrics.MetricTypeTech, metrics.MetricClassService)

	// Simula algumas métricas
	counter.Add(ctx, 100)
	gauge.Set(ctx, 1024*1024*128) // 128 MB
	histogram.Record(ctx, 0.043)

	log.Println("✓ Prometheus setup (util) example completed")
	log.Println("  (Registry ready for HTTP handler)")
	log.Println("  (Use: http.Handle(\"/metrics\", setup.Handler()))")
}

// adapterWithMetricsEndpointExample demonstra adapter.NewDefaultMetrics com endpoint /metrics.
func adapterWithMetricsEndpointExample() {
	setup, err := adapter.NewPrometheusMetrics("adapter-with-endpoint")
	if err != nil {
		log.Fatalf("Failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	ctx := context.Background()

	// Cria métricas (já tem labels TOTVS: platform="totvs.apps")
	counter := setup.Metrics.GetOrCreateCounter("api_calls_total", metrics.MetricTypeTech, metrics.MetricClassService)
	counter.Add(ctx, 42)

	// Para expor /metrics, use o helper:
	// http.Handle("/metrics", setup.Handler())
	// Ou com opções customizadas:
	// http.Handle("/metrics", setup.HandlerWithOpts(promhttp.HandlerOpts{...}))

	log.Println("✓ Adapter with /metrics endpoint example completed")
	log.Println("  (Use setup.Handler() para expor endpoint)")
	log.Println("  (Sem precisar importar promhttp!)")
}

// totvsMetricsExample demonstra setup com labels da rfc.
func totvsMetricsExample() {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{
		ServiceName: "totvs-example",
		Platform:    "totvs.apps",
	})
	if err != nil {
		log.Fatalf("Failed to setup TOTVS metrics: %v", err)
	}
	defer setup.Shutdown()

	ctx := context.Background()

	// Métrica técnica (Prometheus apenas, não vai para Carol)
	techCounter := setup.Metrics.GetOrCreateCounter("http_requests_total", metrics.MetricTypeTech, metrics.MetricClassService)
	techCounter.Add(ctx, 100)

	// Métrica de negócio (enviada para Carol)
	businessCounter := setup.Metrics.GetOrCreateCounter("business_orders_total", metrics.MetricTypeBusiness, metrics.MetricClassService)
	businessCounter.Add(ctx, 50)

	// Métrica de instância (CPU individual)
	cpuGauge := setup.Metrics.GetOrCreateGauge("process_cpu_usage_percent", metrics.MetricTypeTech, metrics.MetricClassInstance)
	cpuGauge.Set(ctx, 42.5)

	log.Println("✓ TOTVS metrics example completed")
	log.Println("  (Application-level labels: platform)")
	log.Println("  (Per-metric labels: metric_type, metric_class)")
}

// httpServerExample inicia um servidor HTTP com métricas via Prometheus.
func httpServerExample() {
	setup, err := adapter.NewPrometheusMetrics("http-server")
	if err != nil {
		log.Fatalf("Failed to setup metrics: %v", err)
	}
	defer setup.Shutdown()

	// Application routes (wrapped with metrics middleware)
	appMux := http.NewServeMux()
	appMux.HandleFunc("/health/live", handlers.HealthLive)
	appMux.HandleFunc("/health/ready", handlers.HealthReady)

	// Wrap app routes with metrics middleware
	handler := util.WithMetrics(setup.Metrics, setup.ServiceName(), appMux)

	// Final mux with /metrics endpoint (usando helper do SDK!)
	finalMux := http.NewServeMux()
	finalMux.Handle("/metrics", handlers.Metrics(setup.Registry)) // handlers.Metrics também é um helper
	finalMux.Handle("/", handler)

	log.Println("✓ Starting HTTP server on :8080")
	log.Println("  • Metrics: http://localhost:8080/metrics")
	log.Println("  • Health:  http://localhost:8080/health/live")
	log.Println("  • Nota: util.PrometheusSetup também tem setup.Handler() se preferir!")
	log.Println("  Press Ctrl+C to stop")

	// Listen and serve (blocks until server stops)
	err = http.ListenAndServe(":8080", finalMux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func main() {
	log.Println("=== Metrics Examples ===")
	log.Println()

	log.Println("1. Basic Counter (OTEL adapter)")
	basicCounterExample()

	log.Println("\n2. Global Metrics (OTEL adapter)")
	setGlobalMetricsExample()

	log.Println("\n3. Injected Metrics (OTEL adapter)")
	injectedMetricsExample()

	log.Println("\n4. Multiple Metric Types (OTEL adapter)")
	multipleMetricTypesExample()

	log.Println("\n5. OTEL with Prometheus Exporter (adapter + manual setup)")
	otelWithPrometheusExporterExample()

	log.Println("\n6. Prometheus Setup (util - batteries included)")
	prometheusSetupExample()

	log.Println("\n7. Adapter with /metrics endpoint (setup.Handler())")
	adapterWithMetricsEndpointExample()

	log.Println("\n8. TOTVS Metrics")
	totvsMetricsExample()

	log.Println("\n9. HTTP Server with Metrics (util + Prometheus)")
	httpServerExample()

	log.Println("\n=== All examples completed ===")
}
