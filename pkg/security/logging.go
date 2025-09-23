// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
)

// SecureLogger provides log injection protection
type SecureLogger struct {
	logger       *log.Logger
	maxLogLength int
}

// NewSecureLogger creates a new secure logger wrapper
func NewSecureLogger(logger *log.Logger) *SecureLogger {
	return &SecureLogger{
		logger:       logger,
		maxLogLength: 1024, // Prevent log flooding
	}
}

// SanitizeForLog removes dangerous characters that could be used for log injection
func SanitizeForLog(input string) string {
	if input == "" {
		return ""
	}

	// Remove or escape control characters that could manipulate logs
	var result strings.Builder
	result.Grow(len(input))

	for _, r := range input {
		switch {
		case r == '\n':
			result.WriteString("\\n")
		case r == '\r':
			result.WriteString("\\r")
		case r == '\t':
			result.WriteString("\\t")
		case r == '\v':
			result.WriteString("\\v")
		case r == '\f':
			result.WriteString("\\f")
		case r == '\b':
			result.WriteString("\\b")
		case r == '\a':
			result.WriteString("\\a")
		case r == 0x1b: // ESC character for ANSI codes
			result.WriteString("\\e")
		case unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r':
			// Replace other control characters with placeholder
			result.WriteString(fmt.Sprintf("\\x%02x", r))
		case r == '%':
			// Escape % to prevent format string attacks
			result.WriteString("%%")
		default:
			result.WriteRune(r)
		}
	}

	sanitized := result.String()

	// Truncate if too long to prevent log flooding
	if len(sanitized) > 512 {
		sanitized = sanitized[:509] + "..."
	}

	return sanitized
}

// SanitizeErrorForLog safely formats error messages for logging
func SanitizeErrorForLog(err error) string {
	if err == nil {
		return "<nil>"
	}
	return SanitizeForLog(err.Error())
}

// SanitizeIPForLog validates and sanitizes IP addresses for logging
func SanitizeIPForLog(ip string) string {
	if ip == "" {
		return "<empty>"
	}

	// First validate the IP address
	if err := ValidateIPAddress(ip); err != nil {
		return fmt.Sprintf("<invalid-ip:%s>", SanitizeForLog(ip))
	}

	// IP is valid, but still sanitize for safety
	return SanitizeForLog(ip)
}

// SanitizeStringForLog sanitizes arbitrary strings with additional validation
func SanitizeStringForLog(input string) string {
	if input == "" {
		return "<empty>"
	}

	// Check for suspicious patterns that might indicate injection attempts
	suspiciousPatterns := []string{
		"\x1b[",    // ANSI escape sequences
		"\033[",    // ANSI escape sequences (octal)
		"\\x1b[",   // Escaped ANSI sequences
		"\\033[",   // Escaped ANSI sequences (octal)
		"${",       // Shell variable expansion
		"$((",      // Arithmetic expansion
		"`",        // Command substitution
		"exec(",    // Code execution
		"eval(",    // Code evaluation
		"system(",  // System calls
	}

	sanitized := SanitizeForLog(input)

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(strings.ToLower(sanitized), strings.ToLower(pattern)) {
			return fmt.Sprintf("<suspicious-input:%s>", sanitized[:min(32, len(sanitized))])
		}
	}

	return sanitized
}

// SafeLogf provides format string validation and safe logging
func (sl *SecureLogger) SafeLogf(format string, args ...interface{}) {
	// Validate format string to prevent format string attacks
	if !isValidLogFormat(format) {
		sl.logger.Printf("[SECURITY] Invalid log format detected: %s", SanitizeForLog(format))
		return
	}

	// Sanitize all arguments
	sanitizedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		sanitizedArgs[i] = sanitizeLogArgument(arg)
	}

	// Check total message length
	message := fmt.Sprintf(format, sanitizedArgs...)
	if len(message) > sl.maxLogLength {
		message = message[:sl.maxLogLength-3] + "..."
	}

	sl.logger.Print(message)
}

// SafeLogError safely logs error messages
func (sl *SecureLogger) SafeLogError(prefix string, err error) {
	if err == nil {
		sl.logger.Printf("%s: <nil>", SanitizeForLog(prefix))
		return
	}

	sanitizedPrefix := SanitizeForLog(prefix)
	sanitizedError := SanitizeErrorForLog(err)

	sl.logger.Printf("%s: %s", sanitizedPrefix, sanitizedError)
}

// SafeLogInfo safely logs informational messages
func (sl *SecureLogger) SafeLogInfo(message string) {
	sanitized := SanitizeStringForLog(message)
	sl.logger.Printf("[INFO] %s", sanitized)
}

// SafeLogWarning safely logs warning messages
func (sl *SecureLogger) SafeLogWarning(message string) {
	sanitized := SanitizeStringForLog(message)
	sl.logger.Printf("[WARNING] %s", sanitized)
}

// SafeLogIP safely logs IP addresses with validation
func (sl *SecureLogger) SafeLogIP(prefix, ip string) {
	sanitizedPrefix := SanitizeForLog(prefix)
	sanitizedIP := SanitizeIPForLog(ip)

	sl.logger.Printf("%s: %s", sanitizedPrefix, sanitizedIP)
}

// isValidLogFormat validates log format strings to prevent format string attacks
func isValidLogFormat(format string) bool {
	// Check for basic format string validity
	if strings.Count(format, "%") > 10 {
		return false // Too many format specifiers
	}

	// Check for dangerous format specifiers
	dangerousSpecs := []string{"%n", "%x", "%X", "%p"}
	for _, spec := range dangerousSpecs {
		if strings.Contains(format, spec) {
			return false
		}
	}

	// Use regex to validate format specifiers
	formatRegex := regexp.MustCompile(`%[-#+ 0]?(\*|\d+)?(\.\*|\.\d+)?[vTtbcdoOqxXUeEfFgGsp%]`)

	// Find all format specifiers
	matches := formatRegex.FindAllString(format, -1)

	// Count actual % characters
	percentCount := strings.Count(format, "%")

	// Each %% counts as 1 % but doesn't need an argument
	escapedPercentCount := strings.Count(format, "%%")

	// Subtract escaped percents from total count
	expectedArgs := percentCount - escapedPercentCount*2 + len(matches)

	// Basic sanity check
	return expectedArgs >= 0 && expectedArgs <= 20
}

// sanitizeLogArgument sanitizes individual log arguments based on their type
func sanitizeLogArgument(arg interface{}) interface{} {
	switch v := arg.(type) {
	case string:
		return SanitizeStringForLog(v)
	case error:
		return SanitizeErrorForLog(v)
	case fmt.Stringer:
		return SanitizeStringForLog(v.String())
	default:
		// For other types, convert to string and sanitize
		return SanitizeStringForLog(fmt.Sprintf("%v", v))
	}
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Global secure logging functions for backward compatibility
var defaultSecureLogger *SecureLogger

// InitializeDefaultSecureLogger initializes the default secure logger
func InitializeDefaultSecureLogger(logger *log.Logger) {
	defaultSecureLogger = NewSecureLogger(logger)
}

// SafeLogf provides safe logging with format validation (global function)
func SafeLogf(logger *log.Logger, format string, args ...interface{}) {
	if defaultSecureLogger == nil {
		InitializeDefaultSecureLogger(logger)
	}
	defaultSecureLogger.SafeLogf(format, args...)
}

// SafeLogError provides safe error logging (global function)
func SafeLogError(logger *log.Logger, prefix string, err error) {
	if defaultSecureLogger == nil {
		InitializeDefaultSecureLogger(logger)
	}
	defaultSecureLogger.SafeLogError(prefix, err)
}

// SafeLogWarning provides safe warning logging (global function)
func SafeLogWarning(logger *log.Logger, message string) {
	if defaultSecureLogger == nil {
		InitializeDefaultSecureLogger(logger)
	}
	defaultSecureLogger.SafeLogWarning(message)
}

// SafeLogIP provides safe IP logging (global function)
func SafeLogIP(logger *log.Logger, prefix, ip string) {
	if defaultSecureLogger == nil {
		InitializeDefaultSecureLogger(logger)
	}
	defaultSecureLogger.SafeLogIP(prefix, ip)
}