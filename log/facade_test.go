package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestFacadeBasicWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)
	f.Info().Msg("hello-facade")

	out := buf.String()
	if !strings.Contains(out, "hello-facade") {
		t.Fatalf("expected message in output, got: %s", out)
	}
}

func TestFacadeWithFieldAndFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)

	f2 := f.WithField("service", "orders")
	f3 := f2.WithFields(map[string]interface{}{"version": 3})
	f3.Info().Msg("started")

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, buf.String())
	}
	if m["service"] != "orders" {
		t.Fatalf("expected service=orders, got: %v", m["service"])
	}
	if m["version"].(float64) != 3 {
		t.Fatalf("expected version=3, got: %v", m["version"])
	}
}

func TestGlobalFacadeAndFromContext(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)

	// set global and use package shortcut
	SetGlobal(f)
	Info().Msg("via-global")
	if !strings.Contains(buf.String(), "via-global") {
		t.Fatalf("expected global message, got: %s", buf.String())
	}

	// injecting a facade into context and extracting it
	buf.Reset()
	fctx := NewLog(buf, DebugLevel)
	ctx := ContextWithLogger(context.Background(), fctx)
	f2 := FromContext(ctx)
	f2.Info().Msg("from-ctx")
	if !strings.Contains(buf.String(), "from-ctx") {
		t.Fatalf("expected from-ctx in output, got: %s", buf.String())
	}
}

func TestErrorWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)
	err := errors.New("boom")
	f.Error(err).Msg("failed action")

	s := buf.String()
	if !strings.Contains(s, "boom") {
		t.Fatalf("expected error text in output, got: %s", s)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, s)
	}
	if m["error"] != "boom" {
		t.Fatalf("expected error field 'boom', got: %v", m["error"])
	}
}

func TestErrorwWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)
	err := errors.New("boom")
	f.WithFields(map[string]interface{}{"service": "orders"}).Error(err).Msg("failed action")

	s := buf.String()
	if !strings.Contains(s, "boom") {
		t.Fatalf("expected error text in output, got: %s", s)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v, raw: %s", err, s)
	}
	if m["error"] != "boom" {
		t.Fatalf("expected error field 'boom', got: %v", m["error"])
	}
	if m["service"] != "orders" {
		t.Fatalf("expected service=orders, got: %v", m["service"])
	}
}

func TestErrfWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewLog(buf, DebugLevel)
	err := errors.New("boom")
	f.Error(err).Msgf("failed %s", "start")

	s := buf.String()
	if !strings.Contains(s, "boom") {
		t.Fatalf("expected error text in output, got: %s", s)
	}
	if !strings.Contains(s, "failed start") {
		t.Fatalf("expected formatted message 'failed start', got: %s", s)
	}
}
