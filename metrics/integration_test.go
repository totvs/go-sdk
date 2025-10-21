package metrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	mt "github.com/totvs/go-sdk/metrics"
	"github.com/totvs/go-sdk/metrics/adapter"
	"github.com/totvs/go-sdk/metrics/util"
)

// TestIntegration groups all integration tests
func TestIntegration(t *testing.T) {

	t.Run("Prometheus", func(t *testing.T) {

		t.Run("EndToEnd", func(t *testing.T) {

			t.Run("Success", func(t *testing.T) {

				// Arrange
				setup, err := adapter.NewPrometheusMetrics("integration-test-service")
				if err != nil {
					t.Fatalf("failed to setup prometheus metrics: %v", err)
				}
				defer setup.Shutdown()

				ctx := context.Background()

				requestCounter := setup.Metrics.GetOrCreateCounter("integration_requests_total", mt.MetricTypeTech, mt.MetricClassService)
				memoryGauge := setup.Metrics.GetOrCreateGauge("integration_memory_bytes", mt.MetricTypeTech, mt.MetricClassService)
				durationHistogram := setup.Metrics.GetOrCreateHistogram("integration_duration_seconds", mt.MetricTypeTech, mt.MetricClassService)

				endpoints := []string{"/api/users", "/api/orders", "/api/products"}
				methods := []string{"GET", "POST", "PUT", "DELETE"}
				statuses := []string{"200", "201", "400", "404", "500"}

				for i := 0; i < 50; i++ {
					endpoint := endpoints[i%len(endpoints)]
					method := methods[i%len(methods)]
					status := statuses[i%len(statuses)]

					requestCounter.Inc(ctx,
						mt.Attr("endpoint", endpoint),
						mt.Attr("method", method),
						mt.Attr("status", status),
						mt.Attr("service", "integration-test-service"),
					)

					memoryGauge.Set(ctx, float64(1024*1024+i*1000),
						mt.Attr("type", "heap"),
						mt.Attr("process", "integration-test"),
					)

					duration := 0.01 + float64(i%100)/1000.0
					durationHistogram.Record(ctx, duration,
						mt.Attr("endpoint", endpoint),
						mt.Attr("method", method),
					)
				}

				mux := http.NewServeMux()
				mux.Handle("/metrics", promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{}))

				server := httptest.NewServer(mux)
				defer server.Close()

				time.Sleep(100 * time.Millisecond)

				// Act
				resp, err := http.Get(server.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch metrics: %v", err)
				}
				defer resp.Body.Close()

				buf := make([]byte, 64*1024)
				n, err := resp.Body.Read(buf)
				if err != nil && err.Error() != "EOF" {
					t.Fatalf("failed to read metrics response: %v", err)
				}
				output := string(buf[:n])

				// Assert
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("expected status 200, got: %d", resp.StatusCode)
				}

				if len(output) < 100 {
					t.Fatalf("expected metrics output but got very short response: %s", output)
				}

				if !strings.Contains(output, "integration_duration_seconds") {
					t.Fatalf("expected integration_duration_seconds metric in output")
				}

				expectedPatterns := []string{
					`endpoint="/api/`,
					`method="`,
				}

				for _, pattern := range expectedPatterns {
					if !strings.Contains(output, pattern) {
						t.Fatalf("expected pattern %s in output", pattern)
					}
				}

				if !strings.Contains(output, "le=") {
					t.Fatal("expected histogram buckets (le=) in output")
				}

				if !strings.Contains(output, "_bucket{") {
					t.Fatal("expected histogram bucket metrics in output")
				}

			})

		})

		t.Run("HTTPMiddleware", func(t *testing.T) {

			t.Run("Success", func(t *testing.T) {

				// Arrange
				setup, err := adapter.NewPrometheusMetrics("http-integration-test")
				if err != nil {
					t.Fatalf("failed to setup metrics: %v", err)
				}
				defer setup.Shutdown()

				mux := http.NewServeMux()

				mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status": "healthy"}`))
				})

				mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case "GET":
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"users": []}`))
					case "POST":
						w.WriteHeader(http.StatusCreated)
						w.Write([]byte(`{"id": 1, "name": "test"}`))
					default:
						w.WriteHeader(http.StatusMethodNotAllowed)
					}
				})

				mux.HandleFunc("/api/error", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
				})

				mux.Handle("/metrics", promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{}))

				handler := util.WithMetrics(setup.Metrics, "http-integration-test", mux)

				server := httptest.NewServer(handler)
				defer server.Close()

				testRequests := []struct {
					method string
					path   string
					status int
				}{
					{"GET", "/api/health", 200},
					{"GET", "/api/users", 200},
					{"POST", "/api/users", 201},
					{"PUT", "/api/users", 405},
					{"GET", "/api/error", 500},
					{"GET", "/api/health", 200},
				}

				// Act
				for _, req := range testRequests {
					resp, err := http.NewRequest(req.method, server.URL+req.path, nil)
					if err != nil {
						t.Fatalf("failed to create request: %v", err)
					}

					client := &http.Client{}
					response, err := client.Do(resp)
					if err != nil {
						t.Fatalf("failed to make request: %v", err)
					}
					response.Body.Close()

					if response.StatusCode != req.status {
						t.Fatalf("expected status %d for %s %s, got: %d", req.status, req.method, req.path, response.StatusCode)
					}
				}

				time.Sleep(100 * time.Millisecond)

				resp, err := http.Get(server.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch metrics: %v", err)
				}
				defer resp.Body.Close()

				buf := make([]byte, 32*1024)
				n, _ := resp.Body.Read(buf)
				output := string(buf[:n])

				// Assert
				if !strings.Contains(output, "http_requests") {
					t.Fatal("expected http_requests metric")
				}

				expectedPatterns := []string{
					`method="`,
					`path="/api/`,
					`status="`,
				}

				for _, pattern := range expectedPatterns {
					if !strings.Contains(output, pattern) {
						t.Fatalf("expected pattern %s in output", pattern)
					}
				}

			})

		})

		t.Run("MultipleServices", func(t *testing.T) {

			t.Run("Success", func(t *testing.T) {

				// Arrange
				service1, err := adapter.NewPrometheusMetrics("service-1")
				if err != nil {
					t.Fatalf("failed to setup service 1: %v", err)
				}
				defer service1.Shutdown()

				service2, err := adapter.NewPrometheusMetrics("service-2")
				if err != nil {
					t.Fatalf("failed to setup service 2: %v", err)
				}
				defer service2.Shutdown()

				ctx := context.Background()

				counter1 := service1.Metrics.GetOrCreateCounter("multi_service_counter", mt.MetricTypeTech, mt.MetricClassService)
				counter2 := service2.Metrics.GetOrCreateCounter("multi_service_counter", mt.MetricTypeTech, mt.MetricClassService)

				counter1.Add(ctx, 10, mt.Attr("service", "service-1"), mt.Attr("type", "requests"))
				counter2.Add(ctx, 20, mt.Attr("service", "service-2"), mt.Attr("type", "requests"))

				mux1 := http.NewServeMux()
				mux1.Handle("/metrics", promhttp.HandlerFor(service1.Registry, promhttp.HandlerOpts{}))
				server1 := httptest.NewServer(mux1)
				defer server1.Close()

				mux2 := http.NewServeMux()
				mux2.Handle("/metrics", promhttp.HandlerFor(service2.Registry, promhttp.HandlerOpts{}))
				server2 := httptest.NewServer(mux2)
				defer server2.Close()

				// Act
				resp1, err := http.Get(server1.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch service 1 metrics: %v", err)
				}
				defer resp1.Body.Close()

				resp2, err := http.Get(server2.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch service 2 metrics: %v", err)
				}
				defer resp2.Body.Close()

				buf1 := make([]byte, 16*1024)
				n1, _ := resp1.Body.Read(buf1)
				output1 := string(buf1[:n1])

				buf2 := make([]byte, 16*1024)
				n2, _ := resp2.Body.Read(buf2)
				output2 := string(buf2[:n2])

				// Assert
				if !strings.Contains(output1, `service="service-1"`) {
					t.Fatal("expected service-1 label in service 1 output")
				}

				if strings.Contains(output1, `service="service-2"`) {
					t.Fatal("unexpected service-2 label in service 1 output")
				}

				if !strings.Contains(output2, `service="service-2"`) {
					t.Fatal("expected service-2 label in service 2 output")
				}

				if strings.Contains(output2, `service="service-1"`) {
					t.Fatal("unexpected service-1 label in service 2 output")
				}

			})

		})

		t.Run("GracefulShutdown", func(t *testing.T) {

			t.Run("Success", func(t *testing.T) {

				// Arrange
				setup, err := adapter.NewPrometheusMetrics("shutdown-integration-test")
				if err != nil {
					t.Fatalf("failed to setup metrics: %v", err)
				}

				ctx := context.Background()
				counter := setup.Metrics.GetOrCreateCounter("shutdown_test_counter", mt.MetricTypeTech, mt.MetricClassService)

				counter.Inc(ctx, mt.Attr("phase", "before_shutdown"))

				mux := http.NewServeMux()
				mux.Handle("/metrics", promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{}))
				server := httptest.NewServer(mux)
				defer server.Close()

				resp, err := http.Get(server.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch metrics before shutdown: %v", err)
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Fatalf("expected 200 before shutdown, got: %d", resp.StatusCode)
				}

				// Act
				err = setup.Shutdown()
				if err != nil {
					t.Fatalf("failed to shutdown: %v", err)
				}

				counter.Inc(ctx, mt.Attr("phase", "after_shutdown"))

				resp2, err := http.Get(server.URL + "/metrics")
				if err != nil {
					t.Fatalf("failed to fetch metrics after shutdown: %v", err)
				}
				defer resp2.Body.Close()

				// Assert
				if resp2.StatusCode != http.StatusOK {
					t.Fatalf("expected 200 after shutdown, got: %d", resp2.StatusCode)
				}

			})

		})

		t.Run("AdapterCompatibility", func(t *testing.T) {

			t.Run("Success", func(t *testing.T) {

				// Arrange
				setup1, err := adapter.NewDefaultMetrics(adapter.TOTVSMetricsConfig{ServiceName: "adapter-test-1", Platform: "totvs.apps"})
				if err != nil {
					t.Fatalf("failed to create adapter metrics: %v", err)
				}
				defer setup1.Shutdown()

				setup2, err := adapter.NewPrometheusMetrics("adapter-test-2")
				if err != nil {
					t.Fatalf("failed to create util metrics: %v", err)
				}
				defer setup2.Shutdown()

				ctx := context.Background()

				counter1 := setup1.Metrics.GetOrCreateCounter("adapter_counter", mt.MetricTypeTech, mt.MetricClassService)
				counter2 := setup2.Metrics.GetOrCreateCounter("util_counter", mt.MetricTypeTech, mt.MetricClassService)

				gauge1 := setup1.Metrics.GetOrCreateGauge("adapter_gauge", mt.MetricTypeTech, mt.MetricClassService)
				gauge2 := setup2.Metrics.GetOrCreateGauge("util_gauge", mt.MetricTypeTech, mt.MetricClassService)

				// Act
				counter1.Inc(ctx, mt.Attr("source", "adapter"))
				counter2.Inc(ctx, mt.Attr("source", "util"))

				gauge1.Set(ctx, 100, mt.Attr("type", "adapter"))
				gauge2.Set(ctx, 200, mt.Attr("type", "util"))

				// Assert
				// No assertions needed - if no panic occurred, the test passes

			})

		})

	})

}