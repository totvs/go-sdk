package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFacadeBasicWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFacade(buf, DebugLevel)
	f.Info("hello-facade")

	out := buf.String()
	if !strings.Contains(out, "hello-facade") {
		t.Fatalf("expected message in output, got: %s", out)
	}
}

func TestFacadeWithFieldAndFields(t *testing.T) {
	buf := &bytes.Buffer{}
	f := NewFacade(buf, DebugLevel)

	f2 := f.WithField("service", "orders")
	f3 := f2.WithFields(map[string]interface{}{"version": 3})
	f3.Info("started")

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
	f := NewFacade(buf, DebugLevel)

	// set global and use package shortcut
	SetGlobal(f)
	Info("via-global")
	if !strings.Contains(buf.String(), "via-global") {
		t.Fatalf("expected global message, got: %s", buf.String())
	}

	// injecting a facade into context and extracting it
	buf.Reset()
	fctx := NewFacade(buf, DebugLevel)
	ctx := ContextWithLogger(context.Background(), fctx)
	f2 := FromContextFacade(ctx)
	f2.Info("from-ctx")
	if !strings.Contains(buf.String(), "from-ctx") {
		t.Fatalf("expected from-ctx in output, got: %s", buf.String())
	}
}
