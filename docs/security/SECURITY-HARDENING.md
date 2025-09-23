# O-RAN MANO Security Hardening Guide

## Overview

This document outlines the comprehensive security measures implemented in the O-RAN Intent-Based MANO system to meet Kubernetes security best practices and CIS benchmarks compliance.

## Security Framework

### 1. Pod Security Standards

All namespaces are configured with **restricted** Pod Security Standards:

```yaml
labels:
  pod-security.kubernetes.io/enforce: restricted
  pod-security.kubernetes.io/audit: restricted
  pod-security.kubernetes.io/warn: restricted
```

This ensures:
- Containers run as non-root users
- No privileged containers
- No privilege escalation
- Read-only root filesystems
- Dropped capabilities

### 2. Security Context Configuration

#### Container Security Context
All containers implement strict security contexts:

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
  procMount: Default
```

#### Pod Security Context
Pod-level security contexts provide additional hardening:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault
```

### 3. Network Security

#### Network Policies
Comprehensive NetworkPolicies implement defense-in-depth:

- **Default Deny**: All traffic denied by default
- **Granular Ingress**: Only required ports allowed
- **Restricted Egress**: Limited to necessary services
- **DNS Control**: Specific DNS server access only
- **Namespace Isolation**: Cross-namespace communication controlled

#### Key Network Controls
- DNS limited to kube-system namespace
- Kubernetes API access restricted
- Service-to-service communication explicit
- External egress only on port 443 (HTTPS)

### 4. RBAC and Service Accounts

#### Principle of Least Privilege
- Service account token mounting disabled by default
- Minimal ClusterRole permissions
- Resource-specific access controls
- Status and finalizer separation

#### RBAC Structure
```yaml
# Orchestrator permissions (read-only nodes, limited pod access)
# VNF Operator permissions (VNF lifecycle management only)
# TN Manager permissions (network management only)
```

### 5. Resource Governance

#### Resource Quotas
Namespace-level resource limits prevent resource exhaustion:

```yaml
requests.cpu: "2000m"
requests.memory: "4Gi"
limits.cpu: "8000m"
limits.memory: "16Gi"
persistentvolumeclaims: "10"
pods: "20"
```

#### Limit Ranges
Container and pod-level resource constraints:

```yaml
default:
  cpu: "100m"
  memory: "128Mi"
  ephemeral-storage: "1Gi"
max:
  cpu: "2000m"
  memory: "2Gi"
  ephemeral-storage: "10Gi"
```

### 6. Image Security

#### Image Policies
- Specific image versions with SHA digests
- ImagePullPolicy set to "Always"
- Distroless/minimal base images preferred
- Gatekeeper policies for image validation

#### Allowed Base Images
```yaml
allowedImages:
  - "ghcr.io/oran-mano/"
  - "gcr.io/distroless/"
  - "chainguard.dev/"
  - "registry.access.redhat.com/ubi8/ubi-minimal"
```

### 7. Admission Control

#### ValidatingAdmissionWebhook
Custom admission controller validates:
- Security context compliance
- Resource limit enforcement
- Image policy adherence
- Network policy requirements

#### Pod Security Policy (Legacy Support)
For clusters without Pod Security Standards:
- Restricted PSP for all components
- Non-root user enforcement
- Capability restrictions
- Volume type limitations

## Implementation Details

### Orchestrator Security Profile

**Image**: `ghcr.io/oran-mano/orchestrator:v1.0.1@sha256:...`

**Security Features**:
- Non-root user (65532)
- Read-only root filesystem
- Runtime/default seccomp profile
- AppArmor runtime/default profile
- Minimal RBAC permissions
- Resource limits enforced

**Network Restrictions**:
- Ingress: HTTP (8080), Metrics (9090)
- Egress: RAN DMS, CN DMS, Porch, O2IMS, DNS, HTTPS

### VNF Operator Security Profile

**Image**: `ghcr.io/oran-mano/vnf-operator:v1.0.1@sha256:...`

**Security Features**:
- Non-root user (65532)
- Read-only root filesystem
- Webhook TLS certificates
- Leader election for HA
- Custom resource validation

**Network Restrictions**:
- Ingress: Metrics (8080), Health (8081), Webhook (9443)
- Egress: Kubernetes API, RAN DMS, Porch, DNS

### Transport Network Security Profile

**Components**: TN Manager, TN Agent

**Security Features**:
- Network policy management capabilities
- Node-level access for traffic control
- Limited Kubernetes API permissions
- iperf3 testing (controlled ingress)

## Compliance and Benchmarks

### CIS Kubernetes Benchmark Alignment

| Control | Implementation | Status |
|---------|----------------|--------|
| 4.2.1 | Pod Security Policy/Standards | ✅ Implemented |
| 4.2.2 | Non-root containers | ✅ Implemented |
| 4.2.3 | Read-only root filesystem | ✅ Implemented |
| 4.2.4 | Privilege escalation disabled | ✅ Implemented |
| 4.2.5 | Seccomp profiles | ✅ Implemented |
| 4.2.6 | Capability restrictions | ✅ Implemented |
| 5.1.1 | RBAC enabled | ✅ Implemented |
| 5.1.3 | Service account tokens | ✅ Minimized |
| 5.2.1 | Network policies | ✅ Implemented |
| 5.3.1 | CNI plugin | ✅ Kube-OVN with policies |
| 5.7.1 | General policies | ✅ Implemented |

### NIST Cybersecurity Framework

- **Identify**: Asset inventory and risk assessment
- **Protect**: Access controls, data security
- **Detect**: Logging and monitoring
- **Respond**: Incident response procedures
- **Recover**: Backup and recovery plans

## Monitoring and Alerting

### Security Metrics
- Pod Security Standard violations
- Network policy denials
- RBAC permission failures
- Resource limit breaches
- Image policy violations

### Logging
- Audit logs for API access
- Network policy logs
- Security context violations
- Admission controller denials

## Deployment Checklist

### Pre-deployment
- [ ] Cluster has Pod Security Standards enabled
- [ ] Network policy provider (Calico/Cilium) installed
- [ ] RBAC enabled
- [ ] Admission controllers configured

### Deployment
- [ ] Apply namespace configuration with security labels
- [ ] Deploy RBAC resources
- [ ] Apply security policies
- [ ] Deploy network policies
- [ ] Deploy applications with security contexts

### Post-deployment
- [ ] Verify Pod Security Standard compliance
- [ ] Test network policy enforcement
- [ ] Validate RBAC permissions
- [ ] Monitor security metrics
- [ ] Run security scans

## Security Testing

### Network Policy Validation
```bash
# Test denied connections
kubectl exec -n oran-mano pod-name -- nc -zv unauthorized-service 80

# Test allowed connections
kubectl exec -n oran-mano pod-name -- nc -zv authorized-service 8080
```

### RBAC Testing
```bash
# Test service account permissions
kubectl auth can-i create pods --as=system:serviceaccount:oran-mano:oran-orchestrator

# Test unauthorized access
kubectl auth can-i delete secrets --as=system:serviceaccount:oran-mano:oran-orchestrator
```

### Security Context Validation
```bash
# Verify non-root execution
kubectl exec -n oran-mano pod-name -- id

# Verify read-only filesystem
kubectl exec -n oran-mano pod-name -- touch /test-file
```

## Incident Response

### Security Event Categories
1. **High**: Privilege escalation, unauthorized access
2. **Medium**: Policy violations, unusual network traffic
3. **Low**: Configuration drift, resource limits

### Response Procedures
1. Immediate containment
2. Investigation and analysis
3. Eradication and recovery
4. Post-incident review
5. Documentation and lessons learned

## Maintenance and Updates

### Regular Security Tasks
- Monthly security policy reviews
- Quarterly penetration testing
- Annual security architecture review
- Continuous vulnerability scanning
- Regular RBAC permission audits

### Update Procedures
1. Security patch management
2. Image vulnerability scanning
3. Configuration drift detection
4. Policy compliance monitoring
5. Security training and awareness

## Conclusion

This security hardening implementation provides defense-in-depth protection for the O-RAN MANO system while maintaining operational efficiency. The configuration aligns with industry best practices and compliance frameworks, ensuring a robust security posture for production deployments.

For questions or security concerns, contact the security team at security@oran-mano.org.