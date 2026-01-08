package adapter

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	mt "github.com/totvs/go-sdk/metrics"
	backend "github.com/totvs/go-sdk/metrics/internal/backend"
)

type TOTVSMetricsConfig struct {
	ServiceName string // ServiceName is the name of the service generating metrics
	Platform    string // Examples: "totvs.apps", "erp.protheus", "fluig.apps", "carol.apps"
}

// Validate checks if the TOTVS configuration has valid values
func (c TOTVSMetricsConfig) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("ServiceName is required")
	}

	if c.Platform == "" {
		return fmt.Errorf("platform is required (example: totvs.apps)")
	}

	return nil
}

// NewMetrics delegates to the internal OpenTelemetry backend with the provided meter.
func NewMetrics(meter metric.Meter) mt.MetricsFacade {
	return backend.NewMetrics(meter)
}

// DefaultMetricsSetup contains the metrics facade and Prometheus registry for default setup.
type DefaultMetricsSetup struct {
	Metrics      mt.MetricsFacade
	Registry     *prometheus.Registry
	provider     *sdkmetric.MeterProvider
	shutdownOnce sync.Once
	serviceName  string
}

// ServiceName returns the service name used in the setup.
func (s *DefaultMetricsSetup) ServiceName() string {
	return s.serviceName
}

// Shutdown gracefully shuts down the metrics provider.
func (s *DefaultMetricsSetup) Shutdown() error {
	var err error
	s.shutdownOnce.Do(func() {
		if s.provider != nil {
			err = s.provider.Shutdown(context.Background())
		}
	})
	return err
}

// Handler returns a ready-to-use HTTP handler for the /metrics endpoint.
// Uses default Prometheus handler options.
func (s *DefaultMetricsSetup) Handler() http.Handler {
	return promhttp.HandlerFor(s.Registry, promhttp.HandlerOpts{})
}

// HandlerWithOpts returns an HTTP handler with custom Prometheus options.
// Use this when you need to customize behavior like timeout, error handling,
// compression, or instrumentation.
func (s *DefaultMetricsSetup) HandlerWithOpts(opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(s.Registry, opts)
}

// NewDefaultMetrics creates a metrics setup with Prometheus exporter.
func NewDefaultMetrics(cfg TOTVSMetricsConfig) (*DefaultMetricsSetup, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid TOTVS configuration: %w", err)
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

	// Create meter
	meter := provider.Meter(cfg.ServiceName)

	// default labels
	defaultLabels := []mt.Attribute{
		mt.Attr("platform", cfg.Platform),
	}

	// Create metrics facade with TOTVS labels
	metrics := backend.NewMetricsWithAttributes(meter, defaultLabels)

	return &DefaultMetricsSetup{
		Metrics:     metrics,
		Registry:    registry,
		provider:    provider,
		serviceName: cfg.ServiceName,
	}, nil
}

// NewMetricsWithRegistry creates a metrics setup using an existing Prometheus registry.
// Use when integrating with apps that already have a registry (e.g., kuebbuilder controller-runtime).
//
// Example:
//
//	setup, err := adapter.NewMetricsWithRegistry(
//	    ctrlmetrics.Registry,
//	    adapter.TOTVSMetricsConfig{ServiceName: "my-operator", Platform: "totvs.apps"},
//	)
func NewMetricsWithRegistry(registry prometheus.Registerer, cfg TOTVSMetricsConfig) (*DefaultMetricsSetup, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid TOTVS configuration: %w", err)
	}

	exporter, err := otelprom.New(
		otelprom.WithRegisterer(registry),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	meter := provider.Meter(cfg.ServiceName)

	defaultLabels := []mt.Attribute{
		mt.Attr("platform", cfg.Platform),
	}

	metrics := backend.NewMetricsWithAttributes(meter, defaultLabels)

	var reg *prometheus.Registry
	if r, ok := registry.(*prometheus.Registry); ok {
		reg = r
	}

	return &DefaultMetricsSetup{
		Metrics:     metrics,
		Registry:    reg,
		provider:    provider,
		serviceName: cfg.ServiceName,
	}, nil
}

// NewMetricsWithProvider delegates to the internal OpenTelemetry backend with a custom provider.
func NewMetricsWithProvider(provider metric.MeterProvider, serviceName string) mt.MetricsFacade {
	return backend.NewMetricsWithProvider(provider, serviceName)
}

// NewMetricsWithAttributes creates a MetricsFacade with base attributes that will be
// applied to all metrics created from this instance.
func NewMetricsWithAttributes(meter metric.Meter, attrs []mt.Attribute) mt.MetricsFacade {
	return backend.NewMetricsWithAttributes(meter, attrs)
}
