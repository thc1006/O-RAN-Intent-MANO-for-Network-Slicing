# Security Compliance Test Suite

This directory contains comprehensive security testing tools for the O-RAN Intent-MANO Network Slicing project.

## Overview

The security test suite validates all security fixes and ensures compliance with security best practices through automated scanning and validation.

## Files

### Main Security Check Script
- **`security-check.sh`** - Comprehensive security validation script
- **`test-security-check.sh`** - Validation test script for the security checker

### GitHub Actions Integration
- **`.github/workflows/security-scan.yml`** - Automated security scanning workflow for CI/CD

## Security Check Script Features

The `security-check.sh` script performs the following validations:

### 1. Go Code Security Analysis (`gosec`)
- Scans all Go source files for security vulnerabilities
- Checks for error handling issues
- Validates input sanitization
- Detects hardcoded credentials
- Analyzes cryptographic usage

**Covered Security Issues:**
- G101: Hardcoded credentials
- G102: Bind to all interfaces
- G103: Audit the use of unsafe block
- G104: Audit errors not checked
- G106: Audit the use of ssh.InsecureIgnoreHostKey
- G201-G204: SQL injection vulnerabilities
- G301-G307: File system security
- G401-G404: Cryptographic security
- G501-G505: Import security
- G601: Implicit memory aliasing

### 2. Kubernetes Manifest Security (`checkov` + `kubesec`)
- Validates Kubernetes YAML files for security misconfigurations
- Checks for compliance with CIS Kubernetes Benchmark
- Analyzes Pod Security Standards
- Validates resource constraints and limits

**Checked Policies:**
- CKV_K8S_1: Process ID limits
- CKV_K8S_2: Privileged containers
- CKV_K8S_3: Docker socket access
- CKV_K8S_4: Privileged escalation
- CKV_K8S_5: SYS_ADMIN capability
- CKV_K8S_8: Root user restrictions
- CKV_K8S_9: Required probes
- CKV_K8S_10-K8S_49: Various security controls

### 3. NetworkPolicy Validation
- Ensures NetworkPolicy resources exist
- Validates policy syntax and structure
- Checks for overly permissive policies
- Verifies network segmentation implementation

**Requirements:**
- At least one NetworkPolicy per namespace
- Proper podSelector configuration
- Explicit policyTypes definition
- Ingress/egress rules defined

### 4. Container Security Context Validation
- Validates security contexts for all containers
- Ensures non-root user execution
- Checks for read-only root filesystem
- Verifies privilege escalation prevention

**Security Context Checks:**
- `runAsNonRoot: true`
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- Proper user ID configuration

### 5. Image Specification Validation
- Validates container image references
- Ensures digest-based or tagged images (no `:latest`)
- Checks for secure image sources
- Verifies image signing where applicable

**Image Security Levels:**
- **Most Secure**: Digest-based (`@sha256:...`)
- **Acceptable**: Specific tags (`:v1.2.3`)
- **Insecure**: Latest tags (`:latest`) or no tags

### 6. Service Account Configuration
- Validates ServiceAccount resources
- Checks for disabled token automounting
- Ensures proper RBAC associations
- Verifies custom ServiceAccount usage

**ServiceAccount Requirements:**
- `automountServiceAccountToken: false`
- Custom ServiceAccounts (not `default`)
- Proper RBAC bindings
- Minimal privilege principle

## Usage

### Local Development
```bash
# Run full security check
./scripts/security-check.sh

# Run without tool installation (if tools already present)
./scripts/security-check.sh --skip-install

# Run with verbose output
./scripts/security-check.sh --verbose

# Custom report location
./scripts/security-check.sh --report-file /path/to/report.json
```

### CI/CD Pipeline
The security check automatically runs in GitHub Actions on:
- Pull requests to `main` or `develop` branches
- Pushes to `main` or `develop` branches
- Daily scheduled scans at 2 AM UTC
- Manual workflow dispatch

### Validation Testing
```bash
# Test the security check functionality
./scripts/test-security-check.sh
```

## Prerequisites

### Required Tools
The script automatically installs these tools if missing:

1. **gosec** (v2.18.2+) - Go security analyzer
2. **checkov** (v3.0.0+) - Infrastructure security scanner
3. **kubesec** (v2.14.0+) - Kubernetes security analyzer
4. **jq** - JSON processor
5. **yq** - YAML processor

### System Requirements
- Go 1.23.4+
- Python 3.11+
- Bash 4.0+
- Internet access (for tool installation)

## Output and Reporting

### Report Format
The script generates a JSON report with:
```json
{
  "timestamp": "2025-09-24T01:27:51Z",
  "environment": "CI|LOCAL",
  "overall_status": "PASSED|WARNING|FAILED",
  "summary": {
    "total_issues": 0,
    "critical_issues": 0,
    "high_issues": 0,
    "medium_issues": 0,
    "low_issues": 0
  },
  "checks": {
    "go_security_scan": "PASSED|WARNING|FAILED",
    "kubernetes_manifest_scan": "PASSED|WARNING|FAILED",
    "network_policy_validation": "PASSED|WARNING|FAILED",
    "security_context_validation": "PASSED|WARNING|FAILED",
    "image_specification_validation": "PASSED|WARNING|FAILED",
    "service_account_validation": "PASSED|WARNING|FAILED"
  },
  "statistics": {
    "failed_checks": 0,
    "warning_checks": 0,
    "passed_checks": 6
  }
}
```

### Exit Codes
- **0**: All checks passed or warnings only
- **1**: One or more critical security issues found

### Log Files
- `security-check.log` - Detailed execution log
- `security-check-report.json` - Structured results

## GitHub Actions Integration

### Workflow Features
- **Multi-job execution**: Parallel security scans
- **PR comments**: Automatic security report comments
- **Issue creation**: Auto-creates issues for critical failures
- **SARIF upload**: Integration with GitHub Security tab
- **Artifact storage**: 30-day retention of reports

### Workflow Jobs
1. **security-scan**: Main comprehensive security check
2. **dependency-scan**: Go dependency vulnerability check
3. **container-scan**: Container image security analysis

### PR Integration
The workflow automatically:
- Comments on PRs with security results
- Updates existing comments on subsequent runs
- Provides detailed breakdown of issues
- Links to full reports and logs

## Security Thresholds

### Failure Conditions
The script fails (exit code 1) when:
- Critical or high-severity Go security issues found
- More than 20 Kubernetes manifest issues
- Missing NetworkPolicy resources
- More than 50% of containers have insecure contexts
- More than 33% of images use insecure specifications

### Warning Conditions
The script warns but passes when:
- Medium/low-severity security issues found
- Some containers have security context issues
- Some images use `:latest` tags
- ServiceAccount automount not disabled

## Customization

### Environment Variables
```bash
export GOSEC_VERSION="2.18.2"        # Override gosec version
export CHECKOV_VERSION="3.0.0"       # Override checkov version
export KUBESEC_VERSION="2.14.0"      # Override kubesec version
export SKIP_INSTALL=true             # Skip tool installation
```

### Configuration Files
- `.gosec.json` - Gosec configuration (if present)
- `.checkov.yaml` - Checkov configuration (if present)

## Troubleshooting

### Common Issues

1. **Tool Installation Failures**
   ```bash
   # Use skip-install and manually install tools
   ./scripts/security-check.sh --skip-install
   ```

2. **Permission Errors**
   ```bash
   # Ensure script is executable
   chmod +x scripts/security-check.sh
   ```

3. **Missing Dependencies**
   ```bash
   # Install system dependencies
   sudo apt-get install jq yq bc curl wget
   ```

4. **Windows Environment**
   - Use WSL2 or Git Bash
   - Ensure proper path formatting
   - Install Windows equivalents of tools

### Debug Mode
```bash
# Enable verbose debugging
./scripts/security-check.sh --verbose
```

## Best Practices

1. **Run security checks before committing**
2. **Address critical and high-severity issues immediately**
3. **Review warnings and plan fixes**
4. **Keep security tools updated**
5. **Monitor CI/CD pipeline results**
6. **Review security reports regularly**

## Contributing

When modifying security checks:
1. Update validation logic in `security-check.sh`
2. Add corresponding tests in `test-security-check.sh`
3. Update GitHub Actions workflow if needed
4. Test in both local and CI environments
5. Document changes in this README

## Support

For issues with the security test suite:
1. Check the troubleshooting section
2. Review log files for detailed errors
3. Run validation tests to isolate issues
4. Open an issue with full error details and environment info

---

**Note**: This security test suite is designed to catch common security issues but does not replace manual security reviews and penetration testing. Use it as part of a comprehensive security strategy.