package security

import (
	"os"
	"path/filepath"
	"testing"
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
			expectedViolations: 5,
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
			scanner := NewSecurityScanner()

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
					for i, v := range violations {
						t.Logf("Violation %d: Type=%s, Severity=%s, Message=%s", i+1, v.Type, v.Severity, v.Message)
					}
				}
			}
		})
	}
}

func TestSecurityScanner_PrivilegedContainer(t *testing.T) {
	scanner := NewSecurityScanner()

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
	scanner := NewSecurityScanner()

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
	scanner := NewSecurityScanner()

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
	scanner := NewSecurityScanner()

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

func TestSecurityScanner_GenerateReport(t *testing.T) {
	scanner := NewSecurityScanner()

	err := scanner.ScanPackage("../testdata/insecure-package")
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	reportPath := "../reports/test-security-scan.md"

	// Ensure reports directory exists
	os.MkdirAll("../reports", 0755)

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

	scanner := NewSecurityScanner()
	err := scanner.ScanPackage(tmpDir)

	if err != nil {
		t.Fatalf("scan failed on empty package: %v", err)
	}

	violations := scanner.GetViolations()
	if len(violations) != 0 {
		t.Errorf("expected no violations for empty package, got %d", len(violations))
	}
}