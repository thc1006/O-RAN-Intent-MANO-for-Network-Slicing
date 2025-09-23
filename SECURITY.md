# Security Policy

## Supported Versions

Currently supported versions for security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability within this project, please follow these steps:

1. **DO NOT** create a public GitHub issue
2. Report the vulnerability by emailing the maintainers at [security@oran-mano.org](mailto:security@oran-mano.org)
3. Include detailed information about the vulnerability:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Resolution Target**: Critical: 7 days, High: 14 days, Medium: 30 days, Low: 90 days

## Command Injection Security Fixes

### Overview

This codebase has been secured against "subprocess launched with variable" vulnerabilities and command injection attacks through comprehensive input validation and secure coding practices.

### Security Measures Implemented

#### Input Validation Framework (`pkg/security/validation.go`)

A comprehensive validation framework that provides secure input validation for all external inputs:

**Network Security**
- Interface Name Validation: Validates network interface names against allowlist patterns
- IP Address Validation: Robust IPv4/IPv6 validation with security checks
- Port Validation: Port range validation with allowlist for privileged ports
- VNI Validation: VXLAN Network Identifier validation with reserved range checks
- Bandwidth Validation: Bandwidth string validation with reasonable limits

**File System Security**
- Path Validation: Prevents path traversal attacks (`../` sequences)
- File Existence Checks: Secure file and directory existence validation
- Allowed Directory Restrictions: Restricts absolute paths to safe directories

**Command Security**
- Argument Sanitization: Removes dangerous shell metacharacters
- Command Injection Prevention: Validates command arguments for suspicious patterns
- Environment Variable Validation: Prevents code injection through environment values

#### Secured Files

**Traffic Control (`tn/agent/pkg/tc.go`)**
- Validates network interface names before TC operations
- Validates bandwidth configuration parameters
- Prevents injection through interface names in TC commands

**VXLAN Management (`tn/agent/pkg/vxlan.go`)**
- Validates device names, VNI values, ports, and IP addresses
- Validates MTU ranges and remote peer IPs
- Secure FDB entry management with IP validation

**Network Performance Testing (`tn/agent/pkg/iperf.go`)**
- Validates server IPs and ports for iperf3 operations
- Validates test duration and parallel stream limits
- Validates bandwidth and window size parameters

**Network Monitoring (`tn/agent/pkg/monitor.go`)**
- Validates interface names for monitoring operations
- Secure interface discovery with validation
- Validates queue statistics collection

**Git Operations (`clusters/validation-framework/git_repository.go`)**
- Validates SSH key paths and Git references
- Validates branch names for Git operations
- Validates commit hashes for diff and reset operations

**Package Validation (`clusters/validation-framework/nephio_validator.go`)**
- Validates package paths and file paths
- Secure file reading with path validation
- Validates Kptfile structure and content

**Rollback Operations (`clusters/validation-framework/rollback_manager.go`)**
- Validates Git references for rollback operations
- Validates file paths for resource parsing
- Secure file operations with path validation

#### Security Features

1. **Defense in Depth**: Multiple layers of validation for each input type
2. **Allowlist-Based Validation**: Only approved patterns and values allowed
3. **Length and Range Limits**: Prevents buffer overflow and DoS attacks
4. **Pattern-Based Detection**: Identifies dangerous characters and injection attempts
5. **Sanitization and Escaping**: Safe handling of user input

#### Testing

Comprehensive test suite (`pkg/security/validation_test.go`) covering:
- All validation functions with positive and negative test cases
- Edge cases and boundary conditions
- Security attack vectors and injection attempts
- Performance benchmarks for validation functions

### Common Vulnerabilities Addressed

- **CWE-78**: OS Command Injection
- **CWE-22**: Path Traversal
- **CWE-20**: Improper Input Validation
- **CWE-88**: Argument Injection
- **CWE-94**: Code Injection
- **CWE-116**: Improper Encoding or Escaping of Output

## General Security Measures

### Container Security

- All containers run as non-root users
- Read-only root filesystems where applicable
- Security contexts with minimal capabilities
- Regular vulnerability scanning with OSV-Scanner and Trivy

### Kubernetes Security

- NetworkPolicies for pod-to-pod communication control
- RBAC with least privilege principles
- Secrets management using Kubernetes native secrets
- Pod security standards enforcement
- Seccomp profiles enabled

### Code Security

- Static code analysis with CodeQL
- Dependency scanning for known vulnerabilities
- Regular security audits
- Input validation and sanitization
- Secure communication using TLS/mTLS

### CI/CD Security

- Signed container images
- SBOM (Software Bill of Materials) generation
- Security scanning in CI/CD pipeline
- Protected branches and code review requirements

## Security Updates

Security updates will be released as patches to supported versions. Users will be notified through:
- GitHub Security Advisories
- Release notes
- Project mailing list

## Best Practices for Deployment

1. **Network Isolation**: Deploy in isolated network segments
2. **Access Control**: Implement strict RBAC policies
3. **Monitoring**: Enable audit logging and monitoring
4. **Updates**: Keep all components updated to latest stable versions
5. **Secrets**: Use proper secret management solutions
6. **TLS**: Enable TLS for all external communications

## Compliance

This project aims to comply with:
- CIS Kubernetes Benchmark
- NIST Cybersecurity Framework
- O-RAN Security Specifications
- Cloud Native Security best practices

## Contact

For security-related inquiries that don't need to be private:
- Open a discussion in the Security category
- Contact maintainers through GitHub

For sensitive security reports:
- Email: security@oran-mano.org
- GPG Key: [Available upon request]
