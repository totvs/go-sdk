// Package metrics provides a metrics abstraction (facade) for application instrumentation.
// Implementations can use OpenTelemetry, Prometheus, or other backends without
// coupling consumers to specific libraries.
//
// Quick start:
//
//	setup, _ := adapter.NewPrometheusMetrics("my-service")
//	counter := setup.Metrics.GetOrCreateCounter("requests_total", MetricTypeTech, MetricClassService)
//	counter.Inc(ctx)
//
// Thread safety: All operations are safe for concurrent use.
package metrics

import (
	"context"
	"sync/atomic"
)

// Attribute represents a key-value pair for metric attributes (labels).
type Attribute struct {
	Key   string
	Value any
}

// MetricType defines the destination/purpose of a metric.
type MetricType string

const (
	// MetricTypeTech indicates technical metrics (Prometheus only).
	MetricTypeTech MetricType = "tech"
	// MetricTypeBusiness indicates business metrics (sent to Carol).
	MetricTypeBusiness MetricType = "bus"
)

// MetricClass defines the aggregation scope for metrics.
type MetricClass string

const (
	// MetricClassInstance scopes metrics to individual instances (e.g., per-pod CPU).
	MetricClassInstance MetricClass = "instance"
	// MetricClassService scopes metrics to the service level (aggregated).
	MetricClassService MetricClass = "service"
)

// Counter is a metric that only increases (e.g., request count, errors).
type Counter interface {
	// Add increments the counter by the given non-negative value.
	Add(ctx context.Context, incr int64, attrs ...Attribute)
	// Inc increments the counter by 1.
	Inc(ctx context.Context, attrs ...Attribute)
}

// Gauge is a metric that can increase or decrease (e.g., memory usage, connections).
type Gauge interface {
	// Set records an absolute value for the gauge.
	Set(ctx context.Context, value float64, attrs ...Attribute)
	// Add is provided for interface compatibility but has a KNOWN LIMITATION:
	// the OpenTelemetry backend does NOT support atomic add operations.
	// It records the increment value directly, NOT adding to current value.
	// Use Set() with the computed absolute value for correct behavior.
	//
	// Deprecated: Use Set() with the computed absolute value instead.
	Add(ctx context.Context, incr float64, attrs ...Attribute)
}

// Histogram records distributions of values (e.g., latency, request sizes).
type Histogram interface {
	// Record adds a value to the histogram distribution.
	Record(ctx context.Context, value float64, attrs ...Attribute)
}

// MetricsFacade is the public abstraction for metrics used by applications.
// Implementations can use OpenTelemetry or other libraries.
// All methods are safe for concurrent use.
type MetricsFacade interface {
	// WithAttributes returns a new facade with additional base attributes.
	WithAttributes(attrs ...Attribute) MetricsFacade
	// WithAttributesFromContext extracts attributes from context (e.g., trace ID).
	WithAttributesFromContext(ctx context.Context) MetricsFacade
	// GetOrCreateCounter returns an existing counter or creates a new one.
	GetOrCreateCounter(name string, metricType MetricType, metricClass MetricClass) Counter
	// GetOrCreateGauge returns an existing gauge or creates a new one.
	GetOrCreateGauge(name string, metricType MetricType, metricClass MetricClass) Gauge
	// GetOrCreateHistogram returns an existing histogram or creates a new one.
	GetOrCreateHistogram(name string, metricType MetricType, metricClass MetricClass) Histogram
}

// Public helper functions

// Attr creates an attribute conveniently.
func Attr(key string, value any) Attribute {
	return Attribute{Key: key, Value: value}
}

// Context keys and global storage

// ctxKey is used for storing values in context without colliding with other packages.
type ctxKey string

const metricsKey ctxKey = "metrics"

// globalMetrics stores the package-level metrics in an atomic.Value to make
// reads/writes safe for concurrent access. Use SetGlobal/GetGlobal to access.
var globalMetrics atomic.Value

// storedMetrics is a stable concrete type stored in the atomic.Value. We keep
// a pointer to this type so we always store the same concrete type, avoiding
// atomic.Value panics when swapping implementations at runtime.
type storedMetrics struct{ mf MetricsFacade }

// Global management functions

// SetGlobal replaces the package-level global metrics.
func SetGlobal(m MetricsFacade) { globalMetrics.Store(&storedMetrics{mf: m}) }

// GetGlobal returns the package-level global metrics. If none is configured
// yet, it stores and returns a no-op metrics to avoid nil panics.
func GetGlobal() MetricsFacade {
	if v := globalMetrics.Load(); v != nil {
		if sm, ok := v.(*storedMetrics); ok {
			if sm.mf != nil {
				return sm.mf
			}
		}
	}
	def := &storedMetrics{mf: nopMetrics{}}
	globalMetrics.Store(def)
	return def.mf
}

// Package-level helpers for global metrics

// NewCounter creates a Counter using the global metrics.
func NewCounter(name string, metricType MetricType, metricClass MetricClass) Counter {
	return GetGlobal().GetOrCreateCounter(name, metricType, metricClass)
}

// NewGauge creates a Gauge using the global metrics.
func NewGauge(name string, metricType MetricType, metricClass MetricClass) Gauge {
	return GetGlobal().GetOrCreateGauge(name, metricType, metricClass)
}

// NewHistogram creates a Histogram using the global metrics.
func NewHistogram(name string, metricType MetricType, metricClass MetricClass) Histogram {
	return GetGlobal().GetOrCreateHistogram(name, metricType, metricClass)
}

// Context functions

// ContextWithMetrics stores a MetricsFacade in the context so callers can inject
// a metrics instance (facade) that will be used by library code.
func ContextWithMetrics(ctx context.Context, m MetricsFacade) context.Context {
	return context.WithValue(ctx, metricsKey, m)
}

// MetricsFromContext extracts a MetricsFacade from the context. The boolean indicates whether metrics was present.
// The context value must be a MetricsFacade.
func MetricsFromContext(ctx context.Context) (MetricsFacade, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(metricsKey); v != nil {
		if mf, ok := v.(MetricsFacade); ok {
			return mf, true
		}
	}
	return nil, false
}

// FromContext returns a MetricsFacade extracted from the context if present; otherwise returns the global metrics.
func FromContext(ctx context.Context) MetricsFacade {
	if mf, ok := MetricsFromContext(ctx); ok {
		return mf
	}
	return GetGlobal()
}

// nop implementations used as safe fallbacks when no global metrics is set.

type nopCounter struct{}

func (nopCounter) Add(ctx context.Context, incr int64, attrs ...Attribute) {}
func (nopCounter) Inc(ctx context.Context, attrs ...Attribute)             {}

type nopGauge struct{}

func (nopGauge) Set(ctx context.Context, value float64, attrs ...Attribute) {}
func (nopGauge) Add(ctx context.Context, incr float64, attrs ...Attribute)  {}

type nopHistogram struct{}

func (nopHistogram) Record(ctx context.Context, value float64, attrs ...Attribute) {}

type nopMetrics struct{}

func (nopMetrics) WithAttributes(attrs ...Attribute) MetricsFacade             { return nopMetrics{} }
func (nopMetrics) WithAttributesFromContext(ctx context.Context) MetricsFacade { return nopMetrics{} }
func (nopMetrics) GetOrCreateCounter(name string, metricType MetricType, metricClass MetricClass) Counter {
	return nopCounter{}
}
func (nopMetrics) GetOrCreateGauge(name string, metricType MetricType, metricClass MetricClass) Gauge {
	return nopGauge{}
}
func (nopMetrics) GetOrCreateHistogram(name string, metricType MetricType, metricClass MetricClass) Histogram {
	return nopHistogram{}
}
