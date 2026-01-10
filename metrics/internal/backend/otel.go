package backend

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	mt "github.com/totvs/go-sdk/metrics"
)

// otelCounter wraps an OpenTelemetry counter
type otelCounter struct {
	counter metric.Int64Counter
	attrs   []attribute.KeyValue
}

func (c *otelCounter) Add(ctx context.Context, incr int64, attrs ...mt.Attribute) {
	if c.counter == nil {
		return // no-op if counter creation failed
	}
	combinedAttrs := combineAttributes(c.attrs, attrs)
	c.counter.Add(ctx, incr, metric.WithAttributes(combinedAttrs...))
}

func (c *otelCounter) Inc(ctx context.Context, attrs ...mt.Attribute) {
	c.Add(ctx, 1, attrs...)
}

// otelGauge wraps an OpenTelemetry gauge
type otelGauge struct {
	gauge metric.Float64Gauge
	attrs []attribute.KeyValue
}

func (g *otelGauge) Set(ctx context.Context, value float64, attrs ...mt.Attribute) {
	if g.gauge == nil {
		return // no-op if gauge creation failed
	}
	combinedAttrs := combineAttributes(g.attrs, attrs)
	g.gauge.Record(ctx, value, metric.WithAttributes(combinedAttrs...))
}

// Add is provided for interface compatibility but has a KNOWN LIMITATION:
// OpenTelemetry gauges do not support atomic add operations. This method
// records the increment value directly, NOT adding to the current value.
// For proper gauge semantics, use Set() with the absolute value instead.
//
// Deprecated: Use Set() with the computed absolute value for correct behavior.
func (g *otelGauge) Add(ctx context.Context, incr float64, attrs ...mt.Attribute) {
	if g.gauge == nil {
		return // no-op if gauge creation failed
	}
	// WARNING: This does NOT add to current value - it records incr as-is.
	// OTel gauges are "last value wins" and don't support atomic increments.
	combinedAttrs := combineAttributes(g.attrs, attrs)
	g.gauge.Record(ctx, incr, metric.WithAttributes(combinedAttrs...))
}

// otelHistogram wraps an OpenTelemetry histogram
type otelHistogram struct {
	histogram metric.Float64Histogram
	attrs     []attribute.KeyValue
}

func (h *otelHistogram) Record(ctx context.Context, value float64, attrs ...mt.Attribute) {
	if h.histogram == nil {
		return // no-op if histogram creation failed
	}
	combinedAttrs := combineAttributes(h.attrs, attrs)
	h.histogram.Record(ctx, value, metric.WithAttributes(combinedAttrs...))
}

// implMetrics is the concrete metrics implementation based on OpenTelemetry.
type implMetrics struct {
	meter      metric.Meter
	attrs      []attribute.KeyValue
	counters   sync.Map // map[string]*otelCounter
	gauges     sync.Map // map[string]*otelGauge
	histograms sync.Map // map[string]*otelHistogram
}

// newMetrics creates a metrics implementation with the provided meter and attributes.
func newMetrics(meter metric.Meter, attrs []attribute.KeyValue) mt.MetricsFacade {
	return &implMetrics{
		meter: meter,
		attrs: attrs,
	}
}

func (m *implMetrics) WithAttributes(attrs ...mt.Attribute) mt.MetricsFacade {
	combinedAttrs := combineAttributes(m.attrs, attrs)
	return &implMetrics{
		meter: m.meter,
		attrs: combinedAttrs,
	}
}

func (m *implMetrics) WithAttributesFromContext(ctx context.Context) mt.MetricsFacade {
	// Extract trace ID if available (similar to log package)
	// For now, just return self as we don't have trace integration yet
	return m
}

// buildMetricAttrs creates attributes with metric_type and metric_class
func (m *implMetrics) buildMetricAttrs(metricType mt.MetricType, metricClass mt.MetricClass) []attribute.KeyValue {
	return append(m.attrs,
		attribute.String("metric_type", string(metricType)),
		attribute.String("metric_class", string(metricClass)),
	)
}

// buildKey creates a cache key for a metric
func buildKey(metricKind, name string, metricType mt.MetricType, metricClass mt.MetricClass) string {
	return fmt.Sprintf("%s:%s:%s:%s", metricKind, name, metricType, metricClass)
}

// getOrCreate is a generic helper for the cache-and-create pattern used by all metric types.
// It checks the cache first, and if the metric doesn't exist, calls the creator function
// to instantiate it, stores it in the cache, and returns it.
// Uses LoadOrStore for atomic cache operations to prevent race conditions.
func getOrCreate[T any](cache *sync.Map, key string, creator func() (T, error)) T {
	// Check if metric already exists in cache (read-optimized, no lock)
	if cached, ok := cache.Load(key); ok {
		return cached.(T) // Return existing singleton instance
	}

	// Cache miss: create new metric instance lazily
	newMetric, err := creator()
	if err != nil {
		// Log error to stderr - we don't have access to the log package here
		// to avoid circular dependencies
		fmt.Fprintf(os.Stderr, "[metrics] failed to create metric %s: %v\n", key, err)
		var zero T  // Initialize zero value of type T
		return zero // Return no-op metric on error
	}

	// Use LoadOrStore for atomic operation - if another goroutine stored first,
	// we return their value and discard ours (safe because metrics are idempotent)
	actual, loaded := cache.LoadOrStore(key, newMetric)
	if loaded {
		return actual.(T) // Another goroutine won the race, use their instance
	}
	return newMetric
}

func (m *implMetrics) GetOrCreateCounter(name string, metricType mt.MetricType, metricClass mt.MetricClass) mt.Counter {
	key := buildKey("counter", name, metricType, metricClass)
	return getOrCreate(&m.counters, key, func() (*otelCounter, error) {
		counter, err := m.meter.Int64Counter(name)
		if err != nil {
			return &otelCounter{}, err
		}
		return &otelCounter{
			counter: counter,
			attrs:   m.buildMetricAttrs(metricType, metricClass),
		}, nil
	})
}

func (m *implMetrics) GetOrCreateGauge(name string, metricType mt.MetricType, metricClass mt.MetricClass) mt.Gauge {
	key := buildKey("gauge", name, metricType, metricClass)
	return getOrCreate(&m.gauges, key, func() (*otelGauge, error) {
		gauge, err := m.meter.Float64Gauge(name)
		if err != nil {
			return &otelGauge{}, err
		}
		return &otelGauge{
			gauge: gauge,
			attrs: m.buildMetricAttrs(metricType, metricClass),
		}, nil
	})
}

func (m *implMetrics) GetOrCreateHistogram(name string, metricType mt.MetricType, metricClass mt.MetricClass) mt.Histogram {
	key := buildKey("histogram", name, metricType, metricClass)
	return getOrCreate(&m.histograms, key, func() (*otelHistogram, error) {
		histogram, err := m.meter.Float64Histogram(name)
		if err != nil {
			return &otelHistogram{}, err
		}
		return &otelHistogram{
			histogram: histogram,
			attrs:     m.buildMetricAttrs(metricType, metricClass),
		}, nil
	})
}

// convertAttribute converts a single mt.Attribute to OTEL attribute.KeyValue
func convertAttribute(attr mt.Attribute) attribute.KeyValue {
	switch v := attr.Value.(type) {
	case string:
		return attribute.String(attr.Key, v)
	case int:
		return attribute.Int(attr.Key, v)
	case int64:
		return attribute.Int64(attr.Key, v)
	case float64:
		return attribute.Float64(attr.Key, v)
	case bool:
		return attribute.Bool(attr.Key, v)
	default:
		return attribute.String(attr.Key, fmt.Sprintf("%v", v))
	}
}

// combineAttributes merges base attributes with additional ones
func combineAttributes(base []attribute.KeyValue, additional []mt.Attribute) []attribute.KeyValue {
	result := make([]attribute.KeyValue, len(base)+len(additional))
	copy(result, base)

	for i, attr := range additional {
		result[len(base)+i] = convertAttribute(attr)
	}

	return result
}

// NewMetrics creates a MetricsFacade based on OpenTelemetry with the provided meter.
func NewMetrics(meter metric.Meter) mt.MetricsFacade {
	return newMetrics(meter, nil)
}

// NewMetricsWithProvider creates a MetricsFacade using a custom MeterProvider.
func NewMetricsWithProvider(provider metric.MeterProvider, serviceName string) mt.MetricsFacade {
	meter := provider.Meter(serviceName)
	return newMetrics(meter, nil)
}

// NewMetricsWithAttributes creates a MetricsFacade with base attributes that will be
// applied to all metrics created from this instance.
func NewMetricsWithAttributes(meter metric.Meter, attrs []mt.Attribute) mt.MetricsFacade {
	otelAttrs := make([]attribute.KeyValue, len(attrs))
	for i, attr := range attrs {
		otelAttrs[i] = convertAttribute(attr)
	}
	return newMetrics(meter, otelAttrs)
}
