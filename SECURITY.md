# Security Policy

## Supported Versions

This project is currently under active development. Security updates will be applied to:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously in the O-RAN Intent-Based MANO project. If you believe you have found a security vulnerability, please report it to us as described below.

### How to Report

Please report security vulnerabilities by creating a private security advisory on GitHub:

1. Go to the Security tab of this repository
2. Click on "Report a vulnerability"
3. Fill out the security advisory form with:
   - A description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact
   - Any suggested fixes (if applicable)

### What to Expect

- **Initial Response**: We will acknowledge receipt of your vulnerability report within 48 hours
- **Assessment**: We will investigate and validate the reported vulnerability within 7 days
- **Resolution**: We aim to provide a fix or mitigation within 30 days, depending on complexity
- **Disclosure**: We will coordinate disclosure timing with you

## Security Best Practices

This project implements several security measures:

### Container Security
- All containers run as non-root users
- Health checks are configured for all services
- Minimal base images are used (Alpine/distroless where possible)
- Regular dependency updates and vulnerability scanning

### Network Security
- Network segmentation using O-RAN O2 interfaces
- TLS/mTLS for inter-service communication
- Network policies for pod-to-pod communication
- No hardcoded credentials or secrets

### Code Security
- Static code analysis with gosec and Trivy
- Secret scanning with TruffleHog
- Dependency vulnerability scanning
- Regular security audits

### Kubernetes Security
- RBAC policies for service accounts
- Pod Security Standards enforcement
- Resource quotas and limits
- Network policies

## Security Tools in CI/CD

Our CI/CD pipeline includes:
- **Trivy**: Container and dependency vulnerability scanning
- **gosec**: Go security checker
- **Checkov**: Infrastructure as Code security scanning
- **TruffleHog**: Secret detection
- **Hadolint**: Dockerfile linting

## Responsible Disclosure

We believe in responsible disclosure and will:
- Work with security researchers to understand and fix vulnerabilities
- Credit researchers who report valid vulnerabilities (unless they prefer to remain anonymous)
- Maintain transparency about security issues once fixed

## Contact

For urgent security issues that cannot be reported via GitHub, please contact the maintainers directly through the contact information in the repository.

## Security Updates

Security updates will be announced through:
- GitHub Security Advisories
- Release notes
- Project documentation

Thank you for helping keep the O-RAN Intent-Based MANO project secure!