# Security Validation System Documentation

## Overview

The Security Validation System provides comprehensive security scanning for Nephio-generated Kubernetes packages, enforcing Pod Security Standards and detecting common security misconfigurations.

## Features

### 1. Security Scanning
- **Pod Security**: Detects privileged containers, root users, host access
- **Network Policies**: Validates NetworkPolicy existence and configuration
- **Security Contexts**: Checks container security settings
- **Capabilities**: Identifies dangerous Linux capabilities
- **Volume Mounts**: Detects insecure hostPath usage

### 2. Policy Enforcement
- **Strict Mode**: Block deployments with critical/high violations
- **Custom Rules**: Define organization-specific security policies
- **Pod Security Standards**: Validate against Privileged/Baseline/Restricted levels
- **Severity-based Blocking**: Configure which severity levels block deployment

### 3. Reporting
- **Markdown Reports**: Detailed violation reports with remediation steps
- **Summary Metrics**: Violations grouped by severity and type
- **Compliance Status**: Pod Security Standards compliance checking
- **Actionable Recommendations**: Prioritized security improvements

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Validation System                │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │ SecurityScanner  │────────▶│ PolicyEnforcer   │          │
│  └────────┬─────────┘         └────────┬─────────┘          │
│           │                             │                     │
│           ▼                             ▼                     │
│  ┌─────────────────────────────────────────────┐            │
│  │          Violation Detection                 │            │
│  ├─────────────────────────────────────────────┤            │
│  │ • Pod Security                               │            │
│  │ • Container Security Context                 │            │
│  │ • Network Policies                           │            │
│  │ • Host Access                                │            │
│  │ • Capabilities                               │            │
│  └─────────────────────────────────────────────┘            │
│                           │                                   │
│                           ▼                                   │
│  ┌─────────────────────────────────────────────┐            │
│  │          Report Generation                   │            │
│  └─────────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

## Usage

### 1. Basic Scanning

```go
package main

import (
    "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"
)

func main() {
    // Create scanner
    scanner := security.NewSecurityScanner()

    // Scan package
    err := scanner.ScanPackage("path/to/nephio-package")
    if err != nil {
        panic(err)
    }

    // Get violations
    violations := scanner.GetViolations()

    // Generate report
    scanner.GenerateReport("security-report.md")
}
```

### 2. Policy Enforcement

```go
// Strict mode - blocks on critical and high violations
enforcer := security.NewPolicyEnforcer(true)

violations := scanner.GetViolations()
err := enforcer.Enforce(violations)
if err != nil {
    log.Fatalf("Security policy violation: %v", err)
}
```

### 3. Custom Enforcement Rules

```go
config := security.EnforcerConfig{
    StrictMode: true,
    MaxCritical: 0,
    MaxHigh: 2,  // Allow up to 2 high severity violations
    BlockOnCritical: true,
    BlockOnHigh: false,
    CustomRules: []security.EnforcementRule{
        {
            ViolationType: "HostPathVolume",
            MaxAllowed: 0,
            BlockOnExceed: true,
            Message: "No hostPath volumes allowed",
        },
    },
}

enforcer := security.NewPolicyEnforcerWithConfig(config)
```

### 4. Integration with Package Rendering

```go
config := security.SecurityIntegrationConfig{
    EnableScanning:   true,
    StrictMode:       true,
    GenerateReports:  true,
    ReportOutputPath: "reports/security-scan.md",
    FailOnViolations: true,
    PodSecurityLevel: "Baseline",
    BlockCritical:    true,
    BlockHigh:        true,
}

renderer := security.NewSecurePackageRenderer(config)
result, err := renderer.ValidatePackage("nephio-package")

if !result.Valid {
    security.PrintValidationResult(result)
    os.Exit(1)
}
```

### 5. CLI Tool

```bash
# Build the CLI tool
cd work_dir/security/cmd
go build -o security-scanner

# Scan a package
./security-scanner --scan /path/to/package --report security-report.md

# Non-strict mode
./security-scanner --scan /path/to/package --strict=false
```

## Violation Types and Severity

### Critical Severity
- **PrivilegedContainer**: Container runs with privileged: true
  - Remediation: Remove privileged flag or use specific capabilities

### High Severity
- **RunAsRoot**: Container runs as UID 0
  - Remediation: Set runAsUser to non-zero value
- **HostNetworkAccess**: Pod uses hostNetwork: true
  - Remediation: Remove host network access
- **HostPIDAccess**: Pod uses hostPID: true
  - Remediation: Remove host PID namespace access
- **HostIPCAccess**: Pod uses hostIPC: true
  - Remediation: Remove host IPC namespace access
- **AllowPrivilegeEscalation**: Container allows privilege escalation
  - Remediation: Set allowPrivilegeEscalation: false
- **HostPathVolume**: Pod mounts hostPath volumes
  - Remediation: Use PersistentVolumes instead
- **DangerousCapability**: Container adds dangerous Linux capabilities
  - Remediation: Remove or replace with safer alternatives
- **MissingNetworkPolicy**: No NetworkPolicy for namespace
  - Remediation: Create NetworkPolicy for pod-to-pod isolation

### Medium Severity
- **MissingSecurityContext**: Container lacks security context
  - Remediation: Add security context with restrictive settings
- **RunAsNonRootNotEnforced**: runAsNonRoot not set to true
  - Remediation: Set runAsNonRoot: true
- **MissingReadOnlyRootFS**: readOnlyRootFilesystem not enabled
  - Remediation: Enable read-only root filesystem
- **EmptyNetworkPolicy**: NetworkPolicy with no rules
  - Remediation: Define specific ingress/egress rules
- **BroadNetworkPolicySelector**: NetworkPolicy with empty selector
  - Remediation: Use specific pod selectors

### Low Severity
- **CapabilitiesNotDropped**: Container doesn't drop all capabilities
  - Remediation: Drop all capabilities, add back only required ones

## Pod Security Standards

The scanner validates against Kubernetes Pod Security Standards:

### Privileged
- Most permissive level
- Only blocks critical violations
- Use for system-level workloads only

### Baseline (Recommended)
- Minimally restrictive
- Blocks common privilege escalations
- Suitable for most applications

### Restricted
- Most restrictive
- Hardening best practices
- Blocks almost all violations

## Integration with CI/CD

### GitLab CI

```yaml
security-scan:
  stage: validate
  script:
    - go run work_dir/security/cmd/main.go --scan $NEPHIO_PACKAGE_PATH
    - |
      if [ $? -ne 0 ]; then
        echo "Security scan failed!"
        exit 1
      fi
  artifacts:
    reports:
      security: work_dir/reports/security-scan.md
```

### GitHub Actions

```yaml
- name: Security Scan
  run: |
    cd work_dir/security/cmd
    go run main.go --scan ${{ github.workspace }}/nephio-packages

- name: Upload Security Report
  uses: actions/upload-artifact@v3
  with:
    name: security-report
    path: work_dir/reports/security-scan.md
```

### Jenkins Pipeline

```groovy
stage('Security Scan') {
    steps {
        script {
            def result = sh(
                script: 'cd work_dir/security/cmd && go run main.go --scan nephio-packages',
                returnStatus: true
            )
            if (result != 0) {
                error("Security scan failed!")
            }
        }
        archiveArtifacts artifacts: 'work_dir/reports/security-scan.md'
    }
}
```

## Best Practices

### 1. Development Workflow
```bash
# 1. Generate package
nephio generate package my-network-function

# 2. Scan for security issues
security-scanner --scan my-network-function

# 3. Fix violations
# Edit manifests based on remediation guidance

# 4. Re-scan to verify
security-scanner --scan my-network-function

# 5. Deploy if clean
kubectl apply -f my-network-function/
```

### 2. Secure Manifest Template

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: secure-app
  namespace: production
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 2000
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: app
    image: myapp:latest
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      runAsNonRoot: true
      runAsUser: 1000
      capabilities:
        drop:
        - ALL
        add:
        - NET_BIND_SERVICE
    volumeMounts:
    - name: tmp
      mountPath: /tmp
  volumes:
  - name: tmp
    emptyDir: {}
```

### 3. NetworkPolicy Template

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: app-network-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: myapp
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          role: frontend
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          role: database
    ports:
    - protocol: TCP
      port: 5432
```

## API Reference

### SecurityScanner

```go
type SecurityScanner struct{}

func NewSecurityScanner() *SecurityScanner
func (s *SecurityScanner) ScanPackage(pkgPath string) error
func (s *SecurityScanner) GetViolations() []SecurityViolation
func (s *SecurityScanner) GenerateReport(outputPath string) error
func (s *SecurityScanner) GetSummary() map[string]interface{}
```

### PolicyEnforcer

```go
type PolicyEnforcer struct{}

func NewPolicyEnforcer(strictMode bool) *PolicyEnforcer
func NewPolicyEnforcerWithConfig(config EnforcerConfig) *PolicyEnforcer
func (e *PolicyEnforcer) Enforce(violations []SecurityViolation) error
func (e *PolicyEnforcer) IsDeploymentAllowed(violations []SecurityViolation) bool
func (e *PolicyEnforcer) GetRecommendedActions(violations []SecurityViolation) []string
func (e *PolicyEnforcer) ValidateAgainstPodSecurityStandards(violations []SecurityViolation, level string) (bool, []string)
```

### SecurePackageRenderer

```go
type SecurePackageRenderer struct{}

func NewSecurePackageRenderer(config SecurityIntegrationConfig) *SecurePackageRenderer
func (r *SecurePackageRenderer) ValidatePackage(pkgPath string) (*ValidationResult, error)
```

## Troubleshooting

### Scanner not detecting violations
- Verify YAML files are valid Kubernetes manifests
- Check file extensions (.yaml or .yml)
- Ensure manifests include apiVersion and kind fields

### False positives
- Use custom rules to allow specific violations
- Adjust severity thresholds in enforcer config
- Document exceptions in your security policy

### Integration issues
- Verify Go module paths are correct
- Check Kubernetes API version compatibility
- Ensure all dependencies are installed

## Contributing

To add new security checks:

1. Add violation type to scanner.go
2. Implement detection logic
3. Add test cases
4. Update documentation
5. Submit pull request

## Support

For issues or questions:
- GitHub Issues: https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/issues
- Documentation: work_dir/docs/
- Security Policy: SECURITY.md

## License

This security scanner is part of the O-RAN Intent MANO project.