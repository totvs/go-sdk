package backend_test

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	mt "github.com/totvs/go-sdk/metrics"
	backend "github.com/totvs/go-sdk/metrics/internal/backend"
)

func TestNewMetrics(t *testing.T) {
	// Create a test meter provider
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")

	// Create metrics facade with test meter
	metrics := backend.NewMetrics(meter)
	if metrics == nil {
		t.Fatal("expected non-nil metrics facade")
	}
}

func TestNewMetricsWithProvider(t *testing.T) {
	// Create custom provider
	provider := sdkmetric.NewMeterProvider()

	// Test creating metrics with custom provider
	metrics := backend.NewMetricsWithProvider(provider, "test-service")
	if metrics == nil {
		t.Fatal("expected non-nil metrics facade")
	}
}

func TestCounterOperations(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	metrics := backend.NewMetrics(meter)

	// Create counter
	counter := metrics.GetOrCreateCounter("test_counter", mt.MetricTypeTech, mt.MetricClassService)
	if counter == nil {
		t.Fatal("expected counter to be created")
	}

	ctx := context.Background()

	// Test Inc operation
	counter.Inc(ctx, mt.Attr("operation", "inc"))

	// Test Add operation
	counter.Add(ctx, 5, mt.Attr("operation", "add"), mt.Attr("value", 5))

	// Test with multiple attributes
	attrs := []mt.Attribute{
		mt.Attr("service", "test"),
		mt.Attr("version", "1.0"),
		mt.Attr("env", "test"),
	}
	counter.Add(ctx, 10, attrs...)
}

func TestGaugeOperations(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	metrics := backend.NewMetrics(meter)

	// Create gauge
	gauge := metrics.GetOrCreateGauge("test_gauge", mt.MetricTypeTech, mt.MetricClassService)
	if gauge == nil {
		t.Fatal("expected gauge to be created")
	}

	ctx := context.Background()

	// Test Set operation
	gauge.Set(ctx, 100.5, mt.Attr("type", "memory"))

	// Test Add operation
	gauge.Add(ctx, 25.0, mt.Attr("operation", "increment"))

	// Test with negative values
	gauge.Set(ctx, -10.0, mt.Attr("type", "delta"))
	gauge.Add(ctx, -5.0, mt.Attr("operation", "decrement"))
}

func TestHistogramOperations(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	metrics := backend.NewMetrics(meter)

	// Create histogram
	histogram := metrics.GetOrCreateHistogram("test_histogram", mt.MetricTypeTech, mt.MetricClassService)
	if histogram == nil {
		t.Fatal("expected histogram to be created")
	}

	ctx := context.Background()

	// Test Record operation with various values
	testValues := []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 10.0}
	for i, value := range testValues {
		histogram.Record(ctx, value,
			mt.Attr("request_id", string(rune(i))),
			mt.Attr("endpoint", "/api/test"),
		)
	}
}

func TestMetricReuseWithSameKey(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	metrics := backend.NewMetrics(meter)

	// Create same counter twice with same parameters
	counter1 := metrics.GetOrCreateCounter("reuse_test", mt.MetricTypeTech, mt.MetricClassService)
	counter2 := metrics.GetOrCreateCounter("reuse_test", mt.MetricTypeTech, mt.MetricClassService)

	// Both should work (implementation caches them)
	ctx := context.Background()
	counter1.Inc(ctx, mt.Attr("source", "counter1"))
	counter2.Inc(ctx, mt.Attr("source", "counter2"))

	// Create gauge with same name - should reuse
	gauge1 := metrics.GetOrCreateGauge("reuse_test", mt.MetricTypeTech, mt.MetricClassService)
	gauge2 := metrics.GetOrCreateGauge("reuse_test", mt.MetricTypeTech, mt.MetricClassService)

	gauge1.Set(ctx, 100, mt.Attr("unit", "bytes"))
	gauge2.Set(ctx, 50, mt.Attr("unit", "count"))
}

func TestWithAttributes(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	baseMetrics := backend.NewMetrics(meter)

	// Add base attributes
	baseAttrs := []mt.Attribute{
		mt.Attr("service", "test-service"),
		mt.Attr("version", "1.0.0"),
	}

	metricsWithAttrs := baseMetrics.WithAttributes(baseAttrs...)
	if metricsWithAttrs == nil {
		t.Fatal("expected metrics with attributes")
	}

	// Create counter with base attributes
	counter := metricsWithAttrs.GetOrCreateCounter("attr_counter", mt.MetricTypeTech, mt.MetricClassService)
	ctx := context.Background()

	// Add additional attributes when recording
	counter.Inc(ctx, mt.Attr("operation", "test"))

	// Test chaining attributes
	moreAttrs := metricsWithAttrs.WithAttributes(mt.Attr("region", "us-east-1"))
	counter2 := moreAttrs.GetOrCreateCounter("chained_counter", mt.MetricTypeTech, mt.MetricClassService)
	counter2.Add(ctx, 5, mt.Attr("status", "success"))
}

func TestWithAttributesFromContext(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	metrics := backend.NewMetrics(meter)

	ctx := context.Background()

	// Test with context (currently returns self in implementation)
	metricsFromCtx := metrics.WithAttributesFromContext(ctx)
	if metricsFromCtx == nil {
		t.Fatal("expected metrics from context")
	}

	// Should work normally
	counter := metricsFromCtx.GetOrCreateCounter("ctx_counter", mt.MetricTypeTech, mt.MetricClassService)
	counter.Inc(ctx, mt.Attr("from_context", "true"))
}

func TestAttributeCombination(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test-service")
	baseMetrics := backend.NewMetrics(meter)

	// Test various attribute types and combinations
	attrs := []mt.Attribute{
		mt.Attr("string_attr", "value"),
		mt.Attr("number_attr", "123"),
		mt.Attr("bool_attr", "true"),
		mt.Attr("empty_attr", ""),
	}

	metricsWithAttrs := baseMetrics.WithAttributes(attrs...)
	counter := metricsWithAttrs.GetOrCreateCounter("combo_counter", mt.MetricTypeTech, mt.MetricClassService)

	ctx := context.Background()
	counter.Inc(ctx, mt.Attr("additional", "attr"))
}
