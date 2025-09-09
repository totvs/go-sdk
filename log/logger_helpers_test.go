package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/totvs/go-sdk/log/middleware"
)

func TestWithFieldAddsSingleField(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFacade(buf, DebugLevel)

	lg := f.WithField("service", "orders")
	lg.Info("started")

	out := buf.String()
	if out == "" {
		t.Fatal("no log output")
	}

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if m["service"] != "orders" {
		t.Fatalf("expected service=orders, got: %v", m["service"])
	}
	if m["message"] != "started" {
		t.Fatalf("expected message=started, got: %v", m["message"])
	}
}

func TestHTTPMiddlewareGeneratesTraceIfMissing(t *testing.T) {
	buf := &bytes.Buffer{}
	baseF := NewFacade(buf, DebugLevel)

	handler := middleware.HTTPMiddlewareWithLogger(baseF)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// response header should contain X-Request-Id set by middleware
	rid := rr.Header().Get("X-Request-Id")
	if rid == "" {
		t.Fatalf("expected X-Request-Id header to be set, got empty")
	}

	// log output should contain the same id
	s := buf.String()
	if !strings.Contains(s, rid) {
		t.Fatalf("expected log to contain trace id %s, got: %s", rid, s)
	}
}

func TestHTTPMiddlewarePreservesProvidedTrace(t *testing.T) {
	buf := &bytes.Buffer{}
	baseF := NewFacade(buf, DebugLevel)

	handler := middleware.HTTPMiddlewareWithLogger(baseF)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "trace-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	rid := rr.Header().Get("X-Request-Id")
	if rid != "trace-123" {
		t.Fatalf("expected X-Request-Id to be preserved as trace-123, got: %s", rid)
	}

	// log output should contain the same id
	s := buf.String()
	if !strings.Contains(s, "trace-123") {
		t.Fatalf("expected log to contain provided trace id, got: %s", s)
	}
}

func TestHTTPMiddlewareInjectsLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	baseF := NewFacade(buf, DebugLevel)

	handler := middleware.HTTPMiddlewareWithLogger(baseF)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// handler should avoid duplicating the middleware log when middleware already logged
		if !LoggedFromContext(r.Context()) {
			if lf, ok := LoggerFromContext(r.Context()); ok {
				lf.Info("handler-log")
			}
		}
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	s := buf.String()
	// middleware should have logged the request
	if !strings.Contains(s, "http request received") {
		t.Fatalf("expected middleware to log request, got: %s", s)
	}
	// handler should not log the same message
	if strings.Contains(s, "handler-log") {
		t.Fatalf("expected handler not to duplicate middleware log, got: %s", s)
	}
}
