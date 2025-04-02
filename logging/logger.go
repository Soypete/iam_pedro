package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// LogLevel represents logging levels
type LogLevel string

const (
	// LogLevelDebug enables all logs
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo enables info, warn, and error logs
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn enables warn and error logs
	LogLevelWarn LogLevel = "warn"
	// LogLevelError enables only error logs
	LogLevelError LogLevel = "error"
)

// Logger is the application's custom logger
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger with the specified level and output
// TODO: replace with a log file or a mode for debugging
func NewLogger(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	var logLevel slog.Level
	switch level {
	case LogLevelDebug:
		logLevel = slog.LevelDebug
	case LogLevelInfo:
		logLevel = slog.LevelInfo
	case LogLevelWarn:
		logLevel = slog.LevelWarn
	case LogLevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: logLevel,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext adds context values to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	return &Logger{
		Logger: l.Logger.With("trace_id", ctx.Value("trace_id")),
	}
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	if len(fields) == 0 {
		return l
	}

	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}

	return &Logger{
		Logger: l.Logger.With(attrs...),
	}
}

// Default returns a default logger with info level directed to stdout
func Default() *Logger {
	return NewLogger(LogLevelInfo, os.Stdout)
}
