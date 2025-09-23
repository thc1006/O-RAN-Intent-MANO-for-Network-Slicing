# Network Security Policy Documentation

## Overview

This document describes the enhanced Kubernetes NetworkPolicy resources for the O-RAN Intent-based MANO system. These policies implement a zero-trust network security model with least-privilege access principles.

## Security Improvements

### 1. Enhanced Orchestrator Security (`oran-orchestrator-netpol`)

**Previous Issues Fixed:**
- Removed overly permissive ingress allowing all pods in namespace
- Eliminated unrestricted HTTPS egress (`to: []`)
- Added proper namespace isolation

**Current Security Features:**
- **Ingress**: HTTP API (8080) restricted to same namespace, Metrics (9090) only from Prometheus
- **Egress**: Strict DNS resolution, targeted service communication only
- **Isolation**: All external services require explicit namespace and pod selectors

### 2. Enhanced VNF Operator Security (`oran-vnf-operator-netpol`)

**Previous Issues Fixed:**
- Restricted webhook access to API server components only
- Removed overly broad Kubernetes API access
- Added specific health endpoint isolation

**Current Security Features:**
- **Ingress**: Health checks from same namespace, Metrics from Prometheus, Webhooks from API server
- **Egress**: Strict Kubernetes API access, DNS resolution, targeted service communication
- **Webhook Security**: Only admission controllers can access webhook endpoint

### 3. Enhanced DMS Security (RAN/CN)

**Previous Issues Fixed:**
- Removed wildcard selectors (`namespaceSelector: {}, podSelector: {}`)
- Added strict API server targeting
- Improved DNS resolution security

**Current Security Features:**
- **Ingress**: Service APIs restricted to authorized consumers
- **Egress**: Kubernetes API access limited to API server pods
- **Monitoring**: Metrics access restricted to Prometheus pods

### 4. Enhanced TN (Transport Network) Security

**Previous Issues Fixed:**
- Restricted network testing egress to specific namespaces
- Added proper isolation for iperf3 testing
- Removed wildcard egress rules

**Current Security Features:**
- **TN Manager**: Communication limited to TN Agents and orchestrator
- **TN Agent**: Network testing restricted to test environments
- **Testing**: iperf3 traffic controlled by namespace labels

### 5. Strengthened Default Deny Policy

**Previous Issues Fixed:**
- Restricted DNS access to kube-system namespace only
- Added CoreDNS targeting
- Enhanced documentation and labeling

**Current Security Features:**
- **Zero Trust**: All traffic denied by default
- **DNS**: Minimal required access to system DNS services
- **Baseline**: Provides security foundation for all pods

## Network Traffic Patterns

### Orchestrator Communication Flow
```
┌─────────────────┐    ┌─────────────┐    ┌─────────────┐
│   Orchestrator  │────│   RAN DMS   │────│   CN DMS    │
│    (8080)       │    │   (8080)    │    │   (8080)    │
└─────────────────┘    └─────────────┘    └─────────────┘
         │                      │                 │
         └──────────────────────┼─────────────────┘
                                │
                    ┌─────────────────────┐
                    │   Porch Server      │
                    │   (porch-system)    │
                    │   (7007)            │
                    └─────────────────────┘
```

### VNF Operator Communication Flow
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Kube APIServer │────│  VNF Operator   │────│   RAN DMS       │
│  (443/6443)     │    │  Webhook(9443)  │    │   (8080)        │
│  (kube-system)  │    │  Health(8081)   │    │   (oran-mano)   │
└─────────────────┘    │  Metrics(8080)  │    └─────────────────┘
                       └─────────────────┘
                                │
                    ┌─────────────────────┐
                    │   Porch Server      │
                    │   (porch-system)    │
                    │   (7007)            │
                    └─────────────────────┘
```

### Monitoring Integration
```
┌─────────────────┐    ┌─────────────────┐
│   Prometheus    │────│   All Services  │
│   (monitoring)  │    │   Metrics Ports │
│                 │    │   (8080/9090)   │
└─────────────────┘    └─────────────────┘
```

## Security Principles Applied

### 1. Least Privilege Access
- Each service receives only minimum required network permissions
- No wildcard selectors or overly broad access rules
- Specific port and protocol restrictions

### 2. Defense in Depth
- Multiple layers of network isolation
- Namespace-level and pod-level selectors
- Protocol and port-specific rules

### 3. Zero Trust Network Model
- Default deny-all baseline policy
- Explicit allow rules for all required communication
- No implicit trust between services

### 4. Secure Service Communication
- Mutual TLS ready (can be added via service mesh)
- Proper webhook security for admission controllers
- DNS resolution restricted to system components

## Compliance and Best Practices

### Kubernetes Security Standards
- ✅ Pod Security Standards compliance
- ✅ CIS Kubernetes Benchmark alignment
- ✅ NIST Cybersecurity Framework adherence

### Network Segmentation
- ✅ Microsegmentation between services
- ✅ Namespace isolation
- ✅ Role-based network access

### Monitoring and Observability
- ✅ Prometheus metrics collection secured
- ✅ Health check endpoints protected
- ✅ Audit trail through policy annotations

## Testing Network Policies

### Validation Commands
```bash
# Test policy enforcement
kubectl exec -n oran-mano <orchestrator-pod> -- nc -zv oran-ran-dms 8080

# Verify DNS resolution
kubectl exec -n oran-mano <any-pod> -- nslookup kubernetes.default

# Check metrics access
kubectl exec -n monitoring <prometheus-pod> -- curl http://oran-orchestrator.oran-mano:9090/metrics
```

### Expected Behaviors
- ✅ Authorized communication succeeds
- ❌ Unauthorized communication fails with connection timeout/refused
- ✅ DNS resolution works for all pods
- ✅ Metrics collection from monitoring namespace works

## Policy Maintenance

### Regular Security Reviews
1. Review traffic patterns quarterly
2. Update policies when adding new services
3. Monitor for policy violations in logs
4. Test policy changes in staging environment

### Emergency Procedures
1. Disable specific policies temporarily: `kubectl delete networkpolicy <policy-name> -n oran-mano`
2. Monitor pod logs for connection issues
3. Use `kubectl describe networkpolicy` for troubleshooting
4. Test changes incrementally

## Implementation Notes

### Namespace Labels Required
Ensure these namespace labels are present:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: oran-mano
  labels:
    name: oran-mano
---
apiVersion: v1
kind: Namespace
metadata:
  name: kube-system
  labels:
    name: kube-system
---
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
  labels:
    name: monitoring
---
apiVersion: v1
kind: Namespace
metadata:
  name: porch-system
  labels:
    name: porch-system
```

### Pod Labels Required
Ensure critical system pods have proper labels:
```yaml
# API Server pods
metadata:
  labels:
    component: kube-apiserver

# DNS pods
metadata:
  labels:
    k8s-app: kube-dns
    # OR
    k8s-app: coredns

# Prometheus pods
metadata:
  labels:
    app.kubernetes.io/name: prometheus
```

## Troubleshooting

### Common Issues
1. **DNS Resolution Fails**: Check kube-system namespace labels
2. **Metrics Not Accessible**: Verify monitoring namespace labels
3. **Webhook Timeouts**: Check API server pod labels
4. **Service Communication Fails**: Verify pod and namespace selectors

### Debugging Commands
```bash
# Check policy status
kubectl get networkpolicy -n oran-mano

# Describe specific policy
kubectl describe networkpolicy oran-orchestrator-netpol -n oran-mano

# Test connectivity
kubectl exec -n oran-mano <pod> -- nc -zv <target-service> <port>
```