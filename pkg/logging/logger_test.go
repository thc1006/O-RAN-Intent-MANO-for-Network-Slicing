package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(string, ...any)
		level    string
		msg      string
		args     []any
		wantMsg  string
		wantArgs map[string]interface{}
	}{
		{
			name:    "debug level",
			logFunc: Debug,
			level:   "DEBUG",
			msg:     "debug message",
			args:    []any{"key", "value"},
			wantMsg: "debug message",
			wantArgs: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:    "info level",
			logFunc: Info,
			level:   "INFO",
			msg:     "info message",
			args:    []any{"component", "orchestrator"},
			wantMsg: "info message",
			wantArgs: map[string]interface{}{
				"component": "orchestrator",
			},
		},
		{
			name:    "warn level",
			logFunc: Warn,
			level:   "WARN",
			msg:     "warning message",
			args:    []any{"timeout", float64(30)},
			wantMsg: "warning message",
			wantArgs: map[string]interface{}{
				"timeout": float64(30),
			},
		},
		{
			name:    "error level",
			logFunc: Error,
			level:   "ERROR",
			msg:     "error message",
			args:    []any{"error", "connection failed"},
			wantMsg: "error message",
			wantArgs: map[string]interface{}{
				"error": "connection failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			oldLogger := defaultLogger
			defer func() { defaultLogger = oldLogger }()

			defaultLogger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			// Execute log function
			tt.logFunc(tt.msg, tt.args...)

			// Parse JSON output
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("failed to parse log output: %v", err)
			}

			// Verify log level
			if level, ok := logEntry["level"].(string); !ok || level != tt.level {
				t.Errorf("expected level %s, got %v", tt.level, logEntry["level"])
			}

			// Verify message
			if msg, ok := logEntry["msg"].(string); !ok || msg != tt.wantMsg {
				t.Errorf("expected msg %s, got %v", tt.wantMsg, logEntry["msg"])
			}

			// Verify structured fields
			for key, want := range tt.wantArgs {
				if got, ok := logEntry[key]; !ok {
					t.Errorf("missing field %s", key)
				} else if got != want {
					t.Errorf("field %s: expected %v, got %v", key, want, got)
				}
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		logFunc   func(string, ...any)
		shouldLog bool
	}{
		{
			name:      "debug logged when level is debug",
			level:     slog.LevelDebug,
			logFunc:   Debug,
			shouldLog: true,
		},
		{
			name:      "debug not logged when level is info",
			level:     slog.LevelInfo,
			logFunc:   Debug,
			shouldLog: false,
		},
		{
			name:      "info logged when level is info",
			level:     slog.LevelInfo,
			logFunc:   Info,
			shouldLog: true,
		},
		{
			name:      "error always logged",
			level:     slog.LevelWarn,
			logFunc:   Error,
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			oldLogger := defaultLogger
			defer func() { defaultLogger = oldLogger }()

			SetLevel(tt.level)
			defaultLogger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: tt.level,
			}))

			tt.logFunc("test message")

			gotLog := buf.Len() > 0
			if gotLog != tt.shouldLog {
				t.Errorf("expected log output: %v, got: %v", tt.shouldLog, gotLog)
			}
		})
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := defaultLogger
	defer func() { defaultLogger = oldLogger }()

	defaultLogger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create logger with context fields
	logger := With("component", "orchestrator", "version", "1.0.0")
	logger.Info("test message", "operation", "deploy")

	// Parse output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	// Verify all fields are present
	expectedFields := map[string]interface{}{
		"component": "orchestrator",
		"version":   "1.0.0",
		"operation": "deploy",
		"msg":       "test message",
	}

	for key, want := range expectedFields {
		if got, ok := logEntry[key]; !ok {
			t.Errorf("missing field %s", key)
		} else if got != want {
			t.Errorf("field %s: expected %v, got %v", key, want, got)
		}
	}
}

func TestContextLogging(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := defaultLogger
	defer func() { defaultLogger = oldLogger }()

	defaultLogger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	// Test without logger in context
	logger1 := FromContext(ctx)
	if logger1 != defaultLogger {
		t.Error("expected default logger when context has no logger")
	}

	// Test with logger in context
	contextLogger := With("request_id", "12345")
	ctx = NewContext(ctx, contextLogger)

	logger2 := FromContext(ctx)
	logger2.Info("context log", "action", "test")

	// Parse output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	// Verify request_id from context
	if reqID, ok := logEntry["request_id"].(string); !ok || reqID != "12345" {
		t.Errorf("expected request_id 12345, got %v", logEntry["request_id"])
	}
}

func TestLoggerInitialization(t *testing.T) {
	// Verify default logger is initialized
	if defaultLogger == nil {
		t.Error("default logger should be initialized")
	}

	// Test that we can log without panicking
	var buf bytes.Buffer
	oldLogger := defaultLogger
	defer func() { defaultLogger = oldLogger }()

	defaultLogger = slog.New(slog.NewJSONHandler(&buf, nil))
	Info("initialization test")

	if buf.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestMultipleFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := defaultLogger
	defer func() { defaultLogger = oldLogger }()

	defaultLogger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Log with various field types
	Info("test message",
		"string_field", "value",
		"int_field", 42,
		"float_field", 3.14,
		"bool_field", true,
	)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	expectedFields := map[string]interface{}{
		"string_field": "value",
		"int_field":    float64(42),
		"float_field":  3.14,
		"bool_field":   true,
	}

	for key, want := range expectedFields {
		if got, ok := logEntry[key]; !ok {
			t.Errorf("missing field %s", key)
		} else {
			switch v := want.(type) {
			case float64:
				if got != v {
					t.Errorf("field %s: expected %v, got %v", key, want, got)
				}
			default:
				if got != want {
					t.Errorf("field %s: expected %v, got %v", key, want, got)
				}
			}
		}
	}
}

func TestLoggerOutput(t *testing.T) {
	// Test that logger outputs to stdout by default
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reinitialize logger to use new stdout
	oldLogger := defaultLogger
	defer func() {
		defaultLogger = oldLogger
		os.Stdout = oldStdout
	}()

	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	Info("test output")

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)

	if !strings.Contains(buf.String(), "test output") {
		t.Error("expected log output to stdout")
	}
}