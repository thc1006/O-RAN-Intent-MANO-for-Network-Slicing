package security

import (
	"strings"
	"testing"
)

func TestPolicyEnforcer_StrictMode(t *testing.T) {
	tests := []struct {
		name        string
		strictMode  bool
		violations  []SecurityViolation
		expectError bool
	}{
		{
			name:       "strict mode with critical violations",
			strictMode: true,
			violations: []SecurityViolation{
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
			violations: []SecurityViolation{
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
			violations: []SecurityViolation{
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
			violations: []SecurityViolation{
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
			enforcer := NewPolicyEnforcer(tt.strictMode)

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
	violations := []SecurityViolation{
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "RunAsRoot", Severity: "High"},
		{Type: "MissingReadOnlyRootFS", Severity: "Medium"},
		{Type: "MissingSeccomp", Severity: "Low"},
	}

	enforcer := NewPolicyEnforcer(true)
	err := enforcer.Enforce(violations)

	if err == nil {
		t.Error("expected error with critical violations")
	}

	// Check error message contains counts
	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "2 critical") || !strings.Contains(errMsg, "1 high") {
			t.Errorf("expected error message to contain violation counts, got: %v", err)
		}
	}
}

func TestPolicyEnforcer_CustomRules(t *testing.T) {
	enforcer := NewPolicyEnforcer(false)

	// Add custom rule
	enforcer.AddCustomRule(EnforcementRule{
		ViolationType: "HostPathVolume",
		MaxAllowed:    1,
		BlockOnExceed: true,
		Message:       "Too many hostPath volumes",
	})

	violations := []SecurityViolation{
		{Type: "HostPathVolume", Severity: "High"},
		{Type: "HostPathVolume", Severity: "High"},
	}

	err := enforcer.Enforce(violations)
	if err == nil {
		t.Error("expected custom rule to trigger error")
	}
}

func TestPolicyEnforcer_IsDeploymentAllowed(t *testing.T) {
	enforcer := NewPolicyEnforcer(true)

	// Test with blocking violations
	criticalViolations := []SecurityViolation{
		{Type: "PrivilegedContainer", Severity: "Critical"},
	}

	if enforcer.IsDeploymentAllowed(criticalViolations) {
		t.Error("deployment should not be allowed with critical violations")
	}

	// Test with non-blocking violations
	lowViolations := []SecurityViolation{
		{Type: "MissingLabel", Severity: "Low"},
	}

	enforcer2 := NewPolicyEnforcer(false)
	if !enforcer2.IsDeploymentAllowed(lowViolations) {
		t.Error("deployment should be allowed with low violations in non-strict mode")
	}
}

func TestPolicyEnforcer_GetRecommendedActions(t *testing.T) {
	enforcer := NewPolicyEnforcer(true)

	violations := []SecurityViolation{
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "RunAsRoot", Severity: "High"},
		{Type: "MissingNetworkPolicy", Severity: "High"},
	}

	actions := enforcer.GetRecommendedActions(violations)

	if len(actions) == 0 {
		t.Error("expected recommended actions")
	}

	// Check that actions are relevant to violations
	actionsStr := strings.Join(actions, " ")
	if !strings.Contains(actionsStr, "privileged") && !strings.Contains(actionsStr, "root") {
		t.Error("expected actions to mention privileged or root issues")
	}
}

func TestPolicyEnforcer_PodSecurityStandards(t *testing.T) {
	enforcer := NewPolicyEnforcer(true)

	violations := []SecurityViolation{
		{Type: "PrivilegedContainer", Severity: "Critical"},
		{Type: "RunAsRoot", Severity: "High"},
		{Type: "MissingReadOnlyRootFS", Severity: "Medium"},
	}

	tests := []struct {
		level         string
		expectPass    bool
		minFailures   int
	}{
		{
			level:         "Privileged",
			expectPass:    false, // Even privileged fails on critical
			minFailures:   1,
		},
		{
			level:         "Baseline",
			expectPass:    false,
			minFailures:   2, // Critical + High
		},
		{
			level:         "Restricted",
			expectPass:    false,
			minFailures:   3, // All severity levels
		},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			passed, failures := enforcer.ValidateAgainstPodSecurityStandards(violations, tt.level)

			if passed != tt.expectPass {
				t.Errorf("expected pass=%v, got=%v", tt.expectPass, passed)
			}

			if len(failures) < tt.minFailures {
				t.Errorf("expected at least %d failures, got %d", tt.minFailures, len(failures))
			}
		})
	}
}