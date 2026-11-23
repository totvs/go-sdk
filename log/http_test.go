package log_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logger "github.com/totvs/go-sdk/log"
	adapter "github.com/totvs/go-sdk/log/adapter"
)

func TestWithHTTPRequestFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := adapter.NewLog(buf, logger.DebugLevel)

	r := httptest.NewRequest("GET", "/api/platform/api/v1/area?x=1", nil)
	r.RemoteAddr = "127.0.0.1:54321"
	r.Header.Set("User-Agent", "Mozilla/5.0 test")

	resp := &http.Response{StatusCode: http.StatusOK, ContentLength: 7418}
	f.WithHTTPRequest(r, resp).WithHTTPLatency(13123 * time.Millisecond).Info().Msg("http request completed")

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, buf.String())
	}

	want := map[string]interface{}{
		logger.HTTPClientIPField:           "127.0.0.1",
		logger.HTTPClientUserAgentField:    "Mozilla/5.0 test",
		logger.HTTPRequestMethodField:      "GET",
		logger.HTTPRequestPathField:        "/api/platform/api/v1/area",
		logger.HTTPResponseStatusCodeField: float64(200),
		logger.HTTPResponseBodySizeField:   float64(7418),
		logger.HTTPResponseLatencyField:    float64(13123),
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("field %q: want %v, got %v", k, v, m[k])
		}
	}
}

func TestWithHTTPRequestWithoutLatency(t *testing.T) {
	buf := &bytes.Buffer{}
	f := adapter.NewLog(buf, logger.DebugLevel)

	r := httptest.NewRequest("GET", "/x", nil)
	f.WithHTTPRequest(r, &http.Response{StatusCode: http.StatusNoContent}).Info().Msg("done")

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, buf.String())
	}
	if _, ok := m[logger.HTTPResponseLatencyField]; ok {
		t.Errorf("field %q should be absent without WithHTTPLatency", logger.HTTPResponseLatencyField)
	}
	if m[logger.HTTPResponseStatusCodeField] != float64(http.StatusNoContent) {
		t.Errorf("status: want %d, got %v", http.StatusNoContent, m[logger.HTTPResponseStatusCodeField])
	}
}

func TestNopLoggerWithHTTPRequest(t *testing.T) {
	prev := logger.GetGlobal()
	defer logger.SetGlobal(prev)
	logger.SetGlobal(nil)

	// GetGlobal falls back to the no-op logger; must not panic.
	logger.GetGlobal().WithHTTPRequest(nil, nil).WithHTTPLatency(0).Info().Msg("ignored")
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		remote  string
		want    string
	}{
		{"remote addr", nil, "10.0.0.1:1234", "10.0.0.1"},
		{"remote addr without port", nil, "10.0.0.1", "10.0.0.1"},
		{"x-forwarded-for first", map[string]string{"X-Forwarded-For": "203.0.113.5, 10.0.0.2"}, "10.0.0.1:1", "203.0.113.5"},
		{"x-real-ip", map[string]string{"X-Real-Ip": "198.51.100.7"}, "10.0.0.1:1", "198.51.100.7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tc.remote
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			if got := logger.ClientIP(r); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
	if got := logger.ClientIP(nil); got != "" {
		t.Fatalf("nil request: want empty, got %q", got)
	}
}
