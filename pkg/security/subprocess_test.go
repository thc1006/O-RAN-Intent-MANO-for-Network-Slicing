// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSecureSubprocessExecutor_NewSecureSubprocessExecutor(t *testing.T) {
	executor := NewSecureSubprocessExecutor()

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}

	if executor.defaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", executor.defaultTimeout)
	}

	if executor.maxOutputSize != 10*1024*1024 {
		t.Errorf("Expected max output size of 10MB, got %d", executor.maxOutputSize)
	}

	// Check that default commands are registered
	allowedCommands := []string{"iperf3", "tc", "ip", "bridge", "ping", "pkill", "cat", "pgrep"}
	for _, cmd := range allowedCommands {
		if _, exists := executor.allowedCommands[cmd]; !exists {
			t.Errorf("Expected command %s to be registered by default", cmd)
		}
	}
}

func TestSecureSubprocessExecutor_RegisterCommand(t *testing.T) {
	executor := NewSecureSubprocessExecutor()

	tests := []struct {
		name    string
		command *AllowedCommand
		wantErr bool
	}{
		{
			name: "valid command",
			command: &AllowedCommand{
				Command:     "test",
				AllowedArgs: map[string]bool{"-h": true},
				MaxArgs:     5,
				Timeout:     10 * time.Second,
				Description: "Test command",
			},
			wantErr: false,
		},
		{
			name: "empty command name",
			command: &AllowedCommand{
				Command: "",
			},
			wantErr: true,
		},
		{
			name: "command with dangerous characters",
			command: &AllowedCommand{
				Command: "test;rm",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.RegisterCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecureSubprocessExecutor_SecureExecute(t *testing.T) {
	executor := NewSecureSubprocessExecutor()
	ctx := context.Background()

	tests := []struct {
		name      string
		command   string
		args      []string
		wantErr   bool
		errorText string
	}{
		{
			name:      "command not in allowlist",
			command:   "unknown_command",
			args:      []string{},
			wantErr:   true,
			errorText: "command not in allowlist",
		},
		{
			name:      "too many arguments",
			command:   "ping",
			args:      make([]string, 20), // More than ping's MaxArgs (15)
			wantErr:   true,
			errorText: "too many arguments",
		},
		{
			name:      "invalid argument - dangerous characters",
			command:   "ping",
			args:      []string{"-c", "3", "example.com; rm -rf /"},
			wantErr:   true,
			errorText: "argument validation failed",
		},
		{
			name:      "echo command with simple args (should work if registered)",
			command:   "echo",
			args:      []string{"hello"},
			wantErr:   true, // Will fail because echo is not in default allowlist
			errorText: "command not in allowlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.SecureExecute(ctx, tt.command, tt.args...)

			if (err != nil) != tt.wantErr {
				t.Errorf("SecureExecute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errorText) {
				t.Errorf("SecureExecute() error = %v, should contain %v", err, tt.errorText)
			}
		})
	}
}

func TestValidateIPerfArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid server args",
			args:    []string{"-s", "-p", "5201"},
			wantErr: false,
		},
		{
			name:    "valid client args",
			args:    []string{"-c", "127.0.0.1", "-p", "5201", "-t", "10"},
			wantErr: false,
		},
		{
			name:    "both server and client",
			args:    []string{"-s", "-c", "127.0.0.1"},
			wantErr: true,
		},
		{
			name:    "neither server nor client",
			args:    []string{"-p", "5201"},
			wantErr: true,
		},
		{
			name:    "invalid port",
			args:    []string{"-s", "-p", "99999"},
			wantErr: true,
		},
		{
			name:    "invalid duration",
			args:    []string{"-c", "127.0.0.1", "-t", "5000"},
			wantErr: true,
		},
		{
			name:    "invalid bandwidth",
			args:    []string{"-c", "127.0.0.1", "-u", "-b", "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid server IP",
			args:    []string{"-c", "999.999.999.999"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPerfArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPerfArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTCArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid qdisc show",
			args:    []string{"qdisc", "show", "dev", "eth0"},
			wantErr: false,
		},
		{
			name:    "valid qdisc add",
			args:    []string{"qdisc", "add", "dev", "eth0", "root", "handle", "1:", "htb"},
			wantErr: false,
		},
		{
			name:    "qdisc without dev",
			args:    []string{"qdisc", "show"},
			wantErr: true,
		},
		{
			name:    "invalid interface name",
			args:    []string{"qdisc", "show", "dev", "eth0; rm -rf /"},
			wantErr: true,
		},
		{
			name:    "invalid rate format",
			args:    []string{"qdisc", "add", "dev", "eth0", "rate", "10MB"},
			wantErr: true,
		},
		{
			name:    "valid rate format",
			args:    []string{"qdisc", "add", "dev", "eth0", "rate", "10Mbit"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTCArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTCArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIPArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "too few arguments",
			args:    []string{"link"},
			wantErr: true,
		},
		{
			name:    "valid link show",
			args:    []string{"link", "show", "eth0"},
			wantErr: false,
		},
		{
			name:    "valid link add vxlan",
			args:    []string{"link", "add", "vxlan0", "type", "vxlan", "id", "100"},
			wantErr: false,
		},
		{
			name:    "invalid link add type",
			args:    []string{"link", "add", "test0", "type", "dummy"},
			wantErr: true,
		},
		{
			name:    "valid link delete",
			args:    []string{"link", "delete", "vxlan0"},
			wantErr: false,
		},
		{
			name:    "invalid VNI",
			args:    []string{"link", "add", "vxlan0", "type", "vxlan", "id", "99999999"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIPArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIPArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecureSubprocessExecutor_ArgumentValidation(t *testing.T) {
	executor := NewSecureSubprocessExecutor()

	// Test command with strict argument validation
	cmd := &AllowedCommand{
		Command:     "test_cmd",
		AllowedArgs: map[string]bool{"-a": true, "-b": true},
		ArgPatterns: []string{`^\d+$`}, // Only numbers
		MaxArgs:     5,
		Timeout:     5 * time.Second,
		Description: "Test command",
	}

	_ = executor.RegisterCommand(cmd)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "allowed argument",
			args:    []string{"-a"},
			wantErr: false,
		},
		{
			name:    "pattern matching argument",
			args:    []string{"123"},
			wantErr: false,
		},
		{
			name:    "disallowed argument",
			args:    []string{"-c"},
			wantErr: true,
		},
		{
			name:    "argument not matching pattern",
			args:    []string{"abc"},
			wantErr: true,
		},
		{
			name:    "dangerous argument",
			args:    []string{"-a", "$(rm -rf /)"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateArguments(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArguments() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecureSubprocessExecutor_Timeout(t *testing.T) {
	executor := NewSecureSubprocessExecutor()

	// Register a command with very short timeout
	shortTimeoutCmd := &AllowedCommand{
		Command:     "sleep",
		AllowedArgs: map[string]bool{},
		ArgPatterns: []string{`^\d+$`},
		MaxArgs:     2,
		Timeout:     100 * time.Millisecond, // Very short timeout
		Description: "Sleep command for timeout testing",
	}

	_ = executor.RegisterCommand(shortTimeoutCmd)

	ctx := context.Background()

	// This should timeout
	_, err := executor.SecureExecute(ctx, "sleep", "1") // Sleep for 1 second, but timeout is 100ms

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestSecureSubprocessExecutor_SecureExecuteWithValidation(t *testing.T) {
	executor := NewSecureSubprocessExecutor()
	ctx := context.Background()

	// Custom validator that rejects arguments containing "bad"
	customValidator := func(args []string) error {
		for _, arg := range args {
			if strings.Contains(arg, "bad") {
				return fmt.Errorf("argument contains 'bad': %s", arg)
			}
		}
		return nil
	}

	tests := []struct {
		name      string
		command   string
		validator func([]string) error
		args      []string
		wantErr   bool
	}{
		{
			name:      "custom validation passes",
			command:   "ping",
			validator: customValidator,
			args:      []string{"-c", "1", "127.0.0.1"},
			wantErr:   false, // May still fail due to command execution, but validation should pass
		},
		{
			name:      "custom validation fails",
			command:   "ping",
			validator: customValidator,
			args:      []string{"-c", "1", "bad.example.com"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executor.SecureExecuteWithValidation(ctx, tt.command, tt.validator, tt.args...)

			if tt.wantErr && err == nil {
				t.Error("Expected error from custom validation, got nil")
			}

			if !tt.wantErr && err != nil && strings.Contains(err.Error(), "custom validation failed") {
				t.Errorf("Unexpected custom validation error: %v", err)
			}
		})
	}
}

func BenchmarkSecureSubprocessExecutor_ArgumentValidation(b *testing.B) {
	executor := NewSecureSubprocessExecutor()
	cmd := executor.allowedCommands["ping"]
	args := []string{"-c", "1", "127.0.0.1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.validateArguments(cmd, args)
	}
}

func BenchmarkValidateIPerfArgs(b *testing.B) {
	args := []string{"-c", "127.0.0.1", "-p", "5201", "-t", "10"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateIPerfArgs(args)
	}
}

func BenchmarkValidateCommandArgument(b *testing.B) {
	arg := "test-argument-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateCommandArgument(arg)
	}
}

// Test helper functions
func TestParseIntSafe(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 123},
		{"0", 0},
		{"-456", -456},
		{"abc", 0}, // Invalid input should return 0
		{"", 0},    // Empty input should return 0
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseIntSafe(tt.input)
			if result != tt.expected {
				t.Errorf("parseIntSafe(%s) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseUint32Safe(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
	}{
		{"123", 123},
		{"0", 0},
		{"4294967295", 4294967295}, // Max uint32
		{"abc", 0},                 // Invalid input should return 0
		{"", 0},                    // Empty input should return 0
		{"-123", 0},                // Negative should return 0
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseUint32Safe(tt.input)
			if result != tt.expected {
				t.Errorf("parseUint32Safe(%s) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// Integration test - only run if system has required commands
func TestSecureSubprocessExecutor_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	executor := NewSecureSubprocessExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test ping to localhost (most systems should have this)
	output, err := executor.SecureExecute(ctx, "ping", "-c", "1", "127.0.0.1")

	if err != nil {
		t.Logf("Ping command failed (this may be expected on some systems): %v", err)
		return
	}

	if len(output) == 0 {
		t.Error("Expected output from ping command")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "127.0.0.1") {
		t.Error("Expected ping output to contain target IP")
	}
}
