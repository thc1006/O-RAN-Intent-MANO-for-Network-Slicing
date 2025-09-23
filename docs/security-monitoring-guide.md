# O-RAN Intent MANO Security and Monitoring Guide

This guide provides comprehensive security hardening and production monitoring for the O-RAN Intent-MANO system, specifically designed to support the thesis performance targets and provide enterprise-grade observability.

## Table of Contents

1. [Overview](#overview)
2. [Security Components](#security-components)
3. [Monitoring Stack](#monitoring-stack)
4. [Deployment](#deployment)
5. [Configuration](#configuration)
6. [Thesis Performance Monitoring](#thesis-performance-monitoring)
7. [Security Policies](#security-policies)
8. [Troubleshooting](#troubleshooting)

## Overview

The security and monitoring stack provides:

- **Production-grade security** with defense in depth
- **Comprehensive observability** for all O-RAN components
- **Automated thesis performance monitoring** with SLA tracking
- **Real-time security monitoring** and threat detection
- **Supply chain security** with SLSA compliance
- **Distributed tracing** for E2E workflow visibility

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Security Layer                          │
├─────────────────────────────────────────────────────────────┤
│ OPA Gatekeeper │ Pod Security │ Network Policies │ RBAC   │
│ Trivy Scanning │ Falco Runtime│ Sealed Secrets   │ SLSA   │
├─────────────────────────────────────────────────────────────┤
│                  Observability Layer                       │
├─────────────────────────────────────────────────────────────┤
│ Prometheus │ Grafana │ Alertmanager │ Jaeger │ Loki │ SLA │
│ OpenTelemetry │ Distributed Tracing │ Log Aggregation     │
├─────────────────────────────────────────────────────────────┤
│                    O-RAN Components                        │
├─────────────────────────────────────────────────────────────┤
│ NLP Service │ Orchestrator │ VNF Operator │ RAN │ CN │ TN │
└─────────────────────────────────────────────────────────────┘
```

## Security Components

### 1. OPA Gatekeeper
- **Purpose**: Admission control and policy enforcement
- **Location**: `security/gatekeeper/`
- **Features**:
  - Custom constraint templates for O-RAN components
  - Security context validation
  - Resource requirements enforcement
  - Image registry allowlisting
  - Label requirements

### 2. Pod Security Standards
- **Purpose**: Pod-level security enforcement
- **Location**: `security/pod-security/`
- **Features**:
  - Restricted security contexts for most components
  - Baseline security for TN components (network access required)
  - Security Context Constraints for OpenShift

### 3. RBAC Policies
- **Purpose**: Role-based access control
- **Location**: `security/rbac/`
- **Features**:
  - Service accounts for each O-RAN component
  - Least-privilege access principles
  - Cluster and namespace-scoped roles

### 4. Network Policies
- **Purpose**: Micro-segmentation and network security
- **Location**: `security/network-policies/`
- **Features**:
  - Default deny-all policies
  - Explicit allow rules for O-RAN communication
  - DNS resolution allowances
  - Monitoring access patterns

### 5. Secret Management
- **Purpose**: Secure secret storage and encryption
- **Location**: `security/secrets/`
- **Features**:
  - Sealed Secrets for GitOps-compatible secret management
  - Encryption at rest
  - Separate secrets per component

### 6. Container Security Scanning
- **Purpose**: Vulnerability detection and compliance
- **Location**: `security/scanning/`
- **Features**:
  - Trivy Operator for automated scanning
  - Daily vulnerability scans
  - CIS compliance checking
  - RBAC assessment

### 7. Supply Chain Security
- **Purpose**: SLSA compliance and build provenance
- **Location**: `security/slsa/`
- **Features**:
  - Tekton Chains integration
  - SLSA Level 3 provenance
  - Container image signing with Cosign
  - SBOM generation

### 8. Runtime Security
- **Purpose**: Real-time threat detection
- **Location**: `observability/falco/`
- **Features**:
  - Falco runtime security monitoring
  - O-RAN specific security rules
  - Kubernetes audit integration
  - Real-time alerting

## Monitoring Stack

### 1. Prometheus
- **Purpose**: Metrics collection and storage
- **Location**: `monitoring/prometheus/`
- **Features**:
  - O-RAN component metric collection
  - Thesis performance target recording rules
  - 15-day retention
  - High availability configuration

### 2. Grafana
- **Purpose**: Visualization and dashboards
- **Location**: `monitoring/grafana/`
- **Features**:
  - O-RAN Intent MANO overview dashboard
  - Security monitoring dashboard
  - Thesis performance tracking
  - Alert visualization

### 3. Alertmanager
- **Purpose**: Alert routing and notification
- **Location**: `monitoring/alerting/`
- **Features**:
  - Thesis performance target alerting
  - Security incident notifications
  - Multi-channel alert routing
  - Alert grouping and deduplication

### 4. OpenTelemetry
- **Purpose**: Distributed tracing and observability
- **Location**: `monitoring/otel/`
- **Features**:
  - Automatic instrumentation
  - O-RAN workflow tracing
  - Metrics and trace correlation
  - Multi-language support

### 5. Loki
- **Purpose**: Log aggregation and analysis
- **Location**: `monitoring/logging/`
- **Features**:
  - Structured log collection
  - O-RAN component log parsing
  - Log-based alerting
  - Long-term log retention

### 6. Jaeger
- **Purpose**: Distributed tracing visualization
- **Location**: `observability/jaeger/`
- **Features**:
  - E2E trace visualization
  - Performance bottleneck identification
  - Service dependency mapping
  - Trace-metrics correlation

### 7. SLA Monitoring
- **Purpose**: Thesis performance tracking and regression detection
- **Location**: `monitoring/sla/`
- **Features**:
  - Automated thesis target monitoring
  - Performance regression detection
  - SLA compliance reporting
  - Automated notifications

## Deployment

### Prerequisites

1. **Kubernetes Cluster**: v1.25+ with the following:
   - Container runtime: containerd or Docker
   - Network CNI: Calico, Cilium, or similar
   - Storage class: Default storage class configured
   - Node access: For Falco kernel module installation

2. **Required Permissions**:
   - Cluster administrator access
   - Ability to create cluster-wide resources
   - Access to deploy privileged containers (for Falco)

3. **Resource Requirements**:
   - **Minimum**: 8 CPU cores, 16GB RAM across nodes
   - **Recommended**: 16 CPU cores, 32GB RAM across nodes
   - **Storage**: 200GB for monitoring data

### Quick Deployment

```bash
# 1. Deploy security and monitoring stack
kubectl apply -f deploy/security-monitoring-stack.yaml

# 2. Monitor deployment progress
kubectl logs -f job/deploy-security-monitoring -n kube-system

# 3. Verify deployment
kubectl get pods --all-namespaces | grep -E "(oran-|gatekeeper|trivy|falco|sealed-secrets)"
```

### Manual Deployment

For step-by-step deployment, follow the order in `deploy/security-monitoring-stack.yaml`:

```bash
# Phase 1: Security Foundation
kubectl apply -f security/pod-security/pod-security-standards.yaml
kubectl apply -f security/secrets/sealed-secrets.yaml
kubectl apply -f security/gatekeeper/gatekeeper-system.yaml
kubectl apply -f security/gatekeeper/constraint-templates.yaml
kubectl apply -f security/gatekeeper/oran-constraints.yaml

# Phase 2: RBAC and Network Security
kubectl apply -f security/rbac/oran-rbac.yaml
kubectl apply -f security/network-policies/oran-network-policies.yaml

# Phase 3: Security Scanning and Compliance
kubectl apply -f security/scanning/trivy-operator.yaml
kubectl apply -f security/slsa/slsa-provenance.yaml

# Phase 4: Monitoring Foundation
kubectl apply -f monitoring/prometheus/prometheus-stack.yaml
kubectl apply -f monitoring/grafana/grafana-stack.yaml
kubectl apply -f monitoring/alerting/alertmanager.yaml

# Phase 5: Observability Stack
kubectl apply -f monitoring/otel/opentelemetry-operator.yaml
kubectl apply -f monitoring/logging/loki-stack.yaml
kubectl apply -f observability/jaeger/jaeger-stack.yaml

# Phase 6: Advanced Monitoring
kubectl apply -f monitoring/sla/sla-monitoring.yaml
kubectl apply -f observability/falco/falco-security.yaml

# Phase 7: Sealed Secrets
kubectl apply -f security/secrets/oran-sealed-secrets.yaml
```

## Configuration

### Accessing Dashboards

```bash
# Grafana (admin/admin by default)
kubectl port-forward -n oran-monitoring svc/grafana 3000:3000

# Prometheus
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090

# Alertmanager
kubectl port-forward -n oran-monitoring svc/alertmanager 9093:9093

# Jaeger
kubectl port-forward -n oran-monitoring svc/jaeger-query 16686:16686
```

### Customizing Alerts

Edit the alert rules in `monitoring/alerting/alertmanager.yaml`:

```yaml
# Example: Modify thesis performance thresholds
- alert: ORanSliceDeploymentTooSlow
  expr: oran:slice_deployment_duration_seconds:rate5m > 600  # 10 minutes
  for: 2m
  labels:
    severity: warning
    component: orchestrator
    thesis_target: deployment_time
```

### Configuring Security Policies

Modify constraints in `security/gatekeeper/oran-constraints.yaml`:

```yaml
# Example: Update allowed container registries
spec:
  parameters:
    allowedRegistries:
    - "gcr.io/"
    - "your-registry.com/"
```

### Secret Management

Generate sealed secrets using kubeseal:

```bash
# Install kubeseal
wget https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.5/kubeseal-0.24.5-linux-amd64.tar.gz
tar -xzf kubeseal-0.24.5-linux-amd64.tar.gz
sudo install -m 755 kubeseal /usr/local/bin/kubeseal

# Create sealed secret
echo -n mypassword | kubectl create secret generic mysecret --dry-run=client --from-file=password=/dev/stdin -o yaml | kubeseal -o yaml > mysealedsecret.yaml
```

## Thesis Performance Monitoring

The monitoring stack specifically tracks the thesis performance targets:

### Key Metrics

1. **E2E Deployment Time**: Target < 10 minutes
   - Metric: `oran:slice_deployment_duration_seconds:rate5m`
   - Alert: ORanSliceDeploymentTooSlow

2. **Throughput Targets**:
   - eMBB: 4.57 Mbps (`oran:network_slice_throughput_mbps:rate5m{slice_type="embb"}`)
   - URLLC: 2.77 Mbps (`oran:network_slice_throughput_mbps:rate5m{slice_type="urllc"}`)
   - mMTC: 0.93 Mbps (`oran:network_slice_throughput_mbps:rate5m{slice_type="mmtc"}`)

3. **Latency Targets**:
   - eMBB: 16.1 ms (`oran:ping_rtt_milliseconds:avg5m{slice_type="embb"}`)
   - URLLC: 15.7 ms (`oran:ping_rtt_milliseconds:avg5m{slice_type="urllc"}`)
   - mMTC: 6.3 ms (`oran:ping_rtt_milliseconds:avg5m{slice_type="mmtc"}`)

### SLA Monitoring

The SLA monitor automatically:
- Tracks compliance against thesis targets
- Detects performance regressions
- Generates compliance reports
- Sends alerts for violations

### Dashboards

1. **O-RAN Overview Dashboard**: Real-time thesis performance metrics
2. **Security Dashboard**: Security posture and threat detection
3. **SLA Compliance Dashboard**: Historical performance and compliance trends

## Security Policies

### Default Security Posture

1. **Pod Security Standards**: Restricted by default
2. **Network Policies**: Default deny with explicit allows
3. **RBAC**: Least privilege access
4. **Container Images**: Verified and scanned
5. **Secrets**: Encrypted at rest and in transit

### Security Monitoring

1. **Runtime Threats**: Falco detects:
   - Privilege escalation attempts
   - Unauthorized file access
   - Suspicious network connections
   - Container escape attempts
   - Cryptocurrency mining

2. **Vulnerability Management**: Trivy provides:
   - Daily container scans
   - CIS compliance checks
   - RBAC assessments
   - Security policy violations

3. **Supply Chain Security**: SLSA framework ensures:
   - Build provenance tracking
   - Container image signing
   - SBOM generation
   - Reproducible builds

### Incident Response

1. **Critical Alerts**: Immediate notification via:
   - PagerDuty integration
   - Slack webhooks
   - Email notifications

2. **Security Events**: Automated response:
   - Falco alerts to Alertmanager
   - Policy violations logged
   - Compliance violations tracked

## Troubleshooting

### Common Issues

1. **Gatekeeper Admission Failures**:
   ```bash
   # Check constraint violations
   kubectl get events --field-selector reason=ConstraintViolation

   # Review constraint status
   kubectl describe constraint oran-security-context
   ```

2. **Network Policy Blocking Traffic**:
   ```bash
   # Test connectivity
   kubectl exec -n oran-nlp deployment/nlp-service -- curl http://orchestrator.oran-orchestrator:8080/health

   # Review network policies
   kubectl get networkpolicies -n oran-nlp
   ```

3. **Monitoring Data Missing**:
   ```bash
   # Check Prometheus targets
   kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090
   # Visit http://localhost:9090/targets

   # Check service discovery
   kubectl get servicemonitor -n oran-monitoring
   ```

4. **Falco Not Starting**:
   ```bash
   # Check kernel module
   kubectl logs daemonset/falco -n falco-system

   # Verify privileged access
   kubectl describe daemonset falco -n falco-system
   ```

### Performance Tuning

1. **Prometheus Storage**:
   - Adjust retention: `--storage.tsdb.retention.time=30d`
   - Increase storage: Update PVC size

2. **Loki Performance**:
   - Configure object storage for better performance
   - Adjust ingestion limits based on log volume

3. **Jaeger Sampling**:
   - Reduce sampling rate for high-volume services
   - Configure adaptive sampling

### Support and Maintenance

1. **Regular Tasks**:
   - Review security scan results weekly
   - Update container images monthly
   - Rotate secrets quarterly
   - Review RBAC permissions quarterly

2. **Monitoring Health**:
   - Monitor dashboard availability
   - Check alert rule effectiveness
   - Review log retention policies
   - Validate backup procedures

For additional support, refer to the individual component documentation or file issues in the project repository.