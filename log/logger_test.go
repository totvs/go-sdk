package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	logger "github.com/totvs/go-sdk/log"
	adapter "github.com/totvs/go-sdk/log/adapter"
	mware "github.com/totvs/go-sdk/log/middleware"
)

func TestWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := adapter.NewLog(buf, logger.DebugLevel)

	fields := map[string]interface{}{
		"str": "s",
		"i":   42,
		"i64": int64(99),
		"b":   true,
		"f":   3.14,
		"obj": map[string]interface{}{"a": "b"},
	}

	lg := f.WithFields(fields)
	lg.Info().Msg("testfields")

	out := buf.String()
	if out == "" {
		t.Fatal("no log output")
	}

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, out)
	}

	if m["str"] != "s" {
		t.Fatalf("unexpected str: %v", m["str"])
	}
	// JSON numbers are decoded as float64
	if m["i"].(float64) != 42 {
		t.Fatalf("unexpected i: %v", m["i"])
	}
	if m["b"] != true {
		t.Fatalf("unexpected b: %v", m["b"])
	}
	if _, ok := m["obj"].(map[string]interface{}); !ok {
		t.Fatalf("expected obj to be object, got: %T", m["obj"])
	}
}

func TestContextLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	f := adapter.NewLog(buf, logger.DebugLevel)

	ctx := logger.ContextWithLogger(context.Background(), f)
	if _, ok := logger.LoggerFromContext(ctx); !ok {
		t.Fatal("expected logger in context")
	}

	lg := logger.FromContext(ctx)
	lg.Info().Msg("ctxmsg")

	if !strings.Contains(buf.String(), "ctxmsg") {
		t.Fatalf("expected ctxmsg in output, got: %s", buf.String())
	}

	// nil context should not panic and should return false
	if _, ok := logger.LoggerFromContext(nil); ok {
		t.Fatal("expected no logger from nil context")
	}
}

func TestHTTPMiddlewareLogsTrace(t *testing.T) {
	// ensure default logger prints at info/debug level regardless of environment
	prevLvl := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", prevLvl)
	os.Setenv("LOG_LEVEL", "DEBUG")

	// capture stdout because HTTPMiddleware uses NewDefaultLog() which writes to stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w

	// perform a request with trace header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-Id", "trace-1")
	resp := httptest.NewRecorder()
	// use the handler directly to ensure middleware executes in-process
	handler := mware.HTTPMiddleware(http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		wr.Write([]byte("ok"))
	}))
	handler.ServeHTTP(resp, req)

	// restore stdout and read buffer
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	s := string(out)

	if !strings.Contains(s, "http request received") {
		t.Fatalf("expected http log, got: %s", s)
	}
	if !strings.Contains(s, "trace-1") {
		t.Fatalf("expected trace id in log, got: %s", s)
	}
}
