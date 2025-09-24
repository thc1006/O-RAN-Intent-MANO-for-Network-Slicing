// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// TestSecureLogger provides controlled logging for security tests
type TestSecureLogger struct {
	entries []LogEntry
	level   LogLevel
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

type LogEntry struct {
	Level     LogLevel
	Message   string
	Fields    map[string]interface{}
	Timestamp time.Time
	Sensitive bool
}

func NewTestSecureLogger() *TestSecureLogger {
	return &TestSecureLogger{
		entries: make([]LogEntry, 0),
		level:   DEBUG,
	}
}

func (l *TestSecureLogger) Log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
		Sensitive: l.containsSensitiveData(message, fields),
	}
	l.entries = append(l.entries, entry)
}

func (l *TestSecureLogger) containsSensitiveData(message string, fields map[string]interface{}) bool {
	sensitivePatterns := []string{
		"password", "token", "secret", "key", "auth",
		"credential", "cert", "private", "session",
	}

	// Check message
	msgLower := strings.ToLower(message)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(msgLower, pattern) {
			return true
		}
	}

	// Check fields
	for key, value := range fields {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(fmt.Sprintf("%v", value))

		for _, pattern := range sensitivePatterns {
			if strings.Contains(keyLower, pattern) || strings.Contains(valueLower, pattern) {
				return true
			}
		}
	}

	return false
}

func (l *TestSecureLogger) GetEntries() []LogEntry {
	return l.entries
}

func (l *TestSecureLogger) GetEntriesByLevel(level LogLevel) []LogEntry {
	var filtered []LogEntry
	for _, entry := range l.entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (l *TestSecureLogger) HasSensitiveData() bool {
	for _, entry := range l.entries {
		if entry.Sensitive {
			return true
		}
	}
	return false
}

func (l *TestSecureLogger) Clear() {
	l.entries = l.entries[:0]
}

// SecureErrorHandler provides secure error handling with proper logging and sanitization
type SecureErrorHandler struct {
	logger          *TestSecureLogger
	sanitizeErrors  bool
	logStackTraces  bool
	maxErrorLength  int
	errorCounter    map[string]int
	rateLimitWindow time.Duration
}

func NewSecureErrorHandler(logger *TestSecureLogger) *SecureErrorHandler {
	return &SecureErrorHandler{
		logger:          logger,
		sanitizeErrors:  true,
		logStackTraces:  false, // Disabled in production
		maxErrorLength:  500,
		errorCounter:    make(map[string]int),
		rateLimitWindow: 1 * time.Minute,
	}
}

func (h *SecureErrorHandler) HandleError(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Sanitize error message
	sanitizedMsg := h.sanitizeErrorMessage(err.Error())

	// Rate limit error logging
	if h.shouldRateLimit(sanitizedMsg) {
		h.logger.Log(WARN, "Error rate limited", map[string]interface{}{
			"error_hash": h.hashError(sanitizedMsg),
		})
		return errors.New("internal error occurred")
	}

	// Log error securely
	logFields := map[string]interface{}{
		"error":     sanitizedMsg,
		"timestamp": time.Now(),
	}

	// Add context but sanitize it
	for key, value := range context {
		logFields["context_"+key] = h.sanitizeValue(value)
	}

	h.logger.Log(ERROR, "Error occurred", logFields)

	// Return sanitized error for external consumption
	if h.sanitizeErrors {
		return errors.New(h.createSafeErrorMessage(sanitizedMsg))
	}

	return errors.New(sanitizedMsg)
}

func (h *SecureErrorHandler) sanitizeErrorMessage(msg string) string {
	// Remove potentially sensitive information
	sensitivePatterns := map[string]string{
		`password=\S+`:           "password=***",
		`token=\S+`:              "token=***",
		`key=\S+`:                "key=***",
		`secret=\S+`:             "secret=***",
		`/home/\w+`:              "/home/***",
		`/Users/\w+`:             "/Users/***",
		`[a-zA-Z0-9+/]{20,}={0,2}`: "***", // Base64-like strings
		`\b\d{4}-\d{2}-\d{2}\b`:  "****-**-**", // Dates
		`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`: "***.***.***.***", // IP addresses
	}

	sanitized := msg
	for pattern, replacement := range sensitivePatterns {
		sanitized = strings.ReplaceAll(sanitized, pattern, replacement)
	}

	// Truncate if too long
	if len(sanitized) > h.maxErrorLength {
		sanitized = sanitized[:h.maxErrorLength] + "..."
	}

	return sanitized
}

func (h *SecureErrorHandler) sanitizeValue(value interface{}) interface{} {
	str := fmt.Sprintf("%v", value)
	return h.sanitizeErrorMessage(str)
}

func (h *SecureErrorHandler) shouldRateLimit(errorMsg string) bool {
	hash := h.hashError(errorMsg)
	h.errorCounter[hash]++

	// Simple rate limiting: max 10 occurrences per window
	return h.errorCounter[hash] > 10
}

func (h *SecureErrorHandler) hashError(errorMsg string) string {
	// Simple hash for demonstration - in production use proper hashing
	return fmt.Sprintf("%x", len(errorMsg)+int(errorMsg[0]))
}

func (h *SecureErrorHandler) createSafeErrorMessage(originalMsg string) string {
	// Create user-friendly error messages that don't reveal internal details
	safeMessages := map[string]string{
		"sql":           "Database operation failed",
		"connection":    "Service temporarily unavailable",
		"timeout":       "Request timeout",
		"unauthorized":  "Access denied",
		"invalid":       "Invalid request",
		"permission":    "Insufficient permissions",
		"not found":     "Resource not found",
	}

	msgLower := strings.ToLower(originalMsg)
	for keyword, safeMsg := range safeMessages {
		if strings.Contains(msgLower, keyword) {
			return safeMsg
		}
	}

	return "An error occurred while processing your request"
}

// Recovery function for handling panics securely
func (h *SecureErrorHandler) RecoverPanic() {
	if r := recover(); r != nil {
		h.logger.Log(FATAL, "Panic recovered", map[string]interface{}{
			"panic":     h.sanitizeValue(r),
			"timestamp": time.Now(),
		})
	}
}

// Test functions start here

func TestSecureErrorHandling(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("basic_error_handling", func(t *testing.T) {
		err := errors.New("database connection failed")
		context := map[string]interface{}{
			"user_id": 12345,
			"action":  "login",
		}

		handledErr := handler.HandleError(err, context)

		assert.Error(t, handledErr)
		assert.NotEqual(t, err.Error(), handledErr.Error(), "Error should be sanitized")

		// Check logging
		entries := logger.GetEntriesByLevel(ERROR)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Message, "Error occurred")
	})

	t.Run("sensitive_data_sanitization", func(t *testing.T) {
		sensitiveErrors := []string{
			"authentication failed for user password=secret123",
			"SQL error: invalid query with token=abc123def456",
			"file not found: /home/user/secret.key",
			"connection error to database with key=sensitive_key",
		}

		for _, errMsg := range sensitiveErrors {
			logger.Clear()
			err := errors.New(errMsg)

			handledErr := handler.HandleError(err, nil)

			// Handled error should not contain sensitive data
			assert.NotContains(t, handledErr.Error(), "password=secret123")
			assert.NotContains(t, handledErr.Error(), "token=abc123def456")
			assert.NotContains(t, handledErr.Error(), "/home/user")
			assert.NotContains(t, handledErr.Error(), "key=sensitive_key")

			// Log should be sanitized too
			entries := logger.GetEntriesByLevel(ERROR)
			if len(entries) > 0 {
				logContent := fmt.Sprintf("%v", entries[0].Fields)
				assert.NotContains(t, logContent, "secret123")
				assert.NotContains(t, logContent, "abc123def456")
			}
		}
	})

	t.Run("error_rate_limiting", func(t *testing.T) {
		logger.Clear()

		// Generate many identical errors
		err := errors.New("repeated error")
		for i := 0; i < 15; i++ {
			handler.HandleError(err, nil)
		}

		// Should have rate limited some errors
		entries := logger.GetEntriesByLevel(WARN)
		rateLimitedCount := 0
		for _, entry := range entries {
			if strings.Contains(entry.Message, "rate limited") {
				rateLimitedCount++
			}
		}

		assert.True(t, rateLimitedCount > 0, "Should rate limit repeated errors")
	})

	t.Run("long_error_truncation", func(t *testing.T) {
		logger.Clear()

		// Create very long error message
		longMsg := strings.Repeat("This is a very long error message. ", 50)
		err := errors.New(longMsg)

		handledErr := handler.HandleError(err, nil)

		// Should be truncated
		assert.True(t, len(handledErr.Error()) <= handler.maxErrorLength+10,
			"Error message should be truncated")

		// Log entry should also be truncated
		entries := logger.GetEntriesByLevel(ERROR)
		if len(entries) > 0 {
			errorField := fmt.Sprintf("%v", entries[0].Fields["error"])
			assert.True(t, len(errorField) <= handler.maxErrorLength+10,
				"Logged error should be truncated")
		}
	})
}

func TestSecureLogging(t *testing.T) {
	logger := NewTestSecureLogger()

	t.Run("sensitive_data_detection", func(t *testing.T) {
		// Log sensitive data
		sensitiveData := map[string]interface{}{
			"password": "secret123",
			"user_id":  12345,
		}

		logger.Log(INFO, "User login attempt", sensitiveData)

		assert.True(t, logger.HasSensitiveData(), "Should detect sensitive data")

		entries := logger.GetEntries()
		assert.Len(t, entries, 1)
		assert.True(t, entries[0].Sensitive, "Entry should be marked as sensitive")
	})

	t.Run("non_sensitive_data_logging", func(t *testing.T) {
		logger.Clear()

		nonSensitiveData := map[string]interface{}{
			"user_id":   12345,
			"timestamp": time.Now(),
			"action":    "view_page",
		}

		logger.Log(INFO, "User action", nonSensitiveData)

		assert.False(t, logger.HasSensitiveData(), "Should not detect sensitive data")

		entries := logger.GetEntries()
		assert.Len(t, entries, 1)
		assert.False(t, entries[0].Sensitive, "Entry should not be marked as sensitive")
	})

	t.Run("log_level_filtering", func(t *testing.T) {
		logger.Clear()

		logger.Log(DEBUG, "Debug message", nil)
		logger.Log(INFO, "Info message", nil)
		logger.Log(WARN, "Warning message", nil)
		logger.Log(ERROR, "Error message", nil)

		debugEntries := logger.GetEntriesByLevel(DEBUG)
		errorEntries := logger.GetEntriesByLevel(ERROR)

		assert.Len(t, debugEntries, 1)
		assert.Len(t, errorEntries, 1)
		assert.Equal(t, "Debug message", debugEntries[0].Message)
		assert.Equal(t, "Error message", errorEntries[0].Message)
	})
}

func TestPanicRecovery(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("panic_recovery_and_logging", func(t *testing.T) {
		func() {
			defer handler.RecoverPanic()
			panic("test panic with sensitive data: password=secret")
		}()

		// Check that panic was logged
		entries := logger.GetEntriesByLevel(FATAL)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Message, "Panic recovered")

		// Check that sensitive data was sanitized
		panicData := fmt.Sprintf("%v", entries[0].Fields["panic"])
		assert.NotContains(t, panicData, "password=secret")
	})

	t.Run("panic_with_complex_data", func(t *testing.T) {
		logger.Clear()

		func() {
			defer handler.RecoverPanic()

			// Complex panic with map
			panicData := map[string]interface{}{
				"error":    "system failure",
				"password": "secret123",
				"token":    "abc123def456",
			}
			panic(panicData)
		}()

		entries := logger.GetEntriesByLevel(FATAL)
		assert.Len(t, entries, 1)

		// Ensure sensitive data is sanitized
		logContent := fmt.Sprintf("%v", entries[0])
		assert.NotContains(t, logContent, "secret123")
		assert.NotContains(t, logContent, "abc123def456")
	})
}

func TestErrorHandlingIntegration(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("integration_with_subprocess_executor", func(t *testing.T) {
		executor := security.NewSecureSubprocessExecutor()
		ctx := context.Background()

		// Try to execute a command that will fail
		_, err := executor.SecureExecute(ctx, "nonexistent_command", "arg1", "arg2")

		if err != nil {
			context := map[string]interface{}{
				"command": "nonexistent_command",
				"args":    []string{"arg1", "arg2"},
				"user":    "test_user",
			}

			handledErr := handler.HandleError(err, context)

			assert.Error(t, handledErr)
			assert.NotEqual(t, err.Error(), handledErr.Error())

			// Check that error was logged securely
			entries := logger.GetEntriesByLevel(ERROR)
			assert.True(t, len(entries) > 0)
		}
	})

	t.Run("integration_with_validation_errors", func(t *testing.T) {
		logger.Clear()
		validator := security.NewInputValidator()

		// Generate validation error
		err := validator.ValidateCommandArgument("dangerous; rm -rf /")

		if err != nil {
			context := map[string]interface{}{
				"input":      "dangerous; rm -rf /",
				"validation": "command_argument",
			}

			handledErr := handler.HandleError(err, context)

			assert.Error(t, handledErr)

			// Ensure the dangerous command is not in the sanitized error
			assert.NotContains(t, handledErr.Error(), "rm -rf /")

			// Check logging
			entries := logger.GetEntriesByLevel(ERROR)
			assert.True(t, len(entries) > 0)
		}
	})
}

func TestErrorSanitizationEdgeCases(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("unicode_in_error_messages", func(t *testing.T) {
		unicodeError := errors.New("Error with unicode: 用户名或密码错误 password=test123")

		handledErr := handler.HandleError(unicodeError, nil)

		// Should handle unicode properly and still sanitize
		assert.NotContains(t, handledErr.Error(), "password=test123")
		assert.Contains(t, handledErr.Error(), "用户名或密码错误") // Unicode should be preserved
	})

	t.Run("nested_error_messages", func(t *testing.T) {
		wrappedErr := fmt.Errorf("outer error: %w",
			fmt.Errorf("inner error with token=secret: %w",
				errors.New("root cause")))

		handledErr := handler.HandleError(wrappedErr, nil)

		// Should sanitize even nested errors
		assert.NotContains(t, handledErr.Error(), "token=secret")
	})

	t.Run("json_in_error_messages", func(t *testing.T) {
		jsonError := errors.New(`{"error": "failed", "password": "secret123", "user": "admin"}`)

		handledErr := handler.HandleError(jsonError, nil)

		// Should sanitize JSON content
		assert.NotContains(t, handledErr.Error(), "secret123")
	})

	t.Run("base64_encoded_data", func(t *testing.T) {
		// Base64 encoded "password:secret123"
		base64Error := errors.New("Authentication failed: cGFzc3dvcmQ6c2VjcmV0MTIz")

		handledErr := handler.HandleError(base64Error, nil)

		// Should detect and sanitize base64-like strings
		assert.NotContains(t, handledErr.Error(), "cGFzc3dvcmQ6c2VjcmV0MTIz")
	})
}

func TestConcurrentErrorHandling(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("concurrent_error_handling", func(t *testing.T) {
		const numGoroutines = 50
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				err := fmt.Errorf("concurrent error %d with password=secret%d", id, id)
				context := map[string]interface{}{
					"goroutine_id": id,
					"test":         "concurrent",
				}

				handledErr := handler.HandleError(err, context)
				results <- handledErr
			}(i)
		}

		// Collect all results
		var handledErrors []error
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-results:
				handledErrors = append(handledErrors, err)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent error handling")
			}
		}

		// All errors should be sanitized
		for i, err := range handledErrors {
			assert.NotContains(t, err.Error(), fmt.Sprintf("password=secret%d", i),
				"Error %d should be sanitized", i)
		}

		// Check that logging worked correctly
		entries := logger.GetEntriesByLevel(ERROR)
		assert.True(t, len(entries) > 0, "Should have logged errors")
	})
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	b.Run("simple_error_handling", func(b *testing.B) {
		err := errors.New("simple test error")
		context := map[string]interface{}{"test": "benchmark"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.HandleError(err, context)
		}
	})

	b.Run("sensitive_data_sanitization", func(b *testing.B) {
		err := errors.New("error with password=secret123 and token=abc123def456")
		context := map[string]interface{}{
			"password": "secret123",
			"token":    "abc123def456",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.HandleError(err, context)
		}
	})

	b.Run("long_error_truncation", func(b *testing.B) {
		longError := errors.New(strings.Repeat("This is a long error message. ", 100))
		context := map[string]interface{}{"test": "benchmark"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.HandleError(longError, context)
		}
	})
}

func TestErrorHandlingConfiguration(t *testing.T) {
	logger := NewTestSecureLogger()

	t.Run("custom_configuration", func(t *testing.T) {
		handler := NewSecureErrorHandler(logger)
		handler.sanitizeErrors = false
		handler.maxErrorLength = 100

		longError := errors.New(strings.Repeat("x", 200))

		handledErr := handler.HandleError(longError, nil)

		// Should respect custom configuration
		assert.True(t, len(handledErr.Error()) <= 110, // 100 + "..." = 103, with some margin
			"Should respect custom max error length")
	})

	t.Run("production_vs_development_mode", func(t *testing.T) {
		// Production mode - sanitize everything
		prodHandler := NewSecureErrorHandler(logger)
		prodHandler.sanitizeErrors = true
		prodHandler.logStackTraces = false

		// Development mode - more verbose
		devHandler := NewSecureErrorHandler(logger)
		devHandler.sanitizeErrors = false
		devHandler.logStackTraces = true

		testError := errors.New("detailed error: file not found at /etc/secret")

		prodErr := prodHandler.HandleError(testError, nil)
		devErr := devHandler.HandleError(testError, nil)

		// Production should be more generic
		assert.NotContains(t, prodErr.Error(), "/etc/secret")

		// Development might preserve more details (but should still be careful)
		// In this case, both should sanitize paths for security
		assert.NotContains(t, devErr.Error(), "/etc/secret")
	})
}

func TestErrorMetrics(t *testing.T) {
	logger := NewTestSecureLogger()
	handler := NewSecureErrorHandler(logger)

	t.Run("error_frequency_tracking", func(t *testing.T) {
		// Generate different types of errors
		errorMessages := []string{
			"database connection failed",
			"authentication failed",
			"validation error",
			"database connection failed", // Repeat
			"authentication failed",     // Repeat
		}

		for _, errMsg := range errorMessages {
			err := fmt.Errorf("%s", errMsg)
			handler.HandleError(err, nil)
		}

		// Check that error counting is working
		// This is internal to the handler, so we verify through rate limiting behavior
		for i := 0; i < 12; i++ {
			err := errors.New("database connection failed")
			handler.HandleError(err, nil)
		}

		// Should see rate limiting warnings
		entries := logger.GetEntriesByLevel(WARN)
		rateLimitedFound := false
		for _, entry := range entries {
			if strings.Contains(entry.Message, "rate limited") {
				rateLimitedFound = true
				break
			}
		}

		assert.True(t, rateLimitedFound, "Should rate limit repeated errors")
	})
}