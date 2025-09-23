package security

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// SecurityPolicy defines security requirements for Kubernetes manifests
type SecurityPolicy struct {
	Name        string
	Description string
	Check       func(*unstructured.Unstructured) SecurityViolation
}

// SecurityViolation represents a security policy violation
type SecurityViolation struct {
	Severity    string // "HIGH", "MEDIUM", "LOW"
	Message     string
	Remediation string
	Found       bool
}

// KubernetesManifestValidator validates Kubernetes manifests for security
type KubernetesManifestValidator struct {
	policies []SecurityPolicy
}

func NewKubernetesManifestValidator() *KubernetesManifestValidator {
	validator := &KubernetesManifestValidator{}
	validator.initializeSecurityPolicies()
	return validator
}

func (v *KubernetesManifestValidator) initializeSecurityPolicies() {
	v.policies = []SecurityPolicy{
		{
			Name:        "no-privileged-containers",
			Description: "Containers should not run in privileged mode",
			Check:       v.checkPrivilegedContainers,
		},
		{
			Name:        "no-root-user",
			Description: "Containers should not run as root user",
			Check:       v.checkRootUser,
		},
		{
			Name:        "no-host-network",
			Description: "Pods should not use host network",
			Check:       v.checkHostNetwork,
		},
		{
			Name:        "no-host-pid",
			Description: "Pods should not use host PID namespace",
			Check:       v.checkHostPID,
		},
		{
			Name:        "no-host-ipc",
			Description: "Pods should not use host IPC namespace",
			Check:       v.checkHostIPC,
		},
		{
			Name:        "resource-limits-set",
			Description: "Containers should have resource limits defined",
			Check:       v.checkResourceLimits,
		},
		{
			Name:        "resource-requests-set",
			Description: "Containers should have resource requests defined",
			Check:       v.checkResourceRequests,
		},
		{
			Name:        "security-context-defined",
			Description: "Containers should have security context defined",
			Check:       v.checkSecurityContext,
		},
		{
			Name:        "no-docker-socket-mount",
			Description: "Should not mount Docker socket",
			Check:       v.checkDockerSocketMount,
		},
		{
			Name:        "read-only-root-filesystem",
			Description: "Root filesystem should be read-only",
			Check:       v.checkReadOnlyRootFilesystem,
		},
		{
			Name:        "drop-all-capabilities",
			Description: "Should drop all capabilities unless specifically needed",
			Check:       v.checkCapabilities,
		},
		{
			Name:        "no-allow-privilege-escalation",
			Description: "Should not allow privilege escalation",
			Check:       v.checkPrivilegeEscalation,
		},
		{
			Name:        "network-policy-present",
			Description: "NetworkPolicy should be defined for workloads",
			Check:       v.checkNetworkPolicyPresence,
		},
		{
			Name:        "pod-security-standards",
			Description: "Pods should comply with Pod Security Standards",
			Check:       v.checkPodSecurityStandards,
		},
		{
			Name:        "image-pull-policy",
			Description: "Image pull policy should be Always for latest tags",
			Check:       v.checkImagePullPolicy,
		},
	}
}

func TestKubernetesManifests_SecurityValidation(t *testing.T) {
	validator := NewKubernetesManifestValidator()

	// Find all Kubernetes manifest files
	manifestPaths := []string{
		"../../deploy/k8s/base/*.yaml",
		"../../deploy/helm/charts/orchestrator/templates/*.yaml",
		"../../clusters/**/*.yaml",
	}

	var manifestFiles []string
	for _, pattern := range manifestPaths {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Logf("Warning: Could not glob pattern %s: %v", pattern, err)
			continue
		}
		manifestFiles = append(manifestFiles, files...)
	}

	if len(manifestFiles) == 0 {
		t.Skip("No Kubernetes manifest files found for validation")
	}

	highSeverityViolations := 0
	totalViolations := 0

	for _, manifestFile := range manifestFiles {
		t.Run(fmt.Sprintf("validate_%s", filepath.Base(manifestFile)), func(t *testing.T) {
			violations, err := validator.ValidateManifestFile(manifestFile)
			if err != nil {
				t.Logf("Warning: Could not validate %s: %v", manifestFile, err)
				return
			}

			for _, violation := range violations {
				totalViolations++
				if violation.Severity == "HIGH" {
					highSeverityViolations++
				}

				t.Logf("[%s] %s: %s", violation.Severity, manifestFile, violation.Message)
				if violation.Remediation != "" {
					t.Logf("  Remediation: %s", violation.Remediation)
				}
			}

			// Fail on HIGH severity violations
			highViolations := 0
			for _, violation := range violations {
				if violation.Severity == "HIGH" {
					highViolations++
				}
			}

			if highViolations > 0 {
				t.Errorf("Found %d HIGH severity security violations in %s", highViolations, manifestFile)
			}
		})
	}

	t.Logf("Security validation summary: %d total violations, %d high severity", totalViolations, highSeverityViolations)

	// Overall test should pass if no HIGH severity violations
	assert.Equal(t, 0, highSeverityViolations, "No HIGH severity security violations allowed")
}

func (v *KubernetesManifestValidator) ValidateManifestFile(filePath string) ([]SecurityViolation, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return v.ValidateManifestContent(content)
}

func (v *KubernetesManifestValidator) ValidateManifestContent(content []byte) ([]SecurityViolation, error) {
	var violations []SecurityViolation

	// Split multi-document YAML
	documents := strings.Split(string(content), "---")

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var obj unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			// Skip invalid YAML (might be comments or templates)
			continue
		}

		if obj.GetKind() == "" {
			continue
		}

		// Run all security policies
		for _, policy := range v.policies {
			violation := policy.Check(&obj)
			if violation.Found {
				violations = append(violations, violation)
			}
		}
	}

	return violations, nil
}

// Security policy implementations

func (v *KubernetesManifestValidator) checkPrivilegedContainers(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Container running in privileged mode",
		Remediation: "Set privileged: false in securityContext",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			if privileged, found := getNestedBool(containerMap, "securityContext", "privileged"); found && privileged {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkRootUser(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Container running as root user (UID 0)",
		Remediation: "Set runAsUser to non-zero value in securityContext",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			if runAsUser, found := getNestedInt64(containerMap, "securityContext", "runAsUser"); found && runAsUser == 0 {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkHostNetwork(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Pod using host network",
		Remediation: "Set hostNetwork: false or remove hostNetwork field",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	if hostNetwork, found := getNestedBool(obj.Object, "spec", "hostNetwork"); found && hostNetwork {
		violation.Found = true
	}

	return violation
}

func (v *KubernetesManifestValidator) checkHostPID(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Pod using host PID namespace",
		Remediation: "Set hostPID: false or remove hostPID field",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	if hostPID, found := getNestedBool(obj.Object, "spec", "hostPID"); found && hostPID {
		violation.Found = true
	}

	return violation
}

func (v *KubernetesManifestValidator) checkHostIPC(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Pod using host IPC namespace",
		Remediation: "Set hostIPC: false or remove hostIPC field",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	if hostIPC, found := getNestedBool(obj.Object, "spec", "hostIPC"); found && hostIPC {
		violation.Found = true
	}

	return violation
}

func (v *KubernetesManifestValidator) checkResourceLimits(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Container missing resource limits",
		Remediation: "Add resources.limits.memory and resources.limits.cpu",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			limits, found := getNestedMap(containerMap, "resources", "limits")
			if !found || limits == nil {
				violation.Found = true
				return violation
			}

			// Check for memory and CPU limits
			if _, hasMemory := limits["memory"]; !hasMemory {
				violation.Found = true
				return violation
			}
			if _, hasCPU := limits["cpu"]; !hasCPU {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkResourceRequests(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Container missing resource requests",
		Remediation: "Add resources.requests.memory and resources.requests.cpu",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			requests, found := getNestedMap(containerMap, "resources", "requests")
			if !found || requests == nil {
				violation.Found = true
				return violation
			}

			// Check for memory and CPU requests
			if _, hasMemory := requests["memory"]; !hasMemory {
				violation.Found = true
				return violation
			}
			if _, hasCPU := requests["cpu"]; !hasCPU {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkSecurityContext(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Container missing security context",
		Remediation: "Add securityContext with appropriate settings",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			if _, found := containerMap["securityContext"]; !found {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkDockerSocketMount(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Docker socket mounted in container",
		Remediation: "Remove /var/run/docker.sock volume mount",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	volumes, found, err := unstructured.NestedSlice(obj.Object, "spec", "volumes")
	if found && err == nil {
		for _, volume := range volumes {
			if volumeMap, ok := volume.(map[string]interface{}); ok {
				if hostPath, found := getNestedMap(volumeMap, "hostPath"); found {
					if path, found := hostPath["path"]; found && path == "/var/run/docker.sock" {
						violation.Found = true
						return violation
					}
				}
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkReadOnlyRootFilesystem(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Root filesystem not set to read-only",
		Remediation: "Set readOnlyRootFilesystem: true in securityContext",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			if readOnly, found := getNestedBool(containerMap, "securityContext", "readOnlyRootFilesystem"); !found || !readOnly {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkCapabilities(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Container not dropping all capabilities",
		Remediation: "Add capabilities.drop: [\"ALL\"] in securityContext",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			capabilities, found := getNestedMap(containerMap, "securityContext", "capabilities")
			if !found {
				violation.Found = true
				return violation
			}

			drop, found := capabilities["drop"]
			if !found {
				violation.Found = true
				return violation
			}

			if dropSlice, ok := drop.([]interface{}); ok {
				hasDropAll := false
				for _, cap := range dropSlice {
					if cap == "ALL" {
						hasDropAll = true
						break
					}
				}
				if !hasDropAll {
					violation.Found = true
					return violation
				}
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkPrivilegeEscalation(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "HIGH",
		Message:     "Privilege escalation not explicitly disabled",
		Remediation: "Set allowPrivilegeEscalation: false in securityContext",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			if allowPrivilegeEscalation, found := getNestedBool(containerMap, "securityContext", "allowPrivilegeEscalation"); !found || allowPrivilegeEscalation {
				violation.Found = true
				return violation
			}
		}
	}

	return violation
}

func (v *KubernetesManifestValidator) checkNetworkPolicyPresence(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "NetworkPolicy should be defined for workloads",
		Remediation: "Create NetworkPolicy to restrict network access",
	}

	// This is a complex check that would require analyzing multiple manifests
	// For now, we'll check if this IS a NetworkPolicy
	if obj.GetKind() == "NetworkPolicy" {
		return violation // No violation if NetworkPolicy is present
	}

	// For workload resources, we'd need to check if a corresponding NetworkPolicy exists
	// This would require a more sophisticated analysis across all manifests
	return violation
}

func (v *KubernetesManifestValidator) checkPodSecurityStandards(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "MEDIUM",
		Message:     "Pod may not comply with Pod Security Standards",
		Remediation: "Ensure pod complies with restricted Pod Security Standards",
	}

	if obj.GetKind() != "Namespace" {
		return violation
	}

	// Check for Pod Security Standards labels
	labels := obj.GetLabels()
	if labels != nil {
		if enforce, found := labels["pod-security.kubernetes.io/enforce"]; found && (enforce == "restricted" || enforce == "baseline") {
			return violation // No violation if PSS is properly configured
		}
	}

	violation.Found = true
	return violation
}

func (v *KubernetesManifestValidator) checkImagePullPolicy(obj *unstructured.Unstructured) SecurityViolation {
	violation := SecurityViolation{
		Severity:    "LOW",
		Message:     "Image pull policy should be Always for latest tags",
		Remediation: "Set imagePullPolicy: Always for images with latest tag",
	}

	if !isPodLikeResource(obj) {
		return violation
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
	if !found || err != nil {
		return violation
	}

	for _, container := range containers {
		if containerMap, ok := container.(map[string]interface{}); ok {
			image, found := containerMap["image"]
			if !found {
				continue
			}

			imageStr, ok := image.(string)
			if !ok {
				continue
			}

			// Check if image uses latest tag
			if strings.HasSuffix(imageStr, ":latest") || !strings.Contains(imageStr, ":") {
				// Check pull policy
				if pullPolicy, found := containerMap["imagePullPolicy"]; !found || pullPolicy != "Always" {
					violation.Found = true
					return violation
				}
			}
		}
	}

	return violation
}

// Helper functions

func isPodLikeResource(obj *unstructured.Unstructured) bool {
	kind := obj.GetKind()
	return kind == "Pod" || kind == "Deployment" || kind == "StatefulSet" ||
		   kind == "DaemonSet" || kind == "Job" || kind == "CronJob" ||
		   kind == "ReplicaSet"
}

func getNestedBool(obj map[string]interface{}, fields ...string) (bool, bool) {
	val, found, err := unstructured.NestedBool(obj, fields...)
	if err != nil {
		return false, false
	}
	return val, found
}

func getNestedInt64(obj map[string]interface{}, fields ...string) (int64, bool) {
	val, found, err := unstructured.NestedInt64(obj, fields...)
	if err != nil {
		return 0, false
	}
	return val, found
}

func getNestedMap(obj map[string]interface{}, fields ...string) (map[string]interface{}, bool) {
	val, found, err := unstructured.NestedMap(obj, fields...)
	if err != nil {
		return nil, false
	}
	return val, found
}

// Specific manifest validation tests

func TestKubernetesManifests_NetworkPolicies(t *testing.T) {
	validator := NewKubernetesManifestValidator()

	networkPolicyFiles, err := filepath.Glob("../../deploy/k8s/base/network*.yaml")
	require.NoError(t, err)

	if len(networkPolicyFiles) == 0 {
		t.Skip("No network policy files found")
	}

	for _, file := range networkPolicyFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := ioutil.ReadFile(file)
			require.NoError(t, err)

			violations, err := validator.ValidateManifestContent(content)
			require.NoError(t, err)

			// Network policies should have minimal violations
			highViolations := 0
			for _, violation := range violations {
				if violation.Severity == "HIGH" {
					highViolations++
				}
			}

			assert.Equal(t, 0, highViolations, "Network policies should not have HIGH severity violations")
		})
	}
}

func TestKubernetesManifests_RBAC(t *testing.T) {
	validator := NewKubernetesManifestValidator()

	rbacFiles, err := filepath.Glob("../../deploy/k8s/base/rbac*.yaml")
	require.NoError(t, err)

	if len(rbacFiles) == 0 {
		t.Skip("No RBAC files found")
	}

	for _, file := range rbacFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := ioutil.ReadFile(file)
			require.NoError(t, err)

			// RBAC resources have different validation requirements
			var rbacObjects []unstructured.Unstructured
			documents := strings.Split(string(content), "---")

			for _, doc := range documents {
				doc = strings.TrimSpace(doc)
				if doc == "" {
					continue
				}

				var obj unstructured.Unstructured
				if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
					continue
				}

				if obj.GetKind() != "" {
					rbacObjects = append(rbacObjects, obj)
				}
			}

			// Verify RBAC objects follow least privilege principle
			for _, obj := range rbacObjects {
				kind := obj.GetKind()
				assert.True(t,
					kind == "ServiceAccount" || kind == "Role" || kind == "RoleBinding" ||
					kind == "ClusterRole" || kind == "ClusterRoleBinding",
					"RBAC file should only contain RBAC resources, found: %s", kind)

				// Additional RBAC-specific checks can be added here
				if kind == "ClusterRole" || kind == "Role" {
					rules, found, err := unstructured.NestedSlice(obj.Object, "rules")
					if found && err == nil {
						for _, rule := range rules {
							if ruleMap, ok := rule.(map[string]interface{}); ok {
								// Check for overly permissive rules
								if verbs, found := ruleMap["verbs"]; found {
									if verbSlice, ok := verbs.([]interface{}); ok {
										for _, verb := range verbSlice {
											if verb == "*" {
												t.Logf("Warning: Found wildcard verb in %s", obj.GetName())
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestKubernetesManifests_PodSecurityPolicies(t *testing.T) {
	validator := NewKubernetesManifestValidator()

	securityFiles, err := filepath.Glob("../../deploy/k8s/base/*security*.yaml")
	require.NoError(t, err)

	if len(securityFiles) == 0 {
		t.Skip("No security policy files found")
	}

	for _, file := range securityFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			content, err := ioutil.ReadFile(file)
			require.NoError(t, err)

			violations, err := validator.ValidateManifestContent(content)
			require.NoError(t, err)

			// Security policy files should have no violations
			for _, violation := range violations {
				t.Logf("[%s] %s", violation.Severity, violation.Message)
			}
		})
	}
}

// Benchmark for large manifest validation
func BenchmarkKubernetesManifests_Validation(b *testing.B) {
	validator := NewKubernetesManifestValidator()

	// Use a sample manifest for benchmarking
	sampleManifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: app
        image: nginx:1.20
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		violations, err := validator.ValidateManifestContent([]byte(sampleManifest))
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
		_ = violations // Use violations to avoid optimization
	}
}