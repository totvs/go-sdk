package util_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/totvs/go-sdk/metrics/adapter"
	"github.com/totvs/go-sdk/metrics/util"
)

func TestNewHTTPMetricsMiddleware(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("middleware-test")
	if err != nil {
		t.Fatalf("failed to setup metrics: %v", err)
	}
	defer setup.Shutdown()

	middleware := util.NewHTTPMetricsMiddleware(setup.Metrics, "test-service")
	if middleware == nil {
		t.Fatal("expected non-nil middleware")
	}
}

func TestHTTPMiddlewareBasicFunctionality(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("middleware-basic-test")
	if err != nil {
		t.Fatalf("failed to setup metrics: %v", err)
	}
	defer setup.Shutdown()

	middleware := util.NewHTTPMetricsMiddleware(setup.Metrics, "test-service")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrappedHandler := middleware.Handler(testHandler)

	// Make a request
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got: %d", rec.Code)
	}
	if rec.Body.String() != "OK" {
		t.Fatalf("expected body 'OK', got: %s", rec.Body.String())
	}
}

func TestHTTPMiddlewareMetricsCollection(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("metrics-collection-test")
	if err != nil {
		t.Fatalf("failed to setup metrics: %v", err)
	}
	defer setup.Shutdown()

	middleware := util.NewHTTPMetricsMiddleware(setup.Metrics, "test-service")

	// Create test handlers with different status codes
	handlers := map[string]http.HandlerFunc{
		"/success": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		},
		"/error": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		},
		"/not-found": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		},
	}

	// Make requests to each handler
	for path, handler := range handlers {
		wrappedHandler := middleware.Handler(handler)

		// Test different HTTP methods
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, method := range methods {
			req := httptest.NewRequest(method, path, nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)
		}
	}

	// Verify metrics are collected by checking Prometheus output
	metricsHandler := promhttp.HandlerFor(setup.Registry, promhttp.HandlerOpts{})
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	metricsHandler.ServeHTTP(rec, req)

	output := rec.Body.String()

	// Verify counter is present
	if !strings.Contains(output, "http_requests_total") {
		t.Fatal("expected http_requests_total metric in output")
	}

	// Verify labels are present
	expectedLabels := []string{
		`method="GET"`,
		`method="POST"`,
		`path="/success"`,
		`path="/error"`,
		`status="200"`,
		`status="500"`,
		`status="404"`,
		`service="test-service"`,
	}

	for _, label := range expectedLabels {
		if !strings.Contains(output, label) {
			t.Fatalf("expected label %s in output, got: %s", label, output)
		}
	}
}

func TestHTTPMiddlewareStatusCodeCapture(t *testing.T) {
	setup, err := adapter.NewPrometheusMetrics("status-code-test")
	if err != nil {
		t.Fatalf("failed to setup metrics: %v", err)
	}
	defer setup.Shutdown()

	middleware := util.NewHTTPMetricsMiddleware(setup.Metrics, "status-test-service")

	testCases := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
	}{
		{
			name: "explicit 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			},
			expectedStatus: 200,
		},
		{
			name: "implicit 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ok"))
			},
			expectedStatus: 200,
		},
		{
			name: "404 error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("not found"))
			},
			expectedStatus: 404,
		},
		{
			name: "500 error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error"))
			},
			expectedStatus: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrappedHandler := middleware.Handler(tc.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got: %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}
