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

## Security Measures

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
