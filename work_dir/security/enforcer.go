package security

import (
	"fmt"
	"strings"
)

// PolicyEnforcer enforces security policies on scan results
type PolicyEnforcer struct {
	strictMode          bool
	allowedViolations   map[string]int
	maxCritical         int
	maxHigh             int
	blockOnCritical     bool
	blockOnHigh         bool
	customRules         []EnforcementRule
}

// EnforcementRule represents a custom enforcement rule
type EnforcementRule struct {
	ViolationType string
	MaxAllowed    int
	BlockOnExceed bool
	Message       string
}

// NewPolicyEnforcer creates a new policy enforcer
func NewPolicyEnforcer(strictMode bool) *PolicyEnforcer {
	return &PolicyEnforcer{
		strictMode:        strictMode,
		allowedViolations: make(map[string]int),
		maxCritical:       0,
		maxHigh:           0,
		blockOnCritical:   strictMode,
		blockOnHigh:       strictMode,
		customRules:       make([]EnforcementRule, 0),
	}
}

// NewPolicyEnforcerWithConfig creates an enforcer with custom configuration
func NewPolicyEnforcerWithConfig(config EnforcerConfig) *PolicyEnforcer {
	return &PolicyEnforcer{
		strictMode:        config.StrictMode,
		allowedViolations: config.AllowedViolations,
		maxCritical:       config.MaxCritical,
		maxHigh:           config.MaxHigh,
		blockOnCritical:   config.BlockOnCritical,
		blockOnHigh:       config.BlockOnHigh,
		customRules:       config.CustomRules,
	}
}

// EnforcerConfig holds configuration for policy enforcement
type EnforcerConfig struct {
	StrictMode        bool
	AllowedViolations map[string]int
	MaxCritical       int
	MaxHigh           int
	BlockOnCritical   bool
	BlockOnHigh       bool
	CustomRules       []EnforcementRule
}

// Enforce checks violations against policy and returns error if policy is violated
func (e *PolicyEnforcer) Enforce(violations []SecurityViolation) error {
	if len(violations) == 0 {
		return nil
	}

	// Count violations by severity
	counts := e.countBySeverity(violations)
	critical := counts["Critical"]
	high := counts["High"]
	medium := counts["Medium"]
	low := counts["Low"]

	var errors []string

	// Check strict mode rules
	if e.strictMode {
		if critical > e.maxCritical {
			errors = append(errors, fmt.Sprintf("%d critical violations detected (max allowed: %d)", critical, e.maxCritical))
		}
		if high > e.maxHigh {
			errors = append(errors, fmt.Sprintf("%d high violations detected (max allowed: %d)", high, e.maxHigh))
		}
	}

	// Check blocking conditions
	if e.blockOnCritical && critical > 0 {
		errors = append(errors, fmt.Sprintf("blocking deployment: %d critical security violations", critical))
	}

	if e.blockOnHigh && high > 0 {
		errors = append(errors, fmt.Sprintf("blocking deployment: %d high security violations", high))
	}

	// Check custom rules
	for _, rule := range e.customRules {
		count := e.countByType(violations, rule.ViolationType)
		if count > rule.MaxAllowed {
			if rule.BlockOnExceed {
				errors = append(errors, fmt.Sprintf("%s: %d violations exceed limit of %d", rule.Message, count, rule.MaxAllowed))
			}
		}
	}

	// Check allowed violations
	for violationType, maxAllowed := range e.allowedViolations {
		count := e.countByType(violations, violationType)
		if count > maxAllowed {
			errors = append(errors, fmt.Sprintf("violation type %s: %d exceeds allowed %d", violationType, count, maxAllowed))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("security policy enforcement failed: %s (summary: %d critical, %d high, %d medium, %d low)",
			strings.Join(errors, "; "), critical, high, medium, low)
	}

	return nil
}

// countBySeverity counts violations by severity level
func (e *PolicyEnforcer) countBySeverity(violations []SecurityViolation) map[string]int {
	counts := map[string]int{
		"Critical": 0,
		"High":     0,
		"Medium":   0,
		"Low":      0,
	}

	for _, v := range violations {
		counts[v.Severity]++
	}

	return counts
}

// countByType counts violations of a specific type
func (e *PolicyEnforcer) countByType(violations []SecurityViolation, violationType string) int {
	count := 0
	for _, v := range violations {
		if v.Type == violationType {
			count++
		}
	}
	return count
}

// SetMaxViolations sets maximum allowed violations for a specific type
func (e *PolicyEnforcer) SetMaxViolations(violationType string, max int) {
	e.allowedViolations[violationType] = max
}

// AddCustomRule adds a custom enforcement rule
func (e *PolicyEnforcer) AddCustomRule(rule EnforcementRule) {
	e.customRules = append(e.customRules, rule)
}

// GetEnforcementReport generates a detailed enforcement report
func (e *PolicyEnforcer) GetEnforcementReport(violations []SecurityViolation) EnforcementReport {
	counts := e.countBySeverity(violations)

	passed := true
	reasons := make([]string, 0)

	if e.blockOnCritical && counts["Critical"] > 0 {
		passed = false
		reasons = append(reasons, fmt.Sprintf("Critical violations: %d", counts["Critical"]))
	}

	if e.blockOnHigh && counts["High"] > 0 {
		passed = false
		reasons = append(reasons, fmt.Sprintf("High violations: %d", counts["High"]))
	}

	return EnforcementReport{
		Passed:            passed,
		TotalViolations:   len(violations),
		CriticalCount:     counts["Critical"],
		HighCount:         counts["High"],
		MediumCount:       counts["Medium"],
		LowCount:          counts["Low"],
		BlockingReasons:   reasons,
		StrictModeEnabled: e.strictMode,
	}
}

// EnforcementReport contains the results of policy enforcement
type EnforcementReport struct {
	Passed            bool
	TotalViolations   int
	CriticalCount     int
	HighCount         int
	MediumCount       int
	LowCount          int
	BlockingReasons   []string
	StrictModeEnabled bool
}

// IsDeploymentAllowed returns whether deployment should be allowed based on violations
func (e *PolicyEnforcer) IsDeploymentAllowed(violations []SecurityViolation) bool {
	err := e.Enforce(violations)
	return err == nil
}

// GetRecommendedActions returns recommended actions based on violations
func (e *PolicyEnforcer) GetRecommendedActions(violations []SecurityViolation) []string {
	actions := make([]string, 0)

	typeCounts := make(map[string]int)
	for _, v := range violations {
		typeCounts[v.Type]++
	}

	// Prioritize critical issues
	if typeCounts["PrivilegedContainer"] > 0 {
		actions = append(actions, "Remove privileged mode from containers or use specific capabilities")
	}

	if typeCounts["RunAsRoot"] > 0 {
		actions = append(actions, "Configure containers to run as non-root user")
	}

	if typeCounts["HostNetworkAccess"] > 0 {
		actions = append(actions, "Disable host network access unless required for specific workloads")
	}

	if typeCounts["MissingNetworkPolicy"] > 0 {
		actions = append(actions, "Implement NetworkPolicies to control pod-to-pod communication")
	}

	if typeCounts["AllowPrivilegeEscalation"] > 0 {
		actions = append(actions, "Set allowPrivilegeEscalation: false in security contexts")
	}

	if typeCounts["MissingReadOnlyRootFS"] > 0 {
		actions = append(actions, "Enable read-only root filesystem for containers")
	}

	if typeCounts["HostPathVolume"] > 0 {
		actions = append(actions, "Replace hostPath volumes with PersistentVolumes")
	}

	if len(actions) == 0 {
		actions = append(actions, "No critical actions required")
	}

	return actions
}

// ValidateAgainstPodSecurityStandards checks violations against Pod Security Standards
func (e *PolicyEnforcer) ValidateAgainstPodSecurityStandards(violations []SecurityViolation, level string) (bool, []string) {
	// Pod Security Standards levels: Privileged, Baseline, Restricted
	failures := make([]string, 0)

	for _, v := range violations {
		switch level {
		case "Restricted":
			// Most restrictive - fails on almost all violations
			if v.Severity == "Critical" || v.Severity == "High" || v.Severity == "Medium" {
				failures = append(failures, fmt.Sprintf("%s: %s", v.Type, v.Message))
			}
		case "Baseline":
			// Moderate restrictions - fails on critical and high
			if v.Severity == "Critical" || v.Severity == "High" {
				failures = append(failures, fmt.Sprintf("%s: %s", v.Type, v.Message))
			}
		case "Privileged":
			// Least restrictive - only fails on critical
			if v.Severity == "Critical" {
				failures = append(failures, fmt.Sprintf("%s: %s", v.Type, v.Message))
			}
		}
	}

	return len(failures) == 0, failures
}