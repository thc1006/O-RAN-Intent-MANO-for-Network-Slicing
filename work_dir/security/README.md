# Security Validation System

Enterprise-grade security scanning for Kubernetes manifests and Nephio packages.

## Quick Start

```bash
# Install dependencies
cd work_dir/security
go mod tidy

# Run tests
go test -v .

# Build CLI tool
cd cmd
go build -o security-scanner main.go

# Scan a package
./security-scanner --scan /path/to/package --report security-report.md
```

## Features

✅ **Comprehensive Security Scanning**
- Pod security contexts
- Container privilege escalation
- Host namespace access
- Network policy validation
- Dangerous capabilities detection
- Volume mount security

✅ **Policy Enforcement**
- Strict mode with customizable thresholds
- Pod Security Standards compliance
- Custom enforcement rules
- Severity-based blocking

✅ **Detailed Reporting**
- Markdown reports with remediation steps
- Severity-based violation grouping
- Compliance status checking
- Actionable recommendations

✅ **CI/CD Integration**
- GitLab CI/CD
- GitHub Actions
- Jenkins pipelines
- Kubernetes admission webhooks

## Architecture

```
work_dir/security/
├── scanner.go           # Core security scanner
├── enforcer.go          # Policy enforcement engine
├── integration.go       # Package renderer integration
├── scanner_test.go      # Scanner tests
├── enforcer_test.go     # Enforcer tests
├── cmd/
│   └── main.go         # CLI tool
├── go.mod
└── README.md
```

## Usage

### Go API

```go
import "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/work_dir/security"

// Create scanner
scanner := security.NewSecurityScanner()

// Scan package
err := scanner.ScanPackage("path/to/package")

// Get violations
violations := scanner.GetViolations()

// Enforce policies
enforcer := security.NewPolicyEnforcer(true)
err = enforcer.Enforce(violations)
```

### CLI Tool

```bash
# Basic scan
security-scanner --scan ./my-package

# Custom report location
security-scanner --scan ./my-package --report ./reports/scan.md

# Non-strict mode
security-scanner --scan ./my-package --strict=false
```

## Detected Violations

### Critical
- Privileged containers
- Root user execution

### High
- Host network/PID/IPC access
- Missing network policies
- Privilege escalation allowed
- Dangerous capabilities
- HostPath volumes

### Medium
- Missing security contexts
- No read-only root filesystem
- RunAsNonRoot not enforced

### Low
- Capabilities not dropped
- Missing best practices

## Testing

```bash
# Run all tests
go test -v .

# Run specific test
go test -v -run TestSecurityScanner_PrivilegedContainer

# Test with coverage
go test -cover .
```

## Test Data

```
work_dir/testdata/
├── insecure-package/
│   ├── pod-privileged.yaml      # Intentionally insecure
│   └── deployment-root.yaml     # Runs as root
└── secure-package/
    ├── pod-secure.yaml          # Best practices
    └── networkpolicy.yaml       # Network isolation
```

## Documentation

- [Complete Documentation](../docs/SECURITY_SCANNER.md)
- [Usage Examples](../docs/SECURITY_EXAMPLES.md)
- [API Reference](../docs/SECURITY_SCANNER.md#api-reference)

## Integration Examples

### GitLab CI

```yaml
security-scan:
  stage: validate
  script:
    - cd work_dir/security/cmd
    - go run main.go --scan $PACKAGE_PATH
  artifacts:
    reports:
      security: work_dir/reports/security-scan.md
```

### GitHub Actions

```yaml
- name: Security Scan
  run: |
    cd work_dir/security/cmd
    go run main.go --scan ${{ github.workspace }}/packages
```

## Pod Security Standards

The scanner validates against three levels:

- **Privileged**: Most permissive (system workloads)
- **Baseline**: Minimal restrictions (default)
- **Restricted**: Hardening best practices

## Contributing

1. Add new violation types in `scanner.go`
2. Add detection logic
3. Write tests in `scanner_test.go`
4. Update documentation
5. Submit PR

## License

Part of the O-RAN Intent MANO project.

## Support

- GitHub Issues: [Report Issues](https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/issues)
- Documentation: [Full Docs](../docs/SECURITY_SCANNER.md)
- Examples: [Usage Examples](../docs/SECURITY_EXAMPLES.md)