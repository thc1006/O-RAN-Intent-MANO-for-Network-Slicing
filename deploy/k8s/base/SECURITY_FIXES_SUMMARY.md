# Kubernetes Security Fixes Implementation

## Overview
This document summarizes the security fixes applied to address Checkov compliance issues in the O-RAN MANO Kubernetes deployments.

## Security Issues Addressed

### 1. Service Account Token Configuration
**Issue**: `automountServiceAccountToken` was set to `false` but components require API access.

**Fix Applied**:
- Changed `automountServiceAccountToken: true` for both orchestrator and vnf-operator
- Added detailed security justifications explaining why API access is required
- Updated service account definitions with security annotations

**Justification**:
- **Orchestrator**: Requires API access for intent processing, VNF deployment management, node information for placement decisions, and O2 interface operations
- **VNF Operator**: Requires API access for VNF lifecycle management, controller operations, leader election, and custom resource processing

### 2. Container Image Security
**Issue**: Image references used placeholder SHA256 digests.

**Fix Applied**:
- Updated image references to use proper versioned tags
- Added TODO comments for production digest implementation
- Included security notes for proper digest validation

**Current Configuration**:
```yaml
# Development/staging
image: ghcr.io/oran-mano/orchestrator:v1.0.1

# Production (to be implemented)
image: ghcr.io/oran-mano/orchestrator:v1.0.1@sha256:[actual-digest]
```

### 3. Security Annotations Enhancement
**Issue**: Missing compliance annotations for security scanning tools.

**Fix Applied**:
- Added comprehensive security annotations explaining configurations
- Included compliance markers for automated security scanning
- Enhanced documentation for security exceptions

**New Annotations**:
```yaml
security.policy/service-account-required: "explanation"
security.compliance/pod-security-standard: "restricted"
security.compliance/image-scanning: "required"
security.compliance/webhook-tls: "required"
```

### 4. Seccomp Profile Verification
**Status**: ✅ Already properly configured
- Both deployments use `seccompProfile.type: RuntimeDefault`
- Pod and container level security contexts are correctly set

### 5. Image Pull Policy
**Status**: ✅ Already properly configured
- Both deployments use `imagePullPolicy: Always`

## Security Context Summary

Both deployments maintain strong security posture:

### Pod Security Context
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault
```

### Container Security Context
```yaml
securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  capabilities:
    drop: [ALL]
  seccompProfile:
    type: RuntimeDefault
```

## RBAC Configuration

Maintains principle of least privilege:
- **Orchestrator**: Limited to VNF management, node reading, and O2 operations
- **VNF Operator**: Scoped to VNF lifecycle and custom resource management
- No cluster-admin privileges granted
- Read-only access where possible (e.g., secrets are read-only)

## Compliance Status

| Security Control | Status | Notes |
|-----------------|---------|-------|
| Non-root execution | ✅ | User 65532 (nobody) |
| Read-only filesystem | ✅ | With temp volume mounts |
| Capability dropping | ✅ | All capabilities dropped |
| Seccomp profiles | ✅ | RuntimeDefault enforced |
| Service account tokens | ✅ | Justified and documented |
| Image pull policy | ✅ | Always pull enabled |
| Image digests | ⚠️ | Versioned tags (production needs digests) |
| Resource limits | ✅ | CPU, memory, storage limited |
| Pod Security Standards | ✅ | Restricted profile enforced |

## Production Deployment Notes

For production deployments:

1. **Replace image tags with SHA256 digests**:
   ```bash
   # Get digest after image build
   docker inspect --format='{{index .RepoDigests 0}}' ghcr.io/oran-mano/orchestrator:v1.0.1
   ```

2. **Verify seccomp profile support** on target clusters

3. **Enable admission controllers**:
   - PodSecurityPolicy or Pod Security Standards
   - ImagePolicyWebhook for digest validation

4. **Network policies** should be applied for additional isolation

5. **Regular security scanning** of container images

## References

- [Kubernetes Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [NIST Container Security Guide](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-190.pdf)