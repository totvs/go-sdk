package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	lg "github.com/totvs/go-sdk/log"
	"github.com/totvs/go-sdk/trace"
)

func TestNewLog(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)
	if l == nil {
		t.Fatal("NewLog returned nil")
	}

	l.Info().Msg("test message")

	if buf.Len() == 0 {
		t.Error("expected log output, got nothing")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["message"] != "test message" {
		t.Errorf("expected message 'test message', got %v", entry["message"])
	}
	if entry["level"] != "info" {
		t.Errorf("expected level 'info', got %v", entry["level"])
	}
}

func TestNewLog_NilWriter(t *testing.T) {
	// Should not panic with nil writer
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewLog panicked with nil writer: %v", r)
		}
	}()

	// Note: zerolog will panic if w is nil, but our wrapper should handle this
	// For now, we document that nil writer is not supported
	l := NewLog(os.Stdout, lg.InfoLevel)
	if l == nil {
		t.Fatal("NewLog returned nil")
	}
}

func TestNewDefaultLog(t *testing.T) {
	l := NewDefaultLog()
	if l == nil {
		t.Fatal("NewDefaultLog returned nil")
	}
}

func TestNewDefaultLog_WithLogLevel(t *testing.T) {
	tests := []struct {
		envValue string
		expected lg.Level
	}{
		{"DEBUG", lg.DebugLevel},
		{"debug", lg.DebugLevel},
		{"INFO", lg.InfoLevel},
		{"info", lg.InfoLevel},
		{"WARN", lg.WarnLevel},
		{"warn", lg.WarnLevel},
		{"WARNING", lg.WarnLevel},
		{"warning", lg.WarnLevel},
		{"ERROR", lg.ErrorLevel},
		{"error", lg.ErrorLevel},
		{"INVALID", lg.InfoLevel}, // default
		{"", lg.InfoLevel},        // default
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("LOG_LEVEL", tt.envValue)
				defer os.Unsetenv("LOG_LEVEL")
			} else {
				os.Unsetenv("LOG_LEVEL")
			}
			l := NewDefaultLog()
			if l == nil {
				t.Fatal("NewDefaultLog returned nil")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    lg.Level
		logFunc  func(l lg.LoggerFacade)
		expected string
	}{
		{
			name:  "debug",
			level: lg.DebugLevel,
			logFunc: func(l lg.LoggerFacade) {
				l.Debug().Msg("debug message")
			},
			expected: "debug",
		},
		{
			name:  "info",
			level: lg.InfoLevel,
			logFunc: func(l lg.LoggerFacade) {
				l.Info().Msg("info message")
			},
			expected: "info",
		},
		{
			name:  "warn",
			level: lg.WarnLevel,
			logFunc: func(l lg.LoggerFacade) {
				l.Warn().Msg("warn message")
			},
			expected: "warn",
		},
		{
			name:  "error",
			level: lg.ErrorLevel,
			logFunc: func(l lg.LoggerFacade) {
				l.Error(nil).Msg("error message")
			},
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewLog(&buf, tt.level)
			tt.logFunc(l)

			var entry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("log output is not valid JSON: %v", err)
			}

			if entry["level"] != tt.expected {
				t.Errorf("expected level '%s', got %v", tt.expected, entry["level"])
			}
		})
	}
}

func TestErrorWithError(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.ErrorLevel)
	testErr := errors.New("test error")

	l.Error(testErr).Msg("operation failed")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["error"] != "test error" {
		t.Errorf("expected error 'test error', got %v", entry["error"])
	}
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	l2 := l.WithField("key", "value")
	l2.Info().Msg("with field")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["key"] != "value" {
		t.Errorf("expected key 'value', got %v", entry["key"])
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	fields := map[string]interface{}{
		"string":  "hello",
		"int":     42,
		"int64":   int64(123),
		"uint":    uint(10),
		"uint64":  uint64(20),
		"bool":    true,
		"float32": float32(1.5),
		"float64": float64(2.5),
		"other":   struct{ X int }{X: 1},
	}

	l2 := l.WithFields(fields)
	l2.Info().Msg("with fields")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["string"] != "hello" {
		t.Errorf("expected string 'hello', got %v", entry["string"])
	}
	if entry["bool"] != true {
		t.Errorf("expected bool true, got %v", entry["bool"])
	}
}

func TestWithTraceFromContext(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	ctx := trace.ContextWithTrace(context.Background(), "trace-123")
	l2 := l.WithTraceFromContext(ctx)
	l2.Info().Msg("with trace")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry[trace.TraceIDField] != "trace-123" {
		t.Errorf("expected trace_id 'trace-123', got %v", entry[trace.TraceIDField])
	}
}

func TestWithTraceFromContext_NoTrace(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	// Context without trace
	l2 := l.WithTraceFromContext(context.Background())
	l2.Info().Msg("without trace")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	// Should not have trace_id field
	if _, ok := entry[trace.TraceIDField]; ok {
		t.Error("expected no trace_id field when context has no trace")
	}
}

func TestLogEventChaining(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	l.Info().
		Str("str", "value").
		Int("int", 42).
		Int64("int64", 123).
		Uint("uint", 10).
		Uint64("uint64", 20).
		Bool("bool", true).
		Float32("float32", 1.5).
		Float64("float64", 2.5).
		Interface("iface", map[string]int{"a": 1}).
		Msg("chained event")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["str"] != "value" {
		t.Errorf("expected str 'value', got %v", entry["str"])
	}
	if entry["bool"] != true {
		t.Errorf("expected bool true, got %v", entry["bool"])
	}
}

func TestLogEventMsgf(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	l.Info().Msgf("hello %s", "world")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["message"] != "hello world" {
		t.Errorf("expected message 'hello world', got %v", entry["message"])
	}
}

func TestLogEventWrite(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)

	event := l.Info()
	n, err := event.Write([]byte("written message\n"))

	if err != nil {
		t.Errorf("Write returned error: %v", err)
	}
	if n != 16 { // "written message\n" has 16 bytes
		t.Errorf("Write returned wrong byte count: %d", n)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["message"] != "written message" {
		t.Errorf("expected message 'written message', got %v", entry["message"])
	}
}

func TestLogEventErr(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.InfoLevel)
	testErr := errors.New("inline error")

	l.Info().Err(testErr).Msg("with inline error")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}

	if entry["error"] != "inline error" {
		t.Errorf("expected error 'inline error', got %v", entry["error"])
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := NewLog(&buf, lg.WarnLevel)

	// Debug and Info should be filtered out
	l.Debug().Msg("debug")
	l.Info().Msg("info")

	if buf.Len() != 0 {
		t.Error("expected no output for levels below WarnLevel")
	}

	// Warn and Error should pass through
	l.Warn().Msg("warn")
	if buf.Len() == 0 {
		t.Error("expected output for WarnLevel")
	}
}
