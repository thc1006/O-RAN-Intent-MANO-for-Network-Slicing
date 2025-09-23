// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"
	"time"
	"crypto/rand"
	"encoding/hex"
)

// SecureLogger provides log injection protection
type SecureLogger struct {
	logger       *log.Logger
	maxLogLength int
	logID        string
	strict       bool // Enable stricter validation
}

// NewSecureLogger creates a new secure logger wrapper
func NewSecureLogger(logger *log.Logger) *SecureLogger {
	return &SecureLogger{
		logger:       logger,
		maxLogLength: 1024, // Prevent log flooding
		logID:        generateLoggerID(),
		strict:       true, // Enable strict mode by default
	}
}

// NewLegacySecureLogger creates a logger with legacy compatibility (less strict)
func NewLegacySecureLogger(logger *log.Logger) *SecureLogger {
	return &SecureLogger{
		logger:       logger,
		maxLogLength: 1024,
		logID:        generateLoggerID(),
		strict:       false,
	}
}

// generateLoggerID creates a unique identifier for the logger instance
func generateLoggerID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SanitizeForLog removes dangerous characters that could be used for log injection
func SanitizeForLog(input string) string {
	if input == "" {
		return ""
	}

	// Pre-validate for common injection patterns
	if containsLogInjectionPatterns(input) {
		return fmt.Sprintf("<LOG_INJECTION_BLOCKED:len=%d>", len(input))
	}

	// Remove or escape control characters that could manipulate logs
	var result strings.Builder
	result.Grow(len(input) + 64) // Extra space for escape sequences

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
		case r == 0x00: // NULL byte
			result.WriteString("\\0")
		case r == 0x7F: // DEL character
			result.WriteString("\\x7F")
		case unicode.IsControl(r):
			// Replace all control characters with safe representation
			result.WriteString(fmt.Sprintf("\\x%02X", r))
		case r == '%':
			// Escape % to prevent format string attacks
			result.WriteString("%%")
		case r == '\\':
			// Escape backslashes to prevent escape sequence injection
			result.WriteString("\\\\")
		case r > unicode.MaxASCII:
			// Handle non-ASCII characters safely
			if unicode.IsPrint(r) {
				result.WriteRune(r)
			} else {
				result.WriteString(fmt.Sprintf("\\u%04X", r))
			}
		default:
			if unicode.IsPrint(r) {
				result.WriteRune(r)
			} else {
				result.WriteString(fmt.Sprintf("\\x%02X", r))
			}
		}
	}

	sanitized := result.String()

	// Truncate if too long to prevent log flooding
	if len(sanitized) > 512 {
		sanitized = sanitized[:509] + "..."
	}

	// Final validation check
	if containsLogInjectionPatterns(sanitized) {
		return fmt.Sprintf("<POST_SANITIZATION_BLOCKED:len=%d>", len(sanitized))
	}

	return sanitized
}

// containsLogInjectionPatterns detects common log injection attack patterns
func containsLogInjectionPatterns(input string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(input)

	// Log injection patterns
	dangerousPatterns := []string{
		// ANSI escape sequences
		"\x1b[",
		"\033[",
		"\u001b[",
		// Log level injection attempts
		"\n[error]",
		"\n[warn]",
		"\n[info]",
		"\n[debug]",
		"\nERROR",
		"\nWARN",
		"\nINFO",
		"\nDEBUG",
		"\nFATAL",
		// Timestamp injection
		"\n2", // Common timestamp prefix
		// Log forging attempts
		"\n20", // Year in timestamp
		"\r\n", // CRLF injection
		// Terminal control sequences
		"\x0c", // Form feed
		"\x08", // Backspace
		// URL encoding attempts
		"%0a", // Encoded newline
		"%0d", // Encoded carriage return
		"%1b", // Encoded escape
		// Unicode newlines
		"\u2028", // Line separator
		"\u2029", // Paragraph separator
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Check for repeated newlines (potential log flooding)
	if strings.Count(input, "\n") > 3 {
		return true
	}

	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return true
	}

	return false
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

	// Pre-check for extreme values
	if len(input) > 10240 { // 10KB limit
		return fmt.Sprintf("<oversized-input:len=%d>", len(input))
	}

	// Check for suspicious patterns that might indicate injection attempts
	suspiciousPatterns := []string{
		// ANSI escape sequences
		"\x1b[", "\033[", "\\x1b[", "\\033[",
		// Shell patterns
		"${", "$((", "`",
		// Code execution
		"exec(", "eval(", "system(", "Runtime.getRuntime(",
		// Script injection
		"<script", "javascript:", "vbscript:",
		// Log level spoofing
		"[ERROR]", "[WARN]", "[INFO]", "[DEBUG]", "[FATAL]",
		// Encoded attacks
		"%3c", "%3e", "%22", "%27", // <, >, ", '
		// Unicode attacks
		"\u0000", "\u001b", "\u2028", "\u2029",
		// Binary data indicators
		"\x00", "\xFF", "\xFE",
	}

	// First sanitization pass
	sanitized := SanitizeForLog(input)

	// Check for suspicious patterns in the sanitized output
	lower := strings.ToLower(sanitized)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return fmt.Sprintf("<suspicious-input:pattern=%s:len=%d>",
				SanitizeForLog(pattern), len(sanitized))
		}
	}

	// Check for excessive repetition (potential DoS)
	if detectExcessiveRepetition(sanitized) {
		return fmt.Sprintf("<repetitive-input:len=%d>", len(sanitized))
	}

	return sanitized
}

// detectExcessiveRepetition checks for patterns that might cause log flooding
func detectExcessiveRepetition(input string) bool {
	// Check for repeated characters
	for i := 0; i < len(input)-10; i++ {
		char := input[i]
		count := 1
		for j := i + 1; j < len(input) && j < i+20; j++ {
			if input[j] == char {
				count++
			} else {
				break
			}
		}
		if count > 10 {
			return true
		}
	}

	// Check for repeated short patterns
	for patternLen := 2; patternLen <= 5; patternLen++ {
		for i := 0; i < len(input)-patternLen*3; i++ {
			pattern := input[i : i+patternLen]
			count := 1
			j := i + patternLen
			for j < len(input)-patternLen && strings.HasPrefix(input[j:], pattern) {
				count++
				j += patternLen
			}
			if count > 5 {
				return true
			}
		}
	}

	return false
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

	// Format message with sanitized arguments
	message := fmt.Sprintf(format, sanitizedArgs...)

	// Validate the final message for injection attempts
	if err := sl.validateLogMessage(message); err != nil {
		sl.logger.Printf("[SECURITY] Log validation failed: %s", SanitizeForLog(err.Error()))
		return
	}

	// Check total message length
	if len(message) > sl.maxLogLength {
		message = message[:sl.maxLogLength-3] + "..."
	}

	// Final sanitization before logging (defense in depth)
	finalMessage := sl.finalSanitize(message)

	// Log with timestamp and logger ID for audit trail
	sl.logger.Printf("[%s][%s] %s", time.Now().Format("2006-01-02T15:04:05.000Z07:00"), sl.logID, finalMessage)
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

// validateLogMessage performs comprehensive validation of the final log message
func (sl *SecureLogger) validateLogMessage(message string) error {
	if message == "" {
		return nil // Empty messages are allowed
	}

	// Check message length
	if len(message) > sl.maxLogLength*2 {
		return fmt.Errorf("message too long: %d characters", len(message))
	}

	// Check for log injection patterns
	if containsLogInjectionPatterns(message) {
		return fmt.Errorf("message contains log injection patterns")
	}

	// Check for excessive newlines (log forging)
	if strings.Count(message, "\\n") > 5 || strings.Count(message, "\n") > 2 {
		return fmt.Errorf("excessive newlines detected")
	}

	// Validate character composition in strict mode
	if sl.strict {
		return sl.strictValidateMessage(message)
	}

	return nil
}

// strictValidateMessage performs strict validation of log messages
func (sl *SecureLogger) strictValidateMessage(message string) error {
	// Count non-printable characters
	nonPrintableCount := 0
	for _, r := range message {
		if !unicode.IsPrint(r) && r != '\t' && r != ' ' {
			nonPrintableCount++
		}
	}

	// Allow some escaped characters, but not too many
	if nonPrintableCount > 10 {
		return fmt.Errorf("too many non-printable characters: %d", nonPrintableCount)
	}

	// Check for binary data
	if strings.Contains(message, "\x00") {
		return fmt.Errorf("null bytes not allowed in log messages")
	}

	// Check for Unicode bidirectional override attacks
	bidirectionalOverrides := []string{
		"\u202D", "\u202E", // Left-to-right/Right-to-left override
		"\u2066", "\u2067", "\u2068", "\u2069", // Directional isolates
	}
	for _, override := range bidirectionalOverrides {
		if strings.Contains(message, override) {
			return fmt.Errorf("bidirectional text override detected")
		}
	}

	return nil
}

// finalSanitize performs final sanitization before logging (defense in depth)
func (sl *SecureLogger) finalSanitize(message string) string {
	// Remove any remaining dangerous patterns that might have been missed
	dangerous := map[string]string{
		"\x1b[":   "\\x1b[",  // ANSI escape
		"\x00":    "\\x00",   // Null byte
		"\x07":    "\\x07",   // Bell character
		"\x08":    "\\x08",   // Backspace
		"\x0c":    "\\x0c",   // Form feed
		"\x7f":    "\\x7f",   // Delete character
		"\ufeff":  "",        // BOM
		"\u200b":  "",        // Zero-width space
		"\u200c":  "",        // Zero-width non-joiner
		"\u200d":  "",        // Zero-width joiner
		"\u2060":  "",        // Word joiner
	}

	result := message
	for dangerous, replacement := range dangerous {
		result = strings.ReplaceAll(result, dangerous, replacement)
	}

	// Ensure message doesn't end with incomplete escape sequence
	if strings.HasSuffix(result, "\\") && !strings.HasSuffix(result, "\\\\") {
		result += "\\"
	}

	return result
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

	// Check for log injection in format string itself
	if containsLogInjectionPatterns(format) {
		return false
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

// SetSecureLoggerStrict enables or disables strict mode for the default logger
func SetSecureLoggerStrict(strict bool) {
	if defaultSecureLogger != nil {
		defaultSecureLogger.strict = strict
	}
}

// ValidateLogContent validates log content before logging (utility function)
func ValidateLogContent(content string) error {
	if containsLogInjectionPatterns(content) {
		return fmt.Errorf("content contains log injection patterns")
	}

	if len(content) > 10240 {
		return fmt.Errorf("content too long: %d characters", len(content))
	}

	// Check for excessive control characters
	controlCount := 0
	for _, r := range content {
		if unicode.IsControl(r) {
			controlCount++
		}
	}

	if controlCount > 50 {
		return fmt.Errorf("too many control characters: %d", controlCount)
	}

	return nil
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