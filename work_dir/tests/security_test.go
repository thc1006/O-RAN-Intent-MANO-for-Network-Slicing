package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func TestSecurityScanner_ScanPackage_InsecureManifests(t *testing.T) {
	tests := []struct {
		name              string
		packagePath       string
		expectedViolations int
		expectError       bool
	}{
		{
			name:              "insecure package with multiple violations",
			packagePath:       "../testdata/insecure-package",
			expectedViolations: 5, // privileged, root user, no network policy, etc.
			expectError:       false,
		},
		{
			name:              "secure package with no violations",
			packagePath:       "../testdata/secure-package",
			expectedViolations: 0,
			expectError:       false,
		},
		{
			name:              "non-existent package",
			packagePath:       "../testdata/non-existent",
			expectedViolations: 0,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := security.NewSecurityScanner()

			err := scanner.ScanPackage(tt.packagePath)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectError {
				violations := scanner.GetViolations()
				if len(violations) < tt.expectedViolations {
					t.Errorf("expected at least %d violations, got %d", tt.expectedViolations, len(violations))
				}
			}
		})
	}
}

func TestSecurityScanner_PrivilegedContainer(t *testing.T) {
	scanner := security.NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	violations := scanner.GetViolations()
	foundPrivileged := false

	for _, v := range violations {
		if v.Type == "PrivilegedContainer" {
			foundPrivileged = true
			if v.Severity != "Critical" {
				t.Errorf("privileged container should be Critical severity, got %s", v.Severity)
			}
			break
		}
	}

	if !foundPrivileged {
		t.Error("expected to find PrivilegedContainer violation")
	}
}

func TestSecurityScanner_RunAsRoot(t *testing.T) {
	scanner := security.NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	violations := scanner.GetViolations()
	foundRootUser := false

	for _, v := range violations {
		if v.Type == "RunAsRoot" {
			foundRootUser = true
			if v.Severity != "High" {
				t.Errorf("root user should be High severity, got %s", v.Severity)
			}
			break
		}
	}

	if !foundRootUser {
		t.Error("expected to find RunAsRoot violation")
	}
}

func TestSecurityScanner_MissingNetworkPolicy(t *testing.T) {
	scanner := security.NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	violations := scanner.GetViolations()
	foundMissingNetPolicy := false

	for _, v := range violations {
		if v.Type == "MissingNetworkPolicy" {
			foundMissingNetPolicy = true
			break
		}
	}

	if !foundMissingNetPolicy {
		t.Error("expected to find MissingNetworkPolicy violation")
	}
}

func TestSecurityScanner_HostNetworkAccess(t *testing.T) {
	scanner := security.NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	violations := scanner.GetViolations()
	foundHostNetwork := false

	for _, v := range violations {
		if v.Type == "HostNetworkAccess" {
			foundHostNetwork = true
			if v.Severity != "High" {
				t.Errorf("host network access should be High severity, got %s", v.Severity)
			}
			break
		}
	}

	if !foundHostNetwork {
		t.Error("expected to find HostNetworkAccess violation")
	}
}

func TestPolicyEnforcer_StrictMode(t *testing.T) {
	tests := []struct {
		name        string
		strictMode  bool
		violations  []security.SecurityViolation
		expectError bool
	}{
		{
			name:       "strict mode with critical violations",
			strictMode: true,
			violations: []security.SecurityViolation{
				{
					Type:     "PrivilegedContainer",
					Severity: "Critical",
					Message:  "test violation",
				},
			},
			expectError: true,
		},
		{
			name:       "strict mode with high violations",
			strictMode: true,
			violations: []security.SecurityViolation{
				{
					Type:     "RunAsRoot",
					Severity: "High",
					Message:  "test violation",
				},
			},
			expectError: true,
		},
		{
			name:       "strict mode with medium violations",
			strictMode: true,
			violations: []security.SecurityViolation{
				{
					Type:     "MissingReadOnlyRootFS",
					Severity: "Medium",
					Message:  "test violation",
				},
			},
			expectError: false,
		},
		{
			name:       "non-strict mode with critical violations",
			strictMode: false,
			violations: []security.SecurityViolation{
				{
					Type:     "PrivilegedContainer",
					Severity: "Critical",
					Message:  "test violation",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enforcer := security.NewPolicyEnforcer(tt.strictMode)

			err := enforcer.Enforce(tt.violations)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestPolicyEnforcer_ViolationCounts(t *testing.T) {
	violations := []security.SecurityViolation{
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "RunAsRoot", Severity: "High"},
		{Type: "MissingReadOnlyRootFS", Severity: "Medium"},
		{Type: "MissingSeccomp", Severity: "Low"},
	}

	enforcer := security.NewPolicyEnforcer(true)
	err := enforcer.Enforce(violations)

	if err == nil {
		t.Error("expected error with critical violations")
	}

	// Check error message contains counts
	if err != nil && !contains(err.Error(), "2 critical") {
		t.Errorf("expected error message to contain counts, got: %v", err)
	}
}

func TestSecurityScanner_GenerateReport(t *testing.T) {
	scanner := security.NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	reportPath := "../reports/test-security-scan.md"
	err = scanner.GenerateReport(reportPath)
	if err != nil {
		t.Fatalf("failed to generate report: %v", err)
	}

	// Verify report exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("report file was not created")
	}

	// Clean up
	defer os.Remove(reportPath)
}

func TestSecurityScanner_EmptyPackage(t *testing.T) {
	// Create temporary empty directory
	tmpDir := filepath.Join(os.TempDir(), "empty-package")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	scanner := security.NewSecurityScanner()
	err := scanner.ScanPackage(tmpDir)

	if err != nil {
		t.Fatalf("scan failed on empty package: %v", err)
	}

	violations := scanner.GetViolations()
	if len(violations) != 0 {
		t.Errorf("expected no violations for empty package, got %d", len(violations))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}