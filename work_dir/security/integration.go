package security

import (
	"fmt"
	"os"
	"path/filepath"
)

// PackageRenderer integration for security scanning
type SecurePackageRenderer struct {
	scanner  *SecurityScanner
	enforcer *PolicyEnforcer
	config   SecurityIntegrationConfig
}

// SecurityIntegrationConfig holds configuration for security integration
type SecurityIntegrationConfig struct {
	EnableScanning     bool
	StrictMode         bool
	AutoRemediation    bool
	GenerateReports    bool
	ReportOutputPath   string
	FailOnViolations   bool
	PodSecurityLevel   string // Privileged, Baseline, Restricted
	BlockCritical      bool
	BlockHigh          bool
}

// NewSecurePackageRenderer creates a new secure package renderer
func NewSecurePackageRenderer(config SecurityIntegrationConfig) *SecurePackageRenderer {
	enforcer := NewPolicyEnforcer(config.StrictMode)
	enforcer.blockOnCritical = config.BlockCritical
	enforcer.blockOnHigh = config.BlockHigh

	return &SecurePackageRenderer{
		scanner:  NewSecurityScanner(),
		enforcer: enforcer,
		config:   config,
	}
}

// ValidatePackage scans and validates a Nephio package for security issues
func (r *SecurePackageRenderer) ValidatePackage(pkgPath string) (*ValidationResult, error) {
	if !r.config.EnableScanning {
		return &ValidationResult{
			Valid:   true,
			Message: "Security scanning disabled",
		}, nil
	}

	// Scan the package
	if err := r.scanner.ScanPackage(pkgPath); err != nil {
		return nil, fmt.Errorf("security scan failed: %w", err)
	}

	violations := r.scanner.GetViolations()

	// Generate report if configured
	if r.config.GenerateReports {
		reportPath := r.getReportPath(pkgPath)
		if err := r.scanner.GenerateReport(reportPath); err != nil {
			fmt.Printf("Warning: failed to generate security report: %v\n", err)
		} else {
			fmt.Printf("Security report generated: %s\n", reportPath)
		}
	}

	// Enforce policies
	enforcementErr := r.enforcer.Enforce(violations)

	// Check against Pod Security Standards if configured
	var pssValid bool
	var pssFailures []string
	if r.config.PodSecurityLevel != "" {
		pssValid, pssFailures = r.enforcer.ValidateAgainstPodSecurityStandards(
			violations,
			r.config.PodSecurityLevel,
		)
	}

	// Build validation result
	result := &ValidationResult{
		Valid:              enforcementErr == nil && (pssValid || r.config.PodSecurityLevel == ""),
		Message:            r.buildValidationMessage(violations, enforcementErr, pssValid, pssFailures),
		Violations:         violations,
		EnforcementReport:  r.enforcer.GetEnforcementReport(violations),
		RecommendedActions: r.enforcer.GetRecommendedActions(violations),
		Summary:            r.scanner.GetSummary(),
	}

	// Fail if configured and violations found
	if r.config.FailOnViolations && !result.Valid {
		return result, fmt.Errorf("security validation failed: %s", result.Message)
	}

	return result, nil
}

// ValidationResult contains the results of security validation
type ValidationResult struct {
	Valid              bool
	Message            string
	Violations         []SecurityViolation
	EnforcementReport  EnforcementReport
	RecommendedActions []string
	Summary            map[string]interface{}
}

// buildValidationMessage creates a comprehensive validation message
func (r *SecurePackageRenderer) buildValidationMessage(
	violations []SecurityViolation,
	enforcementErr error,
	pssValid bool,
	pssFailures []string,
) string {
	if len(violations) == 0 {
		return "✓ No security violations detected"
	}

	counts := make(map[string]int)
	for _, v := range violations {
		counts[v.Severity]++
	}

	msg := fmt.Sprintf("Security scan found %d violations: ", len(violations))
	if counts["Critical"] > 0 {
		msg += fmt.Sprintf("%d critical, ", counts["Critical"])
	}
	if counts["High"] > 0 {
		msg += fmt.Sprintf("%d high, ", counts["High"])
	}
	if counts["Medium"] > 0 {
		msg += fmt.Sprintf("%d medium, ", counts["Medium"])
	}
	if counts["Low"] > 0 {
		msg += fmt.Sprintf("%d low", counts["Low"])
	}

	if enforcementErr != nil {
		msg += fmt.Sprintf("\nPolicy enforcement: %v", enforcementErr)
	}

	if !pssValid && len(pssFailures) > 0 {
		msg += fmt.Sprintf("\nPod Security Standards (%s) failures: %d", r.config.PodSecurityLevel, len(pssFailures))
	}

	return msg
}

// getReportPath generates the output path for security reports
func (r *SecurePackageRenderer) getReportPath(pkgPath string) string {
	if r.config.ReportOutputPath != "" {
		return r.config.ReportOutputPath
	}

	// Default: create report in the same directory as package
	pkgName := filepath.Base(pkgPath)
	return filepath.Join(filepath.Dir(pkgPath), fmt.Sprintf("security-scan-%s.md", pkgName))
}

// PrintValidationResult prints a formatted validation result
func PrintValidationResult(result *ValidationResult) {
	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("SECURITY VALIDATION REPORT")
	fmt.Println(repeatString("=", 80))

	if result.Valid {
		fmt.Println("✓ VALIDATION PASSED")
	} else {
		fmt.Println("✗ VALIDATION FAILED")
	}

	fmt.Printf("\nStatus: %s\n", result.Message)

	if len(result.Violations) > 0 {
		fmt.Printf("\nTotal Violations: %d\n", len(result.Violations))
		fmt.Printf("  Critical: %d\n", result.EnforcementReport.CriticalCount)
		fmt.Printf("  High: %d\n", result.EnforcementReport.HighCount)
		fmt.Printf("  Medium: %d\n", result.EnforcementReport.MediumCount)
		fmt.Printf("  Low: %d\n", result.EnforcementReport.LowCount)

		if len(result.RecommendedActions) > 0 {
			fmt.Println("\nRecommended Actions:")
			for i, action := range result.RecommendedActions {
				fmt.Printf("  %d. %s\n", i+1, action)
			}
		}
	}

	fmt.Println(repeatString("=", 80) + "\n")
}

// Helper function for string repetition
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// ScanAndValidatePackage is a convenience function for scanning and validating
func ScanAndValidatePackage(pkgPath string, strictMode bool) error {
	config := SecurityIntegrationConfig{
		EnableScanning:   true,
		StrictMode:       strictMode,
		GenerateReports:  true,
		FailOnViolations: strictMode,
		BlockCritical:    strictMode,
		BlockHigh:        strictMode,
		PodSecurityLevel: "Baseline",
	}

	renderer := NewSecurePackageRenderer(config)
	result, err := renderer.ValidatePackage(pkgPath)

	if result != nil {
		PrintValidationResult(result)
	}

	return err
}

// CreateDefaultSecurityConfig creates a default security configuration
func CreateDefaultSecurityConfig() SecurityIntegrationConfig {
	return SecurityIntegrationConfig{
		EnableScanning:   true,
		StrictMode:       true,
		AutoRemediation:  false,
		GenerateReports:  true,
		ReportOutputPath: "work_dir/reports/security-scan.md",
		FailOnViolations: true,
		PodSecurityLevel: "Baseline",
		BlockCritical:    true,
		BlockHigh:        true,
	}
}

// ScanPackageDirectory scans all packages in a directory
func ScanPackageDirectory(dirPath string, config SecurityIntegrationConfig) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	renderer := NewSecurePackageRenderer(config)

	for _, entry := range entries {
		if entry.IsDir() {
			pkgPath := filepath.Join(dirPath, entry.Name())
			result, err := renderer.ValidatePackage(pkgPath)
			if err != nil && config.FailOnViolations {
				return results, fmt.Errorf("validation failed for package %s: %w", entry.Name(), err)
			}
			if result != nil {
				results[entry.Name()] = result
			}
		}
	}

	return results, nil
}