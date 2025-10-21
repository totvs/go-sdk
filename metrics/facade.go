package metrics

import (
	"context"
	"sync/atomic"
)

// Public types and interfaces

// Attribute represents a key-value pair for metric attributes.
type Attribute struct {
	Key   string
	Value any
}

type MetricType string

const (
	MetricTypeTech     MetricType = "tech"
	MetricTypeBusiness MetricType = "bus"
)

// MetricClass defines the aggregation scope for metrics.
type MetricClass string

const (
	MetricClassInstance MetricClass = "instance" // Metrics scoped to individual instances
	MetricClassService  MetricClass = "service"  // Metrics aggregated at service level
)

// Counter is a metric that only increases.
type Counter interface {
	Add(ctx context.Context, incr int64, attrs ...Attribute)
	Inc(ctx context.Context, attrs ...Attribute)
}

// Gauge is a metric that can increase or decrease.
type Gauge interface {
	Set(ctx context.Context, value float64, attrs ...Attribute)
	Add(ctx context.Context, incr float64, attrs ...Attribute)
}

// Histogram records distributions of values.
type Histogram interface {
	Record(ctx context.Context, value float64, attrs ...Attribute)
}

// MetricsFacade é a abstração pública para métricas usada pela aplicação.
// Implementações podem usar OpenTelemetry ou qualquer outra biblioteca no futuro.
type MetricsFacade interface {
	WithAttributes(attrs ...Attribute) MetricsFacade
	WithAttributesFromContext(ctx context.Context) MetricsFacade
	GetOrCreateCounter(name string, metricType MetricType, metricClass MetricClass) Counter
	GetOrCreateGauge(name string, metricType MetricType, metricClass MetricClass) Gauge
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
