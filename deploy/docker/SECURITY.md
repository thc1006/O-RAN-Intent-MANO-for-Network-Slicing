# Container Security Guide

This document outlines the security measures implemented in the O-RAN Intent-MANO container images and provides guidance for secure deployment.

## Security Improvements Implemented

### 1. Base Image Security

- **Updated to Alpine 3.19**: Migrated from vulnerable Alpine 3.18 to latest Alpine 3.19
- **Debian 12-slim**: Used for test framework instead of Ubuntu 22.04
- **Distroless images**: VNF Operator uses Google's distroless base for minimal attack surface

### 2. Non-Root User Implementation

All containers run as non-root users with specific UIDs:

| Container | User ID | User Name | Purpose |
|-----------|---------|-----------|---------|
| CN DMS | 10001 | appuser | Core Network DMS |
| O2 Client | 10002 | appuser | O2 Interface Client |
| Orchestrator | 10003 | appuser | Core Orchestrator |
| RAN DMS | 10004 | appuser | RAN DMS |
| Test Framework | 10005 | tester | Testing Framework |
| TN Agent | 10006 | appuser | Transport Network Agent |
| TN Manager | 10007 | appuser | Transport Network Manager |
| VNF Operator | 65532 | nonroot | Kubernetes Operator (distroless) |

### 3. Security Labels and Metadata

All images include comprehensive security metadata:

```dockerfile
LABEL maintainer="O-RAN MANO Team"
LABEL org.opencontainers.image.title="Component Name"
LABEL org.opencontainers.image.description="Component Description"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="O-RAN Alliance"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL security.scan="trivy,grype,snyk"
LABEL security.user="non-root"
```

### 4. Runtime Security

- **dumb-init**: All containers use dumb-init as PID 1 for proper signal handling
- **Read-only root filesystem**: Implemented where possible
- **Dropped capabilities**: All unnecessary Linux capabilities removed
- **Security options**: no-new-privileges, AppArmor profiles

### 5. Network Security

- **Network isolation**: Services isolated in separate networks
- **Host binding**: Ports bound to localhost only (127.0.0.1)
- **Inter-container communication**: Disabled by default

### 6. Secrets Management

- **No hardcoded secrets**: All secrets externalized to mounted volumes
- **Secret paths**: `/config`, `/certs`, `/secrets`
- **Environment variables**: Used for non-sensitive configuration only

## Container-Specific Security Notes

### VNF Operator (Maximum Security)

- Uses Google's distroless static image
- Minimal attack surface with no shell or package manager
- Runs as distroless nonroot user (65532)
- Read-only filesystem
- No health check commands (uses binary-based health check)

### TN Agent/Manager (Network Capabilities)

- Requires `NET_ADMIN` and `NET_RAW` capabilities for traffic control
- Capabilities granted at runtime through Kubernetes security context
- Still runs as non-root user with elevated network privileges

### Test Framework (Development Only)

- Uses Debian 12-slim instead of Ubuntu for better security
- Includes necessary testing tools with pinned versions
- Should not be deployed in production environments

## Security Scanning

### Automated Scanning

Run the security scanner:

```bash
cd deploy/docker
./security-scan.sh
```

This script runs:
- **Trivy**: Vulnerability and misconfiguration scanning
- **Grype**: Vulnerability scanning
- **Snyk**: Security and license scanning

### Manual Scanning

Individual scans can be run manually:

```bash
# Trivy scan
trivy image --severity HIGH,CRITICAL oran-orchestrator:latest

# Grype scan
grype oran-orchestrator:latest --fail-on high

# Snyk scan (requires authentication)
snyk container test oran-orchestrator:latest --severity-threshold=high
```

### Scan Results

- Reports saved to `security-reports/` directory
- Summary report in `security-reports/security-summary.md`
- JSON reports for CI/CD integration

## Deployment Security

### Docker Compose Security Configuration

Use the security-hardened compose file:

```bash
docker-compose -f docker-compose.security.yml up -d
```

Key security features:
- Read-only containers with writable tmpfs
- Security options (no-new-privileges, AppArmor)
- Dropped capabilities
- User namespace isolation
- Network isolation
- Resource limits

### Kubernetes Deployment

For Kubernetes deployments, use these security contexts:

#### Standard Services (CN DMS, RAN DMS, Orchestrator, O2 Client)

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 10001  # Use appropriate UID
  runAsGroup: 10001
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

#### Network Services (TN Agent/Manager)

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 10006  # TN Agent UID
  runAsGroup: 10006
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
    add:
      - NET_ADMIN
      - NET_RAW
  seccompProfile:
    type: RuntimeDefault
```

#### VNF Operator (Distroless)

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault
```

## Security Monitoring

### Runtime Security

Implement runtime security monitoring:

1. **Falco**: Kubernetes runtime security monitoring
2. **OPA Gatekeeper**: Policy enforcement
3. **Pod Security Standards**: Kubernetes built-in policies

### Network Security

1. **Network policies**: Restrict inter-pod communication
2. **Service mesh**: mTLS for service-to-service communication
3. **Ingress security**: WAF and rate limiting

### Log Security

1. **Centralized logging**: Aggregate all container logs
2. **Log integrity**: Tamper-proof log storage
3. **Security events**: Monitor for security-relevant events

## Compliance

### Standards Compliance

- **CIS Benchmark**: Container images follow CIS Docker Benchmark
- **NIST 800-190**: Application Container Security Guide compliance
- **OWASP**: Top 10 container security risks mitigation

### Vulnerability Management

1. **Regular scanning**: Automated vulnerability scanning in CI/CD
2. **Patch management**: Regular base image updates
3. **Zero-day response**: Process for emergency updates

## Incident Response

### Security Incident Handling

1. **Detection**: Automated vulnerability and intrusion detection
2. **Containment**: Immediate container isolation capabilities
3. **Recovery**: Automated rollback and recovery procedures
4. **Analysis**: Forensic analysis and lessons learned

### Emergency Procedures

1. **Image recall**: Process for removing vulnerable images
2. **Service isolation**: Network-level service isolation
3. **Data protection**: Encrypted data and secure backups

## Security Checklist

Before deployment, verify:

- [ ] All containers run as non-root users
- [ ] No hardcoded secrets or credentials
- [ ] Security scanning passes (Trivy, Grype, Snyk)
- [ ] Read-only root filesystems where possible
- [ ] Minimal necessary capabilities only
- [ ] Network policies configured
- [ ] Logging and monitoring enabled
- [ ] Backup and recovery procedures tested
- [ ] Incident response plan in place
- [ ] Security documentation updated

## Contact

For security issues, contact:
- Security Team: security@o-ran.org
- Emergency: emergency-security@o-ran.org

## References

- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [NIST 800-190](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-190.pdf)
- [OWASP Container Security](https://owasp.org/www-project-kubernetes-top-ten/)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)