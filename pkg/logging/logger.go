package logging

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// Initialize default logger with JSON handler
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// SetLevel configures the minimum log level
func SetLevel(level slog.Level) {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

// Debug logs a debug-level message with structured fields
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info-level message with structured fields
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning-level message with structured fields
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error-level message with structured fields
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// With creates a new logger with the given structured fields pre-populated
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

// FromContext retrieves a logger from the context, or returns the default logger
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value("logger").(*slog.Logger); ok {
		return logger
	}
	return defaultLogger
}

// NewContext creates a new context with the given logger attached
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

// GetDefaultLogger returns the default logger instance
func GetDefaultLogger() *slog.Logger {
	return defaultLogger
}

// SetDefaultLogger sets a custom default logger
func SetDefaultLogger(logger *slog.Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}