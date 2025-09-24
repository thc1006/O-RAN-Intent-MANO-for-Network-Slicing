package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePathValidationSecurity(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{
			name:        "valid relative path",
			path:        "config/test.yaml",
			expectError: false,
			description: "Normal relative path should be allowed",
		},
		{
			name:        "directory traversal with ../",
			path:        "../../../etc/passwd",
			expectError: true,
			description: "Directory traversal should be blocked",
		},
		{
			name:        "directory traversal with ..\\ (Windows)",
			path:        "..\\..\\..\\windows\\system32\\config\\sam",
			expectError: true,
			description: "Windows-style directory traversal should be blocked",
		},
		{
			name:        "null byte injection",
			path:        "config.yaml\x00.txt",
			expectError: true,
			description: "Null byte injection should be blocked",
		},
		{
			name:        "double encoded directory traversal",
			path:        "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			expectError: false, // URL decoding is not our responsibility
			description: "Double encoded traversal (will be caught by web server)",
		},
		{
			name:        "very long path",
			path:        strings.Repeat("a", 5000),
			expectError: true,
			description: "Extremely long paths should be rejected",
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			description: "Empty paths should be rejected",
		},
		{
			name:        "path with only dots",
			path:        "....",
			expectError: true, // Actually, this should be blocked as it's suspicious
			description: "Multiple dots should be blocked as suspicious",
		},
		{
			name:        "mixed separators attack",
			path:        "config/../../../etc/passwd",
			expectError: true,
			description: "Mixed separators traversal should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none. %s", tt.path, tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v. %s", tt.path, err, tt.description)
			}
		})
	}
}

func TestSecureJoinPath(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		components  []string
		expectError bool
		description string
	}{
		{
			name:        "valid join",
			base:        "/tmp",
			components:  []string{"test", "file.txt"},
			expectError: false,
			description: "Normal path join should work",
		},
		{
			name:        "traversal in component",
			base:        "/tmp",
			components:  []string{"../", "etc", "passwd"},
			expectError: true,
			description: "Directory traversal in component should be blocked",
		},
		{
			name:        "absolute path in component",
			base:        "/tmp",
			components:  []string{"/etc/passwd"},
			expectError: true,
			description: "Absolute path in component should be blocked",
		},
		{
			name:        "empty component",
			base:        "/tmp",
			components:  []string{"", "file.txt"},
			expectError: false,
			description: "Empty components should be handled safely",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SecureJoinPath(tt.base, tt.components...)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got result: %s. %s", result, tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
			}
			if !tt.expectError && err == nil {
				// Verify result is under base directory
				relPath, err := filepath.Rel(tt.base, result)
				if err != nil || strings.HasPrefix(relPath, "..") {
					t.Errorf("Result path escapes base directory: %s", result)
				}
			}
		})
	}
}

func TestValidateAndCleanPath(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		allowedExtensions []string
		expectError       bool
		expectedCleanPath string
		description       string
	}{
		{
			name:              "valid path with extension check",
			path:              "config/test.yaml",
			allowedExtensions: []string{".yaml", ".yml"},
			expectError:       false,
			expectedCleanPath: "", // Don't check exact path due to OS differences
			description:       "Valid path with allowed extension",
		},
		{
			name:              "invalid extension",
			path:              "config/test.exe",
			allowedExtensions: []string{".yaml", ".yml"},
			expectError:       true,
			description:       "Invalid file extension should be rejected",
		},
		{
			name:              "path with traversal gets cleaned",
			path:              "config/../config/test.yaml",
			allowedExtensions: []string{".yaml"},
			expectError:       true, // Should still error due to .. detection
			description:       "Path with traversal should be rejected even after cleaning",
		},
		{
			name:              "case insensitive extension check",
			path:              "config/test.YAML",
			allowedExtensions: []string{".yaml"},
			expectError:       false,
			expectedCleanPath: "", // Don't check exact path due to OS differences
			description:       "Extension check should be case insensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndCleanPath(tt.path, tt.allowedExtensions)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got result: %s. %s", result, tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
			}
			if !tt.expectError && err == nil && tt.expectedCleanPath != "" {
				if result != tt.expectedCleanPath {
					t.Errorf("Expected clean path %s, got %s", tt.expectedCleanPath, result)
				}
			}
		})
	}
}

func TestFilePathValidatorWithAllowlist(t *testing.T) {
	validator := NewFilePathValidator()

	// Add allowed directory
	validator.AddAllowedDirectory(AllowedDirectory{
		Path:        "/tmp/test",
		Extensions:  []string{".yaml", ".json"},
		Recursive:   true,
		Description: "Test directory",
	})

	tests := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{
			name:        "allowed path in allowlist",
			path:        "/tmp/test/config.yaml",
			expectError: false,
			description: "Path in allowed directory should pass",
		},
		{
			name:        "disallowed path outside allowlist",
			path:        "/etc/passwd",
			expectError: true,
			description: "Path outside allowed directories should fail",
		},
		{
			name:        "allowed directory but wrong extension",
			path:        "/tmp/test/config.exe",
			expectError: true,
			description: "Wrong extension in allowed directory should fail",
		},
		{
			name:        "traversal from allowed directory",
			path:        "/tmp/test/../../../etc/passwd",
			expectError: true,
			description: "Traversal from allowed directory should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none. %s", tt.path, tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v. %s", tt.path, err, tt.description)
			}
		})
	}
}

func TestSecureFileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "security_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test SecureCreateFile
	testFile := filepath.Join(tmpDir, "test.txt")
	file, err := SecureCreateFile(testFile)
	if err != nil {
		t.Errorf("SecureCreateFile failed: %v", err)
	} else {
		file.Close()

		// Check file permissions
		info, err := os.Stat(testFile)
		if err != nil {
			t.Errorf("Failed to stat created file: %v", err)
		} else {
			mode := info.Mode()
			if mode.Perm() != SecureFileMode {
				t.Errorf("Expected file mode %o, got %o", SecureFileMode, mode.Perm())
			}
		}
	}

	// Test SecureCreateDir
	testDir := filepath.Join(tmpDir, "testdir")
	err = SecureCreateDir(testDir)
	if err != nil {
		t.Errorf("SecureCreateDir failed: %v", err)
	} else {
		// Check directory permissions
		info, err := os.Stat(testDir)
		if err != nil {
			t.Errorf("Failed to stat created directory: %v", err)
		} else {
			mode := info.Mode()
			if mode.Perm() != SecureDirMode {
				t.Errorf("Expected directory mode %o, got %o", SecureDirMode, mode.Perm())
			}
		}
	}
}

func TestValidatorCreationFunctions(t *testing.T) {
	workingDir := "/tmp/test"

	// Test Kubernetes validator
	validator := CreateValidatorForKubernetes(workingDir)
	if len(validator.allowedDirs) == 0 {
		t.Error("Kubernetes validator should have allowed directories")
	}

	// Test config validator
	validator = CreateValidatorForConfig(workingDir)
	if len(validator.allowedDirs) == 0 {
		t.Error("Config validator should have allowed directories")
	}

	// Test logs validator
	validator = CreateValidatorForLogsEnhanced(workingDir)
	if len(validator.allowedDirs) == 0 {
		t.Error("Logs validator should have allowed directories")
	}

	// Test test data validator
	validator = CreateValidatorForTestData(workingDir)
	if len(validator.allowedDirs) == 0 {
		t.Error("Test data validator should have allowed directories")
	}

	// Test snapshots validator
	validator = CreateValidatorForSnapshots(workingDir)
	if len(validator.allowedDirs) == 0 {
		t.Error("Snapshots validator should have allowed directories")
	}
}

// Benchmark file path validation performance
func BenchmarkValidateFilePathSecurity(b *testing.B) {
	testPath := "config/test.yaml"
	for i := 0; i < b.N; i++ {
		_ = ValidateFilePath(testPath)
	}
}

func BenchmarkValidateFilePathWithTraversalAttack(b *testing.B) {
	testPath := "../../../etc/passwd"
	for i := 0; i < b.N; i++ {
		_ = ValidateFilePath(testPath)
	}
}
