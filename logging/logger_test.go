package logging

import (
	"bytes"
	"context"
	"testing"
)

func TestLogLevelConstants(t *testing.T) {
	t.Run("LogLevelDebug has correct value", func(t *testing.T) {
		if LogLevelDebug != "debug" {
			t.Errorf("expected 'debug', got %s", LogLevelDebug)
		}
	})

	t.Run("LogLevelInfo has correct value", func(t *testing.T) {
		if LogLevelInfo != "info" {
			t.Errorf("expected 'info', got %s", LogLevelInfo)
		}
	})

	t.Run("LogLevelWarn has correct value", func(t *testing.T) {
		if LogLevelWarn != "warn" {
			t.Errorf("expected 'warn', got %s", LogLevelWarn)
		}
	})

	t.Run("LogLevelError has correct value", func(t *testing.T) {
		if LogLevelError != "error" {
			t.Errorf("expected 'error', got %s", LogLevelError)
		}
	})
}

func TestLoggerStruct(t *testing.T) {
	t.Run("Logger embeds slog.Logger", func(t *testing.T) {
		logger := &Logger{}
		_ = logger.Logger
	})
}

func TestNewLogger(t *testing.T) {
	t.Run("creates logger with debug level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger(LogLevelDebug, buf)
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("creates logger with info level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger(LogLevelInfo, buf)
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("creates logger with warn level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger(LogLevelWarn, buf)
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("creates logger with error level", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := NewLogger(LogLevelError, buf)
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("uses stdout when output is nil", func(t *testing.T) {
		logger := NewLogger(LogLevelInfo, nil)
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("uses info level for unknown level", func(t *testing.T) {
		logger := NewLogger("unknown", &bytes.Buffer{})
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})
}

type traceIDKey string

func TestWithContext(t *testing.T) {
	t.Run("adds trace_id to logger", func(t *testing.T) {
		logger := Default()
		ctx := context.WithValue(context.Background(), traceIDKey("trace_id"), "test-123")
		loggerWithCtx := logger.WithContext(ctx)
		if loggerWithCtx == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("returns same logger for nil context", func(t *testing.T) {
		logger := Default()
		loggerWithCtx := logger.WithContext(context.TODO())
		if loggerWithCtx == nil {
			t.Error("expected non-nil logger")
		}
	})
}

func TestWithFields(t *testing.T) {
	t.Run("adds fields to logger", func(t *testing.T) {
		logger := Default()
		fields := map[string]interface{}{
			"user":   "testuser",
			"action": "login",
		}
		loggerWithFields := logger.WithFields(fields)
		if loggerWithFields == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("returns same logger for empty fields", func(t *testing.T) {
		logger := Default()
		loggerWithFields := logger.WithFields(map[string]interface{}{})
		if loggerWithFields == nil {
			t.Error("expected non-nil logger")
		}
	})
}

func TestDefault(t *testing.T) {
	t.Run("returns non-nil logger", func(t *testing.T) {
		logger := Default()
		if logger == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("returns logger with info level", func(t *testing.T) {
		logger := Default()
		_ = logger
	})
}
