package adapter

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	backend "github.com/totvs/go-sdk/metrics/internal/backend"
)

// NewPrometheusMetrics creates a simple Prometheus metrics setup
func NewPrometheusMetrics(serviceName string) (*DefaultMetricsSetup, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("ServiceName is required")
	}

	// Create isolated Prometheus registry
	registry := prometheus.NewRegistry()

	// Create Prometheus exporter for OpenTelemetry
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Create MeterProvider with the exporter
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	// Create meter (no default labels)
	meter := provider.Meter(serviceName)
	metrics := backend.NewMetrics(meter)

	return &DefaultMetricsSetup{
		Metrics:     metrics,
		Registry:    registry,
		provider:    provider,
		serviceName: serviceName,
	}, nil
}
