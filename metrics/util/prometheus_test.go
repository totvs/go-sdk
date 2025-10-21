package util_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	mt "github.com/totvs/go-sdk/metrics"
	"github.com/totvs/go-sdk/metrics/adapter"
)

func TestSetupPrometheusMetrics(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("test-service")
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}
	if setup == nil {
		t.Fatal("expected non-nil prometheus setup")
	}

	// Verify components are created
	if setup.Metrics == nil {
		t.Fatal("expected metrics facade to be created")
	}
	if setup.Registry == nil {
		t.Fatal("expected prometheus registry to be created")
	}
	if setup.ServiceName() != "test-service" {
		t.Fatalf("expected service name 'test-service', got: %s", setup.ServiceName())
	}

	// Test cleanup
	err = setup.Shutdown()
	if err != nil {
		t.Fatalf("failed to shutdown: %v", err)
	}
}

func TestPrometheusMetricsEndpoint(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("endpoint-test")
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}
	defer setup.Shutdown()

	// Create some metrics
	counter := setup.Metrics.GetOrCreateCounter("test_requests_total", mt.MetricTypeTech, mt.MetricClassService)
	gauge := setup.Metrics.GetOrCreateGauge("test_memory_bytes", mt.MetricTypeTech, mt.MetricClassService)
	histogram := setup.Metrics.GetOrCreateHistogram("test_duration_seconds", mt.MetricTypeTech, mt.MetricClassService)

	ctx := context.Background()

	// Record some values
	counter.Inc(ctx, mt.Attr("method", "GET"), mt.Attr("status", "200"))
	counter.Add(ctx, 5, mt.Attr("method", "POST"), mt.Attr("status", "201"))

	gauge.Set(ctx, 1024*1024, mt.Attr("type", "heap"))
	gauge.Add(ctx, 512, mt.Attr("type", "stack"))

	histogram.Record(ctx, 0.123, mt.Attr("endpoint", "/api/test"))
	histogram.Record(ctx, 0.456, mt.Attr("endpoint", "/api/users"))

	// Create HTTP server with metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	// Request metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	output := string(body)

	// Verify metrics are present in Prometheus format
	expectedMetrics := []string{
		"test_requests_total",
		"test_memory_bytes",
		"test_duration_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Fatalf("expected metric %s in output, got: %s", metric, output)
		}
	}

	// Verify labels are present
	expectedLabels := []string{
		`method="GET"`,
		`status="200"`,
		`type="heap"`,
		`endpoint="/api/test"`,
	}

	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Fatalf("expected label %s in output, got: %s", label, output)
		}
	}
}

func TestMultipleShutdowns(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("shutdown-test")
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}

	// First shutdown should work
	err1 := setup.Shutdown()
	if err1 != nil {
		t.Fatalf("first shutdown failed: %v", err1)
	}

	// Second shutdown should also work (idempotent)
	err2 := setup.Shutdown()
	if err2 != nil {
		t.Fatalf("second shutdown failed: %v", err2)
	}

	// Third shutdown should still work
	err3 := setup.Shutdown()
	if err3 != nil {
		t.Fatalf("third shutdown failed: %v", err3)
	}
}

func TestConcurrentShutdown(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("concurrent-shutdown-test")
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Attempt concurrent shutdowns
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := setup.Shutdown(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// All shutdowns should succeed (or at least not error)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent shutdown failed: %v", err)
		}
	}
}

func TestRegistryIsolation(t *testing.T) {
	// Create two separate setups
	setup1, err := adapter.NewPrometheusMetrics("service-1")
	if err != nil {
		t.Fatalf("failed to setup first metrics: %v", err)
	}
	defer setup1.Shutdown()

	setup2, err := adapter.NewPrometheusMetrics("service-2")
	if err != nil {
		t.Fatalf("failed to setup second metrics: %v", err)
	}
	defer setup2.Shutdown()

	// Verify they have different registries
	if setup1.Registry == setup2.Registry {
		t.Fatal("expected different registries for different setups")
	}

	// Create metrics in each
	counter1 := setup1.Metrics.GetOrCreateCounter("isolation_test", mt.MetricTypeTech, mt.MetricClassService)
	counter2 := setup2.Metrics.GetOrCreateCounter("isolation_test", mt.MetricTypeTech, mt.MetricClassService)

	ctx := context.Background()
	counter1.Inc(ctx, mt.Attr("service", "service-1"))
	counter2.Inc(ctx, mt.Attr("service", "service-2"))

	// Each registry should only have its own metrics
	handler1 := promhttp.HandlerFor(setup1.Registry, promhttp.HandlerOpts{})
	handler2 := promhttp.HandlerFor(setup2.Registry, promhttp.HandlerOpts{})

	// Test first registry
	req1 := httptest.NewRequest("GET", "/metrics", nil)
	rec1 := httptest.NewRecorder()
	handler1.ServeHTTP(rec1, req1)

	// Test second registry
	req2 := httptest.NewRequest("GET", "/metrics", nil)
	rec2 := httptest.NewRecorder()
	handler2.ServeHTTP(rec2, req2)

	// Both should have the metric, but they should be separate
	output1 := rec1.Body.String()
	output2 := rec2.Body.String()

	if !strings.Contains(output1, "isolation_test") {
		t.Fatal("expected isolation_test in first registry output")
	}
	if !strings.Contains(output2, "isolation_test") {
		t.Fatal("expected isolation_test in second registry output")
	}
}

func TestServiceNamePersistence(t *testing.T) {
	serviceName := "persistent-service-test"
	setup, err := adapter.NewPrometheusMetrics(serviceName)
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}
	defer setup.Shutdown()

	if setup.ServiceName() != serviceName {
		t.Fatalf("expected service name %s, got: %s", serviceName, setup.ServiceName())
	}

	// Service name should persist through operations
	counter := setup.Metrics.GetOrCreateCounter("persistence_test", mt.MetricTypeTech, mt.MetricClassService)
	ctx := context.Background()
	counter.Inc(ctx, mt.Attr("test", "value"))

	if setup.ServiceName() != serviceName {
		t.Fatalf("service name changed after metric creation, expected: %s, got: %s", serviceName, setup.ServiceName())
	}
}

func TestErrorHandling(t *testing.T) {
	// Test with empty service name - should return error now (TOTVS validation)
	_, err := adapter.NewPrometheusMetrics("")
	if err == nil {
		t.Fatal("expected error with empty service name, got nil")
	}

	// Expected error message
	if !strings.Contains(err.Error(), "ServiceName is required") {
		t.Fatalf("expected 'ServiceName is required' error, got: %v", err)
	}
}

func TestMetricsAfterShutdown(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("shutdown-test")
	if err != nil {
		t.Fatalf("failed to setup prometheus metrics: %v", err)
	}

	// Create metric before shutdown
	counter := setup.Metrics.GetOrCreateCounter("shutdown_test", mt.MetricTypeTech, mt.MetricClassService)
	ctx := context.Background()
	counter.Inc(ctx, mt.Attr("stage", "before_shutdown"))

	// Shutdown
	err = setup.Shutdown()
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Should still be able to use metrics (they might become no-op or continue working)
	// This depends on the OpenTelemetry implementation, but should not panic
	counter.Inc(ctx, mt.Attr("stage", "after_shutdown"))
}