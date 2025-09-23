# GoSec Configuration Guide

## Overview

This document explains the GoSec configuration (`.gosec.toml`) used in the O-RAN Intent-MANO project to handle false positives while maintaining security standards.

## Configuration File

The project uses `.gosec.toml` in the root directory to configure GoSec scanning with project-specific exclusions.

## Key Exclusions

### G204 (Subprocess Audit) Exclusions

**Why excluded**: All subprocess operations in this project use the centralized security package (`pkg/security`) that provides comprehensive validation.

**Functions excluded**:
- `security.SecureExecute*`
- `security.QuickSecureExecute`
- `security.SecureExecuteWithValidation`
- `DefaultSecureExecutor.SecureExecute`

**Justification**: These functions implement:
- Command allowlisting
- Argument validation with regex patterns
- Timeout controls
- Output size limits
- Environment variable restrictions

### G304 (File Inclusion) Exclusions

**Why excluded**: File operations are restricted to safe, validated paths.

**Patterns excluded**:
- Configuration files: `.yaml`, `.yml`, `.json`, `.toml`
- System monitoring paths: `/proc/net/dev`, `/sys/class/net/*`

**Justification**: All file paths are validated before use and restricted to safe system locations.

### G101 (Hardcoded Credentials) Exclusions

**Why excluded**: Configuration keys are not actual secrets.

**Patterns excluded**:
- `configKey`, `metadataKey`, `labelKey`

**Justification**: These are configuration identifiers, not sensitive credentials.

## Security Framework

### Command Execution Security
```go
// ✅ SECURE: Uses centralized security package
output, err := security.SecureExecuteWithValidation(
    ctx,
    "tc",
    security.ValidateTCArgs,
    "qdisc", "add", "dev", "eth0", "root", "htb"
)

// ❌ INSECURE: Direct execution (flagged by GoSec)
output, err := exec.Command("tc", "qdisc", "add", "dev", "eth0", "root", "htb").Output()
```

### Validation Functions
The security package provides specialized validation:
- `ValidateTCArgs()`: Traffic control command validation
- `ValidateIPerfArgs()`: Network testing command validation
- `ValidateIPArgs()`: IP command validation
- `ValidateGitArgs()`: Git operation validation

## Running GoSec Locally

### With Project Configuration
```bash
# Run GoSec with project configuration
gosec -conf=.gosec.toml ./...

# Generate SARIF output
gosec -conf=.gosec.toml -fmt=sarif -out=gosec.sarif ./...
```

### Without Configuration (shows all findings)
```bash
# Run without project exclusions to see all potential issues
gosec ./...
```

## CI/CD Integration

The GitHub Actions workflow automatically:
1. Uses the project `.gosec.toml` configuration
2. Generates SARIF output for GitHub Security tab
3. Applies quality gates based on severity levels
4. Provides detailed security information on failures

## Adding New Exclusions

### When to Add Exclusions
- New secure wrapper functions in `pkg/security`
- Additional safe file paths for configuration
- New validated command patterns

### How to Add Exclusions
1. Add to the appropriate section in `.gosec.toml`
2. Document the justification
3. Ensure the underlying security measure is robust
4. Test with `gosec -conf=.gosec.toml ./...`

### Example Addition
```toml
[issue.exclude-rules]
  # New secure function exclusion
  {
    text = "security\\.NewSecureFunction",
    rule = "G204"
  }
```

## Monitoring and Maintenance

### Regular Reviews
- Quarterly review of exclusion rules
- Validate that excluded functions maintain security
- Update patterns for new security measures

### Security Updates
- Monitor GoSec updates for new rules
- Review and update exclusions as needed
- Ensure security package evolves with threats

## Testing Security Measures

### Unit Tests
Run security package tests:
```bash
go test ./pkg/security/...
```

### Integration Tests
Test command validation:
```bash
# Test TC command validation
go test -run TestValidateTCArgs ./pkg/security/

# Test iperf validation
go test -run TestValidateIPerfArgs ./pkg/security/
```

### Manual Security Testing
```bash
# Test allowlist enforcement
go run -tags=security_test ./cmd/security-test/

# Verify command injection prevention
./scripts/security-test.sh
```

## Troubleshooting

### Common Issues

**Issue**: New subprocess calls flagged by GoSec
**Solution**: Use `security.SecureExecute*` functions instead of direct `exec.Command()`

**Issue**: Configuration file reads flagged
**Solution**: Ensure file paths match exclusion patterns or add new patterns

**Issue**: Git operations flagged
**Solution**: Use `security.SecureExecuteWithValidation()` with `ValidateGitArgs()`

### Getting Help

1. Check existing security patterns in `pkg/security/`
2. Review validation functions for your use case
3. Open an issue with `security` label for guidance
4. Consult the security team for new validation patterns

## Best Practices

1. **Always use security package functions** for subprocess execution
2. **Validate all inputs** before processing
3. **Review exclusions regularly** to ensure continued validity
4. **Test security measures** thoroughly
5. **Document new security patterns** for team awareness

---

This configuration balances security scanning effectiveness with practical development needs, ensuring real security issues are caught while reducing false positive noise.