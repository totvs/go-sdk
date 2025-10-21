package util

import (
	"net/http"
	"strconv"

	mt "github.com/totvs/go-sdk/metrics"
)

// HTTPMetricsMiddleware provides HTTP metrics collection middleware.
type HTTPMetricsMiddleware struct {
	counter     mt.Counter
	serviceName string
}

// NewHTTPMetricsMiddleware creates a new HTTP metrics middleware.
// HTTP metrics are technical (tech) service-level metrics.
func NewHTTPMetricsMiddleware(metrics mt.MetricsFacade, serviceName string) *HTTPMetricsMiddleware {
	counter := metrics.GetOrCreateCounter("http_requests_total", mt.MetricTypeTech, mt.MetricClassService)

	return &HTTPMetricsMiddleware{
		counter:     counter,
		serviceName: serviceName,
	}
}

// Handler wraps an http.Handler with automatic HTTP metrics collection.
func (m *HTTPMetricsMiddleware) Handler(handler http.Handler) http.Handler {
	return &metricsHandler{
		handler:     handler,
		counter:     m.counter,
		serviceName: m.serviceName,
	}
}

// metricsHandler wraps an http.Handler to automatically collect HTTP metrics
type metricsHandler struct {
	handler     http.Handler
	counter     mt.Counter
	serviceName string
}

func (mh *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	mh.handler.ServeHTTP(wrappedWriter, r)

	mh.counter.Add(r.Context(), 1,
		mt.Attr("method", r.Method),
		mt.Attr("path", r.URL.Path),
		mt.Attr("status", strconv.Itoa(wrappedWriter.statusCode)),
		mt.Attr("service", mh.serviceName),
	)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// WithMetrics is a convenience function that wraps an http.Handler with automatic
// HTTP metrics collection. It creates the middleware internally.
func WithMetrics(metrics mt.MetricsFacade, serviceName string, handler http.Handler) http.Handler {
	middleware := NewHTTPMetricsMiddleware(metrics, serviceName)
	return middleware.Handler(handler)
}
