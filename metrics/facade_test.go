package metrics_test

import (
	"context"
	"sync"
	"testing"

	mt "github.com/totvs/go-sdk/metrics"
	"github.com/totvs/go-sdk/metrics/adapter"
)

func TestMetricsBasicOperations(t *testing.T) {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "test-service", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	// Test counter creation and operations
	counter := setup.Metrics.GetOrCreateCounter("test_counter", mt.MetricTypeTech, mt.MetricClassService)
	if counter == nil {
		t.Fatal("expected counter to be created")
	}

	ctx := context.Background()
	counter.Inc(ctx, mt.Attr("label", "value"))
	counter.Add(ctx, 5, mt.Attr("status", "success"))

	// Test gauge creation and operations
	gauge := setup.Metrics.GetOrCreateGauge("test_gauge", mt.MetricTypeTech, mt.MetricClassService)
	if gauge == nil {
		t.Fatal("expected gauge to be created")
	}

	gauge.Set(ctx, 100.5, mt.Attr("type", "memory"))
	gauge.Add(ctx, 25.0, mt.Attr("operation", "increase"))

	// Test histogram creation and operations
	histogram := setup.Metrics.GetOrCreateHistogram("test_histogram", mt.MetricTypeTech, mt.MetricClassService)
	if histogram == nil {
		t.Fatal("expected histogram to be created")
	}

	histogram.Record(ctx, 0.123, mt.Attr("endpoint", "/api/test"))
	histogram.Record(ctx, 0.456, mt.Attr("endpoint", "/api/test"))
}

func TestWithAttributes(t *testing.T) {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "test-service", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	// Test adding attributes to metrics facade
	baseAttrs := []mt.Attribute{
		mt.Attr("service", "test-service"),
		mt.Attr("version", "1.0.0"),
	}

	metricsWithAttrs := setup.Metrics.WithAttributes(baseAttrs...)
	if metricsWithAttrs == nil {
		t.Fatal("expected metrics with attributes to be created")
	}

	// Create metric with base attributes
	counter := metricsWithAttrs.GetOrCreateCounter("test_counter_with_attrs", mt.MetricTypeTech, mt.MetricClassService)
	ctx := context.Background()

	// Add additional attributes when recording
	counter.Inc(ctx, mt.Attr("operation", "test"))
	counter.Add(ctx, 3, mt.Attr("status", "ok"))
}

func TestGlobalMetricsAndFromContext(t *testing.T) {
	// Test global metrics setup
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "global-test-service", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	// Set global metrics
	mt.SetGlobal(setup.Metrics)

	// Use global shortcuts
	globalCounter := mt.NewCounter("global_counter", mt.MetricTypeTech, mt.MetricClassService)
	if globalCounter == nil {
		t.Fatal("expected global counter to be created")
	}

	globalGauge := mt.NewGauge("global_gauge", mt.MetricTypeTech, mt.MetricClassService)
	if globalGauge == nil {
		t.Fatal("expected global gauge to be created")
	}

	globalHistogram := mt.NewHistogram("global_histogram", mt.MetricTypeTech, mt.MetricClassService)
	if globalHistogram == nil {
		t.Fatal("expected global histogram to be created")
	}

	// Test context injection and extraction
	ctx := mt.ContextWithMetrics(context.Background(), setup.Metrics)
	if _, ok := mt.MetricsFromContext(ctx); !ok {
		t.Fatal("expected metrics in context")
	}

	ctxMetrics := mt.FromContext(ctx)
	if ctxMetrics == nil {
		t.Fatal("expected metrics from context")
	}

	// Create metric using context metrics
	ctxCounter := ctxMetrics.GetOrCreateCounter("ctx_counter", mt.MetricTypeTech, mt.MetricClassService)
	ctxCounter.Inc(context.Background(), mt.Attr("source", "context"))

	// Test nil context should not panic and should return false
	if _, ok := mt.MetricsFromContext(nil); ok {
		t.Fatal("expected no metrics from nil context")
	}
}

func TestWithAttributesFromContext(t *testing.T) {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "test-service", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	ctx := context.Background()

	// Test extracting attributes from context (currently returns self)
	metricsWithCtx := setup.Metrics.WithAttributesFromContext(ctx)
	if metricsWithCtx == nil {
		t.Fatal("expected metrics with context attributes")
	}

	// Should be able to create metrics normally
	counter := metricsWithCtx.GetOrCreateCounter("ctx_attr_counter", mt.MetricTypeTech, mt.MetricClassService)
	counter.Inc(ctx, mt.Attr("test", "value"))
}

func TestAttrHelper(t *testing.T) {
	attr := mt.Attr("key", "value")
	if attr.Key != "key" {
		t.Fatalf("expected key 'key', got: %s", attr.Key)
	}
	if attr.Value != "value" {
		t.Fatalf("expected value 'value', got: %s", attr.Value)
	}
}

func TestMetricReuseWithSameParams(t *testing.T) {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "test-service", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	// Create same metric twice - should reuse
	counter1 := setup.Metrics.GetOrCreateCounter("reuse_counter", mt.MetricTypeTech, mt.MetricClassService)
	counter2 := setup.Metrics.GetOrCreateCounter("reuse_counter", mt.MetricTypeTech, mt.MetricClassService)

	// Should be the same instance (implementation detail, but important for efficiency)
	ctx := context.Background()
	counter1.Inc(ctx, mt.Attr("test", "1"))
	counter2.Inc(ctx, mt.Attr("test", "2"))

	// Both operations should work without issues
}

func TestConcurrentAccess(t *testing.T) {
	setup, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "concurrent-test", Platform: "totvs.apps"})
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}
	defer setup.Shutdown()

	// Test concurrent counter operations
	counter := setup.Metrics.GetOrCreateCounter("concurrent_counter", mt.MetricTypeTech, mt.MetricClassService)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			counter.Inc(ctx, mt.Attr("worker_id", string(rune(id))))
			counter.Add(ctx, 1, mt.Attr("operation", "add"))
		}(i)
	}

	wg.Wait()

	// Test concurrent gauge operations
	gauge := setup.Metrics.GetOrCreateGauge("concurrent_gauge", mt.MetricTypeTech, mt.MetricClassService)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			ctx := context.Background()
			gauge.Set(ctx, val, mt.Attr("set_id", "test"))
			gauge.Add(ctx, 1.0, mt.Attr("add_id", "test"))
		}(float64(i))
	}

	wg.Wait()

	// Test concurrent histogram operations
	histogram := setup.Metrics.GetOrCreateHistogram("concurrent_histogram", mt.MetricTypeTech, mt.MetricClassService)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			ctx := context.Background()
			histogram.Record(ctx, val/1000, mt.Attr("duration_id", "test"))
		}(float64(i))
	}

	wg.Wait()
}

func TestNoOpFallbacks(t *testing.T) {
	// Test that GetGlobal returns a safe no-op when nothing is set
	global := mt.GetGlobal()
	if global == nil {
		t.Fatal("expected non-nil global metrics")
	}

	// Should not panic when using no-op metrics
	ctx := context.Background()
	counter := global.GetOrCreateCounter("noop_counter", mt.MetricTypeTech, mt.MetricClassService)
	counter.Inc(ctx, mt.Attr("test", "noop"))

	gauge := global.GetOrCreateGauge("noop_gauge", mt.MetricTypeTech, mt.MetricClassService)
	gauge.Set(ctx, 100, mt.Attr("test", "noop"))

	histogram := global.GetOrCreateHistogram("noop_histogram", mt.MetricTypeTech, mt.MetricClassService)
	histogram.Record(ctx, 0.1, mt.Attr("test", "noop"))
}