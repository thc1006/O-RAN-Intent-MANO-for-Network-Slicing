// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// TestCommandInjectionPrevention tests the secure subprocess executor against various injection attacks
func TestCommandInjectionPrevention(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("prevent_basic_command_injection", func(t *testing.T) {
		tests := []struct {
			name        string
			command     string
			args        []string
			shouldFail  bool
			description string
		}{
			{
				name:        "semicolon_injection",
				command:     "ping",
				args:        []string{"-c", "1", "127.0.0.1; rm -rf /tmp/test"},
				shouldFail:  true,
				description: "Should prevent semicolon command injection",
			},
			{
				name:        "pipe_injection",
				command:     "ping",
				args:        []string{"-c", "1", "127.0.0.1 | cat /etc/passwd"},
				shouldFail:  true,
				description: "Should prevent pipe command injection",
			},
			{
				name:        "ampersand_injection",
				command:     "ping",
				args:        []string{"-c", "1", "127.0.0.1 & rm -rf /"},
				shouldFail:  true,
				description: "Should prevent ampersand command injection",
			},
			{
				name:        "backtick_injection",
				command:     "ping",
				args:        []string{"-c", "1", "`whoami`"},
				shouldFail:  true,
				description: "Should prevent backtick command injection",
			},
			{
				name:        "dollar_injection",
				command:     "ping",
				args:        []string{"-c", "1", "$(whoami)"},
				shouldFail:  true,
				description: "Should prevent dollar command injection",
			},
			{
				name:        "redirect_injection",
				command:     "ping",
				args:        []string{"-c", "1", "127.0.0.1 > /tmp/output"},
				shouldFail:  true,
				description: "Should prevent redirect injection",
			},
			{
				name:        "legitimate_ping",
				command:     "ping",
				args:        []string{"-c", "1", "127.0.0.1"},
				shouldFail:  false,
				description: "Should allow legitimate ping command",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, tt.command, tt.args...)

				if tt.shouldFail {
					assert.Error(t, err, tt.description)
					assert.True(t, strings.Contains(err.Error(), "argument validation failed") ||
						strings.Contains(err.Error(), "dangerous characters"),
						"Error should indicate argument validation failure")
				} else {
					// Note: legitimate commands might still fail due to network/system issues
					// but should not fail due to security validation
					if err != nil && strings.Contains(err.Error(), "argument validation failed") {
						t.Errorf("Legitimate command failed security validation: %v", err)
					}
				}
			})
		}
	})

	t.Run("prevent_path_traversal_injection", func(t *testing.T) {
		tests := []struct {
			name       string
			command    string
			args       []string
			shouldFail bool
		}{
			{
				name:       "path_traversal_dots",
				command:    "cat",
				args:       []string{"../../../etc/passwd"},
				shouldFail: true,
			},
			{
				name:       "path_traversal_encoded",
				command:    "cat",
				args:       []string{"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd"},
				shouldFail: true,
			},
			{
				name:       "path_traversal_mixed",
				command:    "cat",
				args:       []string{"logs/../../../etc/passwd"},
				shouldFail: true,
			},
			{
				name:       "legitimate_relative_path",
				command:    "cat",
				args:       []string{"./test.txt"},
				shouldFail: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, tt.command, tt.args...)

				if tt.shouldFail {
					assert.Error(t, err, "Should prevent path traversal")
				}
			})
		}
	})

	t.Run("prevent_environment_variable_injection", func(t *testing.T) {
		tests := []struct {
			name       string
			command    string
			args       []string
			shouldFail bool
		}{
			{
				name:       "env_var_home",
				command:    "ping",
				args:       []string{"-c", "1", "$HOME"},
				shouldFail: true,
			},
			{
				name:       "env_var_path",
				command:    "ping",
				args:       []string{"-c", "1", "${PATH}"},
				shouldFail: true,
			},
			{
				name:       "env_var_malicious",
				command:    "ping",
				args:       []string{"-c", "1", "$MALICIOUS_VAR"},
				shouldFail: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, tt.command, tt.args...)

				if tt.shouldFail {
					assert.Error(t, err, "Should prevent environment variable injection")
				}
			})
		}
	})
}

// TestCommandAllowlistEnforcement tests that only allowed commands can be executed
func TestCommandAllowlistEnforcement(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("reject_disallowed_commands", func(t *testing.T) {
		dangerousCommands := []string{
			"rm",
			"dd",
			"mkfs",
			"fdisk",
			"mount",
			"umount",
			"systemctl",
			"service",
			"chmod",
			"chown",
			"su",
			"sudo",
			"passwd",
			"userdel",
			"useradd",
			"crontab",
			"at",
			"nohup",
			"screen",
			"tmux",
			"nc",
			"netcat",
			"telnet",
			"ssh",
			"scp",
			"rsync",
			"wget",
			"curl",
			"lynx",
			"w3m",
			"sh",
			"bash",
			"zsh",
			"fish",
			"perl",
			"python",
			"ruby",
			"node",
			"php",
			"java",
			"gcc",
			"make",
			"cmake",
			"apt",
			"yum",
			"dnf",
			"pacman",
			"brew",
			"pip",
			"npm",
			"gem",
		}

		for _, cmd := range dangerousCommands {
			t.Run(fmt.Sprintf("reject_%s", cmd), func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, cmd, "--help")
				assert.Error(t, err, "Should reject dangerous command: %s", cmd)
				assert.Contains(t, err.Error(), "command not in allowlist")
			})
		}
	})

	t.Run("allow_safe_commands", func(t *testing.T) {
		safeCommands := []string{
			"ping",
			"iperf3",
			"tc",
			"ip",
			"bridge",
		}

		for _, cmd := range safeCommands {
			t.Run(fmt.Sprintf("allow_%s", cmd), func(t *testing.T) {
				// We expect these commands to be allowed by the security layer
				// They might still fail due to missing binaries or invalid args
				_, err := executor.SecureExecute(ctx, cmd, "--invalid-arg-for-testing")

				// Should not fail due to allowlist rejection
				if err != nil {
					assert.False(t, strings.Contains(err.Error(), "command not in allowlist"),
						"Command %s should be in allowlist", cmd)
				}
			})
		}
	})
}

// TestArgumentValidationBypass tests for attempts to bypass argument validation
func TestArgumentValidationBypass(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("unicode_bypass_attempts", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{
				name: "unicode_semicolon",
				args: []string{"-c", "1", "127.0.0.1﹔rm -rf /"},
			},
			{
				name: "unicode_pipe",
				args: []string{"-c", "1", "127.0.0.1｜cat /etc/passwd"},
			},
			{
				name: "unicode_ampersand",
				args: []string{"-c", "1", "127.0.0.1＆rm -rf /"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, "ping", tt.args...)
				assert.Error(t, err, "Should prevent unicode bypass attempts")
			})
		}
	})

	t.Run("encoding_bypass_attempts", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{
				name: "url_encoded_semicolon",
				args: []string{"-c", "1", "127.0.0.1%3Brm%20-rf%20/"},
			},
			{
				name: "html_encoded_pipe",
				args: []string{"-c", "1", "127.0.0.1&#124;cat /etc/passwd"},
			},
			{
				name: "hex_encoded_ampersand",
				args: []string{"-c", "1", "127.0.0.1\\x26rm -rf /"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, "ping", tt.args...)
				assert.Error(t, err, "Should prevent encoding bypass attempts")
			})
		}
	})

	t.Run("null_byte_injection", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{
				name: "null_byte_termination",
				args: []string{"-c", "1", "127.0.0.1\x00; rm -rf /"},
			},
			{
				name: "null_byte_middle",
				args: []string{"-c", "1", "127\x00.0.0.1"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, "ping", tt.args...)
				assert.Error(t, err, "Should prevent null byte injection")
			})
		}
	})
}

// TestTimeoutEnforcement tests that commands respect timeout settings
func TestTimeoutEnforcement(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()

	// Register a command with very short timeout for testing
	shortTimeoutCmd := &security.AllowedCommand{
		Command:     "sleep",
		AllowedArgs: map[string]bool{},
		ArgPatterns: []string{`^\d+$`},
		MaxArgs:     2,
		Timeout:     100 * time.Millisecond,
		Description: "Sleep command for timeout testing",
	}

	err := executor.RegisterCommand(shortTimeoutCmd)
	require.NoError(t, err)

	t.Run("enforce_command_timeout", func(t *testing.T) {
		ctx := context.Background()

		start := time.Now()
		_, err := executor.SecureExecute(ctx, "sleep", "10")
		duration := time.Since(start)

		assert.Error(t, err, "Should timeout")
		assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "timeout"),
			"Error should indicate timeout")
		assert.True(t, duration < 1*time.Second, "Should timeout quickly")
	})

	t.Run("respect_context_timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := executor.SecureExecute(ctx, "sleep", "10")
		duration := time.Since(start)

		assert.Error(t, err, "Should timeout")
		assert.True(t, duration < 200*time.Millisecond, "Should respect context timeout")
	})
}

// TestOutputSizeLimits tests that command output is limited to prevent DoS
func TestOutputSizeLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping output size limit test in short mode")
	}

	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	// Register a command that can generate large output
	yesCmd := &security.AllowedCommand{
		Command:     "yes",
		AllowedArgs: map[string]bool{},
		ArgPatterns: []string{`^[a-zA-Z0-9\s]+$`},
		MaxArgs:     2,
		Timeout:     2 * time.Second,
		Description: "Yes command for output testing",
	}

	err := executor.RegisterCommand(yesCmd)
	require.NoError(t, err)

	t.Run("limit_large_output", func(t *testing.T) {
		// Check if 'yes' command is available
		if _, err := exec.LookPath("yes"); err != nil {
			t.Skip("'yes' command not available on this system")
		}

		output, err := executor.SecureExecute(ctx, "yes", "test")

		// Should either error due to timeout or limit output size
		if err == nil {
			assert.True(t, len(output) <= 10*1024*1024,
				"Output should be limited to max size (10MB)")
		} else {
			// Expected to timeout or be limited
			assert.True(t, strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "output too large"),
				"Should fail due to timeout or output limit")
		}
	})
}

// TestConcurrentExecutionSafety tests thread safety of the secure executor
func TestConcurrentExecutionSafety(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("concurrent_command_execution", func(t *testing.T) {
		const numGoroutines = 50
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				_, err := executor.SecureExecute(ctx, "ping", "-c", "1", "127.0.0.1")
				results <- err
			}(i)
		}

		// Collect all results
		var successCount int
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-results:
				if err == nil {
					successCount++
				} else if !strings.Contains(err.Error(), "ping") &&
					!strings.Contains(err.Error(), "network") {
					// Unexpected error that's not network-related
					t.Errorf("Unexpected error: %v", err)
				}
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for concurrent executions")
			}
		}

		// At least some should succeed or fail predictably
		t.Logf("Successful executions: %d/%d", successCount, numGoroutines)
	})
}

// TestCommandInjectionBypassAttempts tests various sophisticated attack vectors
func TestCommandInjectionBypassAttempts(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("format_string_attacks", func(t *testing.T) {
		attacks := [][]string{
			{"-c", "1", "%s%s%s%s"},
			{"-c", "1", "%x%x%x%x"},
			{"-c", "1", "%n%n%n%n"},
			{"-c", "1", "127.0.0.1%s"},
		}

		for i, attack := range attacks {
			t.Run(fmt.Sprintf("format_string_%d", i), func(t *testing.T) {
				_, err := executor.SecureExecute(ctx, "ping", attack...)
				assert.Error(t, err, "Should prevent format string attacks")
			})
		}
	})

	t.Run("buffer_overflow_attempts", func(t *testing.T) {
		// Very long arguments that might cause buffer overflows
		longArg := strings.Repeat("A", 10000)

		_, err := executor.SecureExecute(ctx, "ping", "-c", "1", longArg)
		assert.Error(t, err, "Should prevent buffer overflow attempts")
	})

	t.Run("race_condition_exploits", func(t *testing.T) {
		// Attempt to exploit race conditions by rapid command submission
		const numRapidRequests = 100
		errors := make(chan error, numRapidRequests)

		for i := 0; i < numRapidRequests; i++ {
			go func() {
				_, err := executor.SecureExecute(ctx, "ping", "-c", "1", "127.0.0.1; echo EXPLOIT")
				errors <- err
			}()
		}

		for i := 0; i < numRapidRequests; i++ {
			select {
			case err := <-errors:
				assert.Error(t, err, "Should reject malicious commands even under race conditions")
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for rapid requests")
			}
		}
	})
}

// BenchmarkCommandValidation benchmarks the performance of command validation
func BenchmarkCommandValidation(b *testing.B) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	b.Run("legitimate_command", func(b *testing.B) {
		args := []string{"-c", "1", "127.0.0.1"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test legitimate command execution including validation
			_, err := executor.SecureExecute(ctx, "ping", args...)
			_ = err // Don't care about result, just performance
		}
	})

	b.Run("malicious_command", func(b *testing.B) {
		args := []string{"-c", "1", "127.0.0.1; rm -rf /"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test malicious command detection including validation
			_, err := executor.SecureExecute(ctx, "ping", args...)
			_ = err // Should return error, just testing performance
		}
	})
}

// TestCustomValidatorIntegration tests the integration with custom validators
func TestCustomValidatorIntegration(t *testing.T) {
	executor := security.NewSecureSubprocessExecutor()
	ctx := context.Background()

	t.Run("custom_validator_security", func(t *testing.T) {
		// Custom validator that checks for specific security patterns
		securityValidator := func(args []string) error {
			for _, arg := range args {
				// Check for suspicious patterns
				if strings.Contains(arg, "eval") ||
					strings.Contains(arg, "exec") ||
					strings.Contains(arg, "system") {
					return fmt.Errorf("security violation: suspicious pattern detected in %s", arg)
				}
			}
			return nil
		}

		tests := []struct {
			name       string
			args       []string
			shouldFail bool
		}{
			{
				name:       "clean_args",
				args:       []string{"-c", "1", "127.0.0.1"},
				shouldFail: false,
			},
			{
				name:       "eval_pattern",
				args:       []string{"-c", "1", "eval(dangerous_code)"},
				shouldFail: true,
			},
			{
				name:       "exec_pattern",
				args:       []string{"-c", "1", "exec('rm -rf /')"},
				shouldFail: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := executor.SecureExecuteWithValidation(ctx, "ping", securityValidator, tt.args...)

				if tt.shouldFail {
					assert.Error(t, err, "Custom validator should reject suspicious patterns")
					assert.Contains(t, err.Error(), "security violation")
				}
			})
		}
	})
}
