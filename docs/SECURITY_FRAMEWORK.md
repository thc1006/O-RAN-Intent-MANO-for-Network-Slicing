# Security Framework Documentation

## Overview

The O-RAN Intent-MANO project implements a comprehensive security framework to mitigate common vulnerabilities while maintaining the functionality required for network slicing operations. This document outlines the security measures in place and explains how false positives in static analysis tools like gosec are handled.

## Centralized Security Package

### Location
All security-related functions are centralized in `pkg/security/subprocess.go`.

### Key Components

#### 1. Secure Subprocess Execution
- **Function**: `security.SecureExecute()`, `security.SecureExecuteWithValidation()`, `security.QuickSecureExecute()`
- **Purpose**: Provides safe subprocess execution with comprehensive validation
- **Security Features**:
  - Command allowlisting (only pre-approved commands can be executed)
  - Argument validation using regex patterns and allowlists
  - Timeout controls to prevent resource exhaustion
  - Output size limits to prevent memory exhaustion
  - Restricted environment variables
  - Context-aware execution with cancellation support

#### 2. Command Allowlisting
The security package maintains a registry of allowed commands with their permitted arguments:

- **iperf3**: Network performance testing with validated IP addresses, ports, and parameters
- **tc**: Traffic control operations with interface and bandwidth validation
- **ip**: Network interface management with VXLAN-specific controls
- **bridge**: Bridge FDB management with MAC address validation
- **ping**: Connectivity testing with restricted targets
- **git**: Version control operations with repository-safe commands only
- **kpt**: Kubernetes package management with path validation
- **cat**: File reading restricted to safe system paths only

#### 3. Input Validation Functions
- `ValidateIPAddress()`: Ensures IP addresses are properly formatted
- `ValidatePort()`: Validates port numbers are in acceptable ranges
- `ValidateBandwidth()`: Ensures bandwidth specifications are safe
- `ValidateNetworkInterface()`: Validates interface names
- `ValidateGitRef()`: Ensures Git references are safe
- `ValidateFilePath()`: Restricts file operations to safe paths

## GoSec Configuration

### False Positive Handling
The `.gosec.toml` configuration file addresses common false positives:

#### G204 (Subprocess Audit) Exclusions
- **Reason**: All subprocess calls use the centralized security package
- **Exclusions**:
  - `security.SecureExecute*` functions
  - `DefaultSecureExecutor.SecureExecute` calls
- **Justification**: These functions implement comprehensive security validation

#### G304 (File Inclusion) Exclusions
- **Reason**: File operations are restricted to safe, validated paths
- **Exclusions**:
  - Configuration files (`.yaml`, `.yml`, `.json`, `.toml`)
  - System monitoring paths (`/proc/net/dev`, `/sys/class/net/*`)
- **Justification**: Paths are validated and restricted to safe operations

#### G101 (Hardcoded Credentials) Exclusions
- **Reason**: Configuration keys are not actual secrets
- **Exclusions**:
  - `configKey`, `metadataKey`, `labelKey` patterns
- **Justification**: These are configuration identifiers, not sensitive data

## Security Measures by Component

### Transport Network (TN) Agent
- **File**: `tn/agent/pkg/tc/shaper.go`
- **Security**: Uses `security.SecureExecuteWithValidation()` with `ValidateTCArgs()`
- **Operations**: Traffic control commands for bandwidth shaping
- **Validation**: Interface names, bandwidth values, and TC command structure

### VXLAN Manager
- **File**: `tn/agent/pkg/vxlan/manager.go`
- **Security**: Uses `security.SecureExecute()` for VXLAN operations
- **Operations**: VXLAN tunnel creation and management
- **Validation**: VNI values, IP addresses, and interface parameters

### Monitoring Components
- **File**: `tn/agent/pkg/monitor.go`
- **Security**: Uses `security.SecureExecute()` for system monitoring
- **Operations**: Network statistics collection
- **Validation**: Safe system paths and monitoring commands

### Validation Framework
- **Files**: `clusters/validation-framework/*.go`
- **Security**: Git operations use `security.SecureExecuteWithValidation()`
- **Operations**: Repository management and validation
- **Validation**: Git references, branch names, and repository operations

## Best Practices Implemented

### 1. Principle of Least Privilege
- Only specific commands are allowed in the allowlist
- Arguments must match strict validation patterns
- File operations restricted to safe system paths

### 2. Defense in Depth
- Multiple validation layers (allowlist + pattern matching + custom validation)
- Timeout controls prevent resource exhaustion
- Output size limits prevent memory attacks
- Environment variable restrictions

### 3. Input Sanitization
- All external inputs validated before use
- Regex patterns prevent injection attacks
- Type-specific validation functions for different data types

### 4. Secure Defaults
- Restricted PATH environment
- Conservative timeout values
- Fail-safe error handling

## Static Analysis Tool Integration

### GoSec Integration
- Configuration file: `.gosec.toml`
- Excludes false positives while maintaining security coverage
- Custom rules for our security patterns
- SARIF output format for CI/CD integration

### CI/CD Pipeline Integration
- Automated security scanning in GitHub Actions
- Quality gates for security vulnerabilities
- SARIF file upload to GitHub Security tab
- Emergency override capability for critical deployments

## Compliance and Standards

### Security Standards Addressed
- **CWE-78**: OS Command Injection Prevention
- **CWE-22**: Path Traversal Prevention
- **CWE-20**: Input Validation
- **CWE-78**: Command Injection Prevention

### Regulatory Compliance
- Follows secure coding practices for network infrastructure
- Implements defense-in-depth strategies
- Provides audit trails for all system operations

## Usage Guidelines

### For Developers
1. **Always use security package functions** for subprocess execution
2. **Never bypass** the security validation functions
3. **Add new commands** to the allowlist through proper registration
4. **Validate all inputs** using provided validation functions

### For Security Auditors
1. **Review allowlist changes** carefully in code reviews
2. **Validate new validation functions** for completeness
3. **Monitor gosec findings** for new vulnerability patterns
4. **Ensure proper testing** of security functions

### Example Usage
```go
// Correct: Using secure execution
output, err := security.SecureExecuteWithValidation(
    ctx,
    "tc",
    security.ValidateTCArgs,
    "qdisc", "add", "dev", "eth0", "root", "htb"
)

// Incorrect: Direct execution (will be flagged by gosec)
output, err := exec.Command("tc", "qdisc", "add", "dev", "eth0", "root", "htb").Output()
```

## Monitoring and Maintenance

### Regular Security Reviews
- Quarterly review of allowlisted commands
- Annual security framework assessment
- Continuous monitoring of new vulnerability patterns

### Update Procedures
- Security package updates require security team review
- GoSec configuration changes need approval
- New command additions require threat assessment

## Contact and Support

For security-related questions or to report vulnerabilities:
- Open an issue in the project repository
- Tag with `security` label for priority handling
- Follow responsible disclosure practices

---

This security framework provides robust protection while enabling the network operations required for O-RAN Intent-MANO functionality. Regular reviews and updates ensure continued effectiveness against evolving threats.