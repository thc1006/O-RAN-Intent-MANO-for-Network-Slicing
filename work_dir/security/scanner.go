package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// SecurityViolation represents a security policy violation
type SecurityViolation struct {
	Resource    string
	Type        string
	Severity    string // Critical, High, Medium, Low
	Message     string
	Remediation string
}

// SecurityScanner scans Kubernetes manifests for security issues
type SecurityScanner struct {
	violations        []SecurityViolation
	hasNetworkPolicy  bool
	scannedNamespaces map[string]bool
	podCount          int
}

// NewSecurityScanner creates a new security scanner instance
func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		violations:        make([]SecurityViolation, 0),
		scannedNamespaces: make(map[string]bool),
		hasNetworkPolicy:  false,
		podCount:          0,
	}
}

// ScanPackage scans all YAML files in a Nephio package for security violations
func (s *SecurityScanner) ScanPackage(pkgPath string) error {
	// Reset state for new scan
	s.violations = make([]SecurityViolation, 0)
	s.hasNetworkPolicy = false
	s.scannedNamespaces = make(map[string]bool)
	s.podCount = 0

	// Check if path exists
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		return fmt.Errorf("package path does not exist: %s", pkgPath)
	}

	// Walk through all files in the package
	err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml") {
			if err := s.scanFile(path); err != nil {
				// Log error but continue scanning
				fmt.Printf("Warning: error scanning %s: %v\n", path, err)
			}
		}
		return nil
	})

	// After scanning all files, check for missing network policies
	if s.podCount > 0 && !s.hasNetworkPolicy {
		for namespace := range s.scannedNamespaces {
			s.violations = append(s.violations, SecurityViolation{
				Resource:    pkgPath,
				Type:        "MissingNetworkPolicy",
				Severity:    "High",
				Message:     fmt.Sprintf("No NetworkPolicy found for namespace: %s", namespace),
				Remediation: "Create NetworkPolicy to restrict pod-to-pod communication",
			})
		}
	}

	return err
}

// scanFile scans a single YAML file for security violations
func (s *SecurityScanner) scanFile(path string) error {
	// Validate and sanitize path to prevent path traversal
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Ensure path doesn't contain traversal attempts
	if strings.Contains(filepath.ToSlash(cleanPath), "..") {
		return fmt.Errorf("path traversal detected in file path")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return err
	}

	// Split on YAML document separator to handle multi-document files
	docs := strings.Split(string(data), "\n---\n")

	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		// Parse as unstructured to determine resource type
		obj := &unstructured.Unstructured{}
		decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 4096)
		if err := decoder.Decode(obj); err != nil {
			continue // Skip malformed documents
		}

		kind := obj.GetKind()
		namespace := obj.GetNamespace()
		if namespace == "" {
			namespace = "default"
		}
		s.scannedNamespaces[namespace] = true

		switch kind {
		case "Pod":
			s.scanPod(path, []byte(doc))
		case "Deployment", "StatefulSet", "DaemonSet":
			s.scanWorkload(path, []byte(doc), kind)
		case "NetworkPolicy":
			s.hasNetworkPolicy = true
			s.scanNetworkPolicy(path, []byte(doc))
		}
	}

	return nil
}

// scanPod scans a Pod resource for security violations
func (s *SecurityScanner) scanPod(path string, data []byte) {
	var pod corev1.Pod
	if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096).Decode(&pod); err != nil {
		return
	}

	s.podCount++

	// Check pod-level security settings
	if pod.Spec.HostNetwork {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "HostNetworkAccess",
			Severity:    "High",
			Message:     fmt.Sprintf("Pod %s uses host network", pod.Name),
			Remediation: "Remove hostNetwork: true unless absolutely necessary",
		})
	}

	if pod.Spec.HostPID {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "HostPIDAccess",
			Severity:    "High",
			Message:     fmt.Sprintf("Pod %s uses host PID namespace", pod.Name),
			Remediation: "Remove hostPID: true unless absolutely necessary",
		})
	}

	if pod.Spec.HostIPC {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "HostIPCAccess",
			Severity:    "High",
			Message:     fmt.Sprintf("Pod %s uses host IPC namespace", pod.Name),
			Remediation: "Remove hostIPC: true unless absolutely necessary",
		})
	}

	// Check container security contexts
	for _, container := range pod.Spec.Containers {
		s.scanContainer(path, pod.Name, container)
	}

	for _, container := range pod.Spec.InitContainers {
		s.scanContainer(path, pod.Name, container)
	}

	// Check for host path volumes
	for _, volume := range pod.Spec.Volumes {
		if volume.HostPath != nil {
			s.violations = append(s.violations, SecurityViolation{
				Resource:    path,
				Type:        "HostPathVolume",
				Severity:    "High",
				Message:     fmt.Sprintf("Pod %s mounts host path: %s", pod.Name, volume.HostPath.Path),
				Remediation: "Avoid using hostPath volumes; use PersistentVolumes instead",
			})
		}
	}
}

// scanContainer scans a container for security violations
func (s *SecurityScanner) scanContainer(path, podName string, container corev1.Container) {
	if container.SecurityContext == nil {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "MissingSecurityContext",
			Severity:    "Medium",
			Message:     fmt.Sprintf("Container %s in pod %s has no security context", container.Name, podName),
			Remediation: "Add securityContext with restrictive settings",
		})
		return
	}

	sc := container.SecurityContext

	// Check for privileged containers
	if sc.Privileged != nil && *sc.Privileged {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "PrivilegedContainer",
			Severity:    "Critical",
			Message:     fmt.Sprintf("Container %s in pod %s runs in privileged mode", container.Name, podName),
			Remediation: "Remove privileged: true or use specific capabilities instead",
		})
	}

	// Check for root user
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "RunAsRoot",
			Severity:    "High",
			Message:     fmt.Sprintf("Container %s in pod %s runs as root (UID 0)", container.Name, podName),
			Remediation: "Set runAsUser to non-zero value (e.g., 1000)",
		})
	}

	// Check for runAsNonRoot not enforced
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "RunAsNonRootNotEnforced",
			Severity:    "Medium",
			Message:     fmt.Sprintf("Container %s in pod %s does not enforce runAsNonRoot", container.Name, podName),
			Remediation: "Set runAsNonRoot: true to prevent running as root",
		})
	}

	// Check for privilege escalation
	if sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "AllowPrivilegeEscalation",
			Severity:    "High",
			Message:     fmt.Sprintf("Container %s in pod %s allows privilege escalation", container.Name, podName),
			Remediation: "Set allowPrivilegeEscalation: false",
		})
	}

	// Check for read-only root filesystem
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "MissingReadOnlyRootFS",
			Severity:    "Medium",
			Message:     fmt.Sprintf("Container %s in pod %s does not use read-only root filesystem", container.Name, podName),
			Remediation: "Set readOnlyRootFilesystem: true and use volumes for writable directories",
		})
	}

	// Check for dangerous capabilities
	if sc.Capabilities != nil {
		for _, cap := range sc.Capabilities.Add {
			dangerousCaps := []string{"SYS_ADMIN", "NET_ADMIN", "SYS_MODULE", "SYS_RAWIO"}
			for _, dangerous := range dangerousCaps {
				if string(cap) == dangerous {
					s.violations = append(s.violations, SecurityViolation{
						Resource:    path,
						Type:        "DangerousCapability",
						Severity:    "High",
						Message:     fmt.Sprintf("Container %s in pod %s adds dangerous capability: %s", container.Name, podName, cap),
						Remediation: fmt.Sprintf("Remove capability %s or use more specific alternatives", cap),
					})
				}
			}
		}

		// Check if all capabilities are dropped
		allDropped := false
		for _, cap := range sc.Capabilities.Drop {
			if string(cap) == "ALL" {
				allDropped = true
				break
			}
		}

		if !allDropped {
			s.violations = append(s.violations, SecurityViolation{
				Resource:    path,
				Type:        "CapabilitiesNotDropped",
				Severity:    "Low",
				Message:     fmt.Sprintf("Container %s in pod %s does not drop all capabilities", container.Name, podName),
				Remediation: "Drop all capabilities with 'drop: [ALL]' and add back only required ones",
			})
		}
	}
}

// scanWorkload scans Deployment, StatefulSet, or DaemonSet for security violations
func (s *SecurityScanner) scanWorkload(path string, data []byte, kind string) {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096)

	switch kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := decoder.Decode(&deployment); err != nil {
			return
		}
		s.scanPodTemplate(path, deployment.Name, &deployment.Spec.Template)
	case "StatefulSet":
		var statefulset appsv1.StatefulSet
		if err := decoder.Decode(&statefulset); err != nil {
			return
		}
		s.scanPodTemplate(path, statefulset.Name, &statefulset.Spec.Template)
	case "DaemonSet":
		var daemonset appsv1.DaemonSet
		if err := decoder.Decode(&daemonset); err != nil {
			return
		}
		s.scanPodTemplate(path, daemonset.Name, &daemonset.Spec.Template)
	}
}

// scanPodTemplate scans a pod template for security violations
func (s *SecurityScanner) scanPodTemplate(path, workloadName string, template *corev1.PodTemplateSpec) {
	s.podCount++

	// Check pod-level security
	if template.Spec.HostNetwork {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "HostNetworkAccess",
			Severity:    "High",
			Message:     fmt.Sprintf("Workload %s uses host network", workloadName),
			Remediation: "Remove hostNetwork: true unless absolutely necessary",
		})
	}

	// Check containers
	for _, container := range template.Spec.Containers {
		s.scanContainer(path, workloadName, container)
	}

	for _, container := range template.Spec.InitContainers {
		s.scanContainer(path, workloadName, container)
	}

	// Check volumes
	for _, volume := range template.Spec.Volumes {
		if volume.HostPath != nil {
			s.violations = append(s.violations, SecurityViolation{
				Resource:    path,
				Type:        "HostPathVolume",
				Severity:    "High",
				Message:     fmt.Sprintf("Workload %s mounts host path: %s", workloadName, volume.HostPath.Path),
				Remediation: "Avoid using hostPath volumes; use PersistentVolumes instead",
			})
		}
	}
}

// scanNetworkPolicy validates NetworkPolicy configuration
func (s *SecurityScanner) scanNetworkPolicy(path string, data []byte) {
	var netpol netv1.NetworkPolicy
	if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(data)), 4096).Decode(&netpol); err != nil {
		return
	}

	// Check if policy is too permissive
	if len(netpol.Spec.Ingress) == 0 && len(netpol.Spec.Egress) == 0 {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "EmptyNetworkPolicy",
			Severity:    "Medium",
			Message:     fmt.Sprintf("NetworkPolicy %s has no ingress or egress rules", netpol.Name),
			Remediation: "Define specific ingress and egress rules",
		})
	}

	// Check for overly broad selectors
	if len(netpol.Spec.PodSelector.MatchLabels) == 0 && len(netpol.Spec.PodSelector.MatchExpressions) == 0 {
		s.violations = append(s.violations, SecurityViolation{
			Resource:    path,
			Type:        "BroadNetworkPolicySelector",
			Severity:    "Medium",
			Message:     fmt.Sprintf("NetworkPolicy %s uses empty pod selector (applies to all pods)", netpol.Name),
			Remediation: "Use specific pod selectors to limit policy scope",
		})
	}
}

// GetViolations returns all detected security violations
func (s *SecurityScanner) GetViolations() []SecurityViolation {
	return s.violations
}

// GenerateReport creates a markdown report of security violations
func (s *SecurityScanner) GenerateReport(outputPath string) error {
	// Validate and sanitize outputPath to prevent path traversal
	cleanPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Ensure path doesn't contain traversal attempts
	if strings.Contains(filepath.ToSlash(cleanPath), "..") {
		return fmt.Errorf("path traversal detected in output path")
	}

	file, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Write report header
	fmt.Fprintf(file, "# Security Scan Report\n\n")
	fmt.Fprintf(file, "**Total Violations:** %d\n\n", len(s.violations))

	// Count by severity
	severityCounts := make(map[string]int)
	for _, v := range s.violations {
		severityCounts[v.Severity]++
	}

	fmt.Fprintf(file, "## Violations by Severity\n\n")
	fmt.Fprintf(file, "- **Critical:** %d\n", severityCounts["Critical"])
	fmt.Fprintf(file, "- **High:** %d\n", severityCounts["High"])
	fmt.Fprintf(file, "- **Medium:** %d\n", severityCounts["Medium"])
	fmt.Fprintf(file, "- **Low:** %d\n\n", severityCounts["Low"])

	// Group violations by type
	violationsByType := make(map[string][]SecurityViolation)
	for _, v := range s.violations {
		violationsByType[v.Type] = append(violationsByType[v.Type], v)
	}

	fmt.Fprintf(file, "## Detailed Findings\n\n")

	// Sort severities for consistent output
	severityOrder := []string{"Critical", "High", "Medium", "Low"}
	for _, severity := range severityOrder {
		violations := s.getViolationsBySeverity(severity)
		if len(violations) == 0 {
			continue
		}

		fmt.Fprintf(file, "### %s Severity\n\n", severity)
		for _, v := range violations {
			fmt.Fprintf(file, "#### %s\n\n", v.Type)
			fmt.Fprintf(file, "**Resource:** `%s`\n\n", v.Resource)
			fmt.Fprintf(file, "**Message:** %s\n\n", v.Message)
			fmt.Fprintf(file, "**Remediation:** %s\n\n", v.Remediation)
			fmt.Fprintf(file, "---\n\n")
		}
	}

	return nil
}

// getViolationsBySeverity returns violations filtered by severity
func (s *SecurityScanner) getViolationsBySeverity(severity string) []SecurityViolation {
	var filtered []SecurityViolation
	for _, v := range s.violations {
		if v.Severity == severity {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// GetSummary returns a summary of the scan results
func (s *SecurityScanner) GetSummary() map[string]interface{} {
	severityCounts := make(map[string]int)
	typeCounts := make(map[string]int)

	for _, v := range s.violations {
		severityCounts[v.Severity]++
		typeCounts[v.Type]++
	}

	return map[string]interface{}{
		"total_violations": len(s.violations),
		"by_severity":      severityCounts,
		"by_type":          typeCounts,
		"has_critical":     severityCounts["Critical"] > 0,
		"has_high":         severityCounts["High"] > 0,
	}
}