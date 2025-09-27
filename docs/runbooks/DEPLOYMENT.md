# O-RAN Monitoring Stack Deployment Guide

This document provides comprehensive instructions for deploying and configuring the O-RAN monitoring stack.

## Overview

The O-RAN monitoring stack consists of:
- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **AlertManager**: Alert routing and management
- **OpenTelemetry**: Observability data collection
- **Loki**: Log aggregation (optional)

## Prerequisites

### System Requirements

| Component | CPU | Memory | Storage | Notes |
|-----------|-----|---------|---------|-------|
| Prometheus | 2 cores | 4GB | 100GB | SSD recommended |
| Grafana | 1 core | 2GB | 10GB | For dashboards and config |
| AlertManager | 0.5 cores | 1GB | 5GB | Minimal resource needs |
| Total | 3.5 cores | 7GB | 115GB | Plus 20% overhead |

### Kubernetes Requirements

- **Kubernetes Version**: 1.24+
- **RBAC**: Enabled
- **Storage Class**: Default storage class configured
- **Networking**: Pod-to-pod communication enabled
- **DNS**: CoreDNS or equivalent

### Dependencies

```bash
# Required tools
kubectl >= 1.24
helm >= 3.8
jq >= 1.6
curl >= 7.68

# Optional tools (for advanced operations)
yq >= 4.0
envsubst
```

## Pre-deployment Checklist

### 1. Cluster Validation

```bash
# Check cluster status
kubectl cluster-info

# Verify nodes are ready
kubectl get nodes

# Check storage classes
kubectl get storageclass

# Verify RBAC
kubectl auth can-i create clusterrole --as=system:serviceaccount:default:default
```

### 2. Namespace Preparation

```bash
# Create monitoring namespace
kubectl create namespace oran-monitoring

# Apply security policies
kubectl label namespace oran-monitoring \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

### 3. Resource Validation

```bash
# Check available resources
kubectl top nodes
kubectl describe nodes | grep -A 3 "Allocatable:"

# Verify storage capacity
kubectl get pv
```

## Deployment Methods

### Method 1: Kubectl Direct Deployment (Recommended for Production)

#### Step 1: Deploy Prometheus

```bash
# Apply Prometheus stack
kubectl apply -f monitoring/prometheus/prometheus-stack.yaml

# Verify deployment
kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=prometheus
kubectl logs -n oran-monitoring deployment/prometheus
```

#### Step 2: Deploy Grafana

```bash
# Apply Grafana stack
kubectl apply -f monitoring/grafana/grafana-stack.yaml

# Verify deployment
kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=grafana
kubectl logs -n oran-monitoring deployment/grafana
```

#### Step 3: Deploy AlertManager

```bash
# Apply AlertManager stack
kubectl apply -f monitoring/alerting/alertmanager.yaml

# Verify deployment
kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=alertmanager
kubectl logs -n oran-monitoring deployment/alertmanager
```

#### Step 4: Deploy Supporting Components

```bash
# Deploy OpenTelemetry Operator
kubectl apply -f monitoring/otel/opentelemetry-operator.yaml

# Deploy SLA monitoring
kubectl apply -f monitoring/sla/sla-monitoring.yaml

# Optional: Deploy Loki stack
kubectl apply -f monitoring/logging/loki-stack.yaml
```

### Method 2: Helm Deployment (Recommended for Development)

#### Step 1: Add Helm Repositories

```bash
# Add required helm repositories
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts

# Update repositories
helm repo update
```

#### Step 2: Install Prometheus Stack

```bash
# Create values file for Prometheus
cat > prometheus-values.yaml << EOF
prometheus:
  prometheusSpec:
    retention: 15d
    retentionSize: 50GB
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 100Gi
    resources:
      limits:
        cpu: 2000m
        memory: 4Gi
      requests:
        cpu: 500m
        memory: 2Gi

grafana:
  enabled: true
  adminPassword: "secure-password-here"
  persistence:
    enabled: true
    size: 10Gi

alertmanager:
  enabled: true
  alertmanagerSpec:
    storage:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 5Gi
EOF

# Install using Helm
helm install oran-monitoring prometheus-community/kube-prometheus-stack \
  --namespace oran-monitoring \
  --create-namespace \
  --values prometheus-values.yaml \
  --wait --timeout 10m
```

### Method 3: GitOps Deployment (Recommended for Enterprise)

#### Step 1: Prepare GitOps Repository

```bash
# Clone your GitOps repository
git clone https://github.com/your-org/oran-gitops.git
cd oran-gitops

# Create monitoring directory structure
mkdir -p apps/monitoring/{prometheus,grafana,alertmanager}
```

#### Step 2: Create ArgoCD Application

```yaml
# apps/monitoring/application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: oran-monitoring
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/oran-gitops.git
    targetRevision: HEAD
    path: apps/monitoring
  destination:
    server: https://kubernetes.default.svc
    namespace: oran-monitoring
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

## Configuration

### Prometheus Configuration

#### Custom Scrape Configs

```yaml
# Add to prometheus-config ConfigMap
scrape_configs:
# O-RAN Intent Service
- job_name: 'oran-intent-service'
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - oran-nlp
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_name]
    action: keep
    regex: intent-service
  - target_label: component
    replacement: intent-processing

# O-RAN Slice Manager
- job_name: 'oran-slice-manager'
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - oran-orchestrator
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_name]
    action: keep
    regex: slice-manager
  - target_label: component
    replacement: slice-management
```

#### Recording Rules

```yaml
# Add to recording-rules ConfigMap
groups:
- name: oran.performance
  interval: 30s
  rules:
  # Intent processing SLA
  - record: oran:intent_processing_sla_compliance
    expr: |
      (
        rate(oran_intent_processing_duration_seconds_sum[5m]) /
        rate(oran_intent_processing_duration_seconds_count[5m])
      ) < 2.0

  # Slice deployment success rate
  - record: oran:slice_deployment_success_rate
    expr: |
      rate(oran_slice_deployment_total{status="success"}[5m]) /
      rate(oran_slice_deployment_total[5m])

  # E2E latency percentiles
  - record: oran:e2e_latency_p99
    expr: |
      histogram_quantile(0.99,
        rate(oran_e2e_request_duration_seconds_bucket[5m])
      )
```

### Grafana Configuration

#### Data Source Configuration

```yaml
# grafana-datasource.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: oran-monitoring
data:
  datasources.yaml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
      editable: false
    - name: Loki
      type: loki
      access: proxy
      url: http://loki:3100
      editable: false
```

#### Dashboard Provisioning

```yaml
# grafana-dashboard-provider.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboard-provider
  namespace: oran-monitoring
data:
  dashboards.yaml: |
    apiVersion: 1
    providers:
    - name: 'oran-dashboards'
      orgId: 1
      folder: 'O-RAN'
      type: file
      disableDeletion: true
      updateIntervalSeconds: 30
      options:
        path: /var/lib/grafana/dashboards/oran
```

### AlertManager Configuration

#### Routing Configuration

```yaml
# alertmanager-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
  namespace: oran-monitoring
stringData:
  alertmanager.yml: |
    global:
      smtp_smarthost: 'smtp.company.com:587'
      smtp_from: 'oran-alerts@company.com'

    route:
      group_by: ['alertname']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 1h
      receiver: 'web.hook'
      routes:
      - match:
          severity: critical
        receiver: 'critical-alerts'
      - match:
          component: 'oran-intent'
        receiver: 'oran-team'

    receivers:
    - name: 'web.hook'
      webhook_configs:
      - url: 'http://webhook-service:8080/alerts'

    - name: 'critical-alerts'
      email_configs:
      - to: 'oncall@company.com'
        subject: 'CRITICAL: O-RAN Alert'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
      slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#oran-alerts'
        title: 'Critical O-RAN Alert'

    - name: 'oran-team'
      email_configs:
      - to: 'oran-team@company.com'
        subject: 'O-RAN Component Alert'
```

## Verification and Testing

### Deployment Verification

```bash
# Run comprehensive health checks
./deployment/kubernetes/health-checks/check-prometheus-targets.sh
./deployment/kubernetes/health-checks/check-grafana-dashboards.sh
./deployment/kubernetes/health-checks/check-metrics-flow.sh
./deployment/kubernetes/health-checks/check-alerts.sh

# Run Helm tests
helm test oran-monitoring -n oran-monitoring

# Verify with kubectl
kubectl get all -n oran-monitoring
kubectl get pvc -n oran-monitoring
kubectl get configmap -n oran-monitoring
kubectl get secret -n oran-monitoring
```

### Functional Testing

```bash
# Test Prometheus queries
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/query?query=up"

# Test Grafana access
kubectl port-forward -n oran-monitoring svc/grafana 3000:3000 &
curl -u admin:admin "http://localhost:3000/api/health"

# Test AlertManager
kubectl port-forward -n oran-monitoring svc/alertmanager 9093:9093 &
curl "http://localhost:9093/api/v2/status"
```

### Performance Testing

```bash
# Run performance tests
go test -v ./tests/performance/monitoring_performance_test.go

# Check resource usage
kubectl top pods -n oran-monitoring
kubectl describe nodes | grep -A 10 "Non-terminated Pods"
```

## Security Configuration

### RBAC Setup

```yaml
# monitoring-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: monitoring-operator
  namespace: oran-monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: monitoring-operator
rules:
- apiGroups: [""]
  resources: ["nodes", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "statefulsets"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: monitoring-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: monitoring-operator
subjects:
- kind: ServiceAccount
  name: monitoring-operator
  namespace: oran-monitoring
```

### Network Policies

```yaml
# monitoring-network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: monitoring-network-policy
  namespace: oran-monitoring
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: oran-monitoring
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 9090
    - protocol: TCP
      port: 3000
    - protocol: TCP
      port: 9093
  egress:
  - {}
```

### TLS Configuration

```bash
# Generate certificates for internal communication
kubectl create secret tls prometheus-tls \
  --cert=prometheus.crt \
  --key=prometheus.key \
  -n oran-monitoring

kubectl create secret tls grafana-tls \
  --cert=grafana.crt \
  --key=grafana.key \
  -n oran-monitoring
```

## Backup and Recovery

### Backup Prometheus Data

```bash
# Create backup script
cat > backup-prometheus.sh << 'EOF'
#!/bin/bash
NAMESPACE="oran-monitoring"
BACKUP_DIR="/backups/prometheus/$(date +%Y%m%d-%H%M%S)"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup Prometheus data
kubectl exec -n "$NAMESPACE" deployment/prometheus -- \
  tar czf - /prometheus | gzip > "$BACKUP_DIR/prometheus-data.tar.gz"

# Backup configuration
kubectl get configmap -n "$NAMESPACE" prometheus-config -o yaml > \
  "$BACKUP_DIR/prometheus-config.yaml"

# Backup rules
kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus \
  -o yaml > "$BACKUP_DIR/prometheus-rules.yaml"

echo "Backup completed: $BACKUP_DIR"
EOF

chmod +x backup-prometheus.sh
```

### Backup Grafana

```bash
# Backup Grafana dashboards and configuration
cat > backup-grafana.sh << 'EOF'
#!/bin/bash
NAMESPACE="oran-monitoring"
BACKUP_DIR="/backups/grafana/$(date +%Y%m%d-%H%M%S)"

mkdir -p "$BACKUP_DIR"

# Backup Grafana database
kubectl exec -n "$NAMESPACE" deployment/grafana -- \
  sqlite3 /var/lib/grafana/grafana.db .dump > "$BACKUP_DIR/grafana.sql"

# Backup configuration
kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name=grafana \
  -o yaml > "$BACKUP_DIR/grafana-config.yaml"

# Backup secrets
kubectl get secret -n "$NAMESPACE" grafana-admin-credentials \
  -o yaml > "$BACKUP_DIR/grafana-secrets.yaml"

echo "Backup completed: $BACKUP_DIR"
EOF

chmod +x backup-grafana.sh
```

## Monitoring the Monitoring

### Self-Monitoring Setup

```yaml
# monitoring-monitoring.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: prometheus-self-monitoring
  namespace: oran-monitoring
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: prometheus
  endpoints:
  - port: web
    interval: 30s
    path: /metrics
---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: grafana-self-monitoring
  namespace: oran-monitoring
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: grafana
  endpoints:
  - port: web
    interval: 30s
    path: /metrics
```

### Health Check Automation

```bash
# Setup cron job for health checks
cat > monitoring-health-check.yaml << 'EOF'
apiVersion: batch/v1
kind: CronJob
metadata:
  name: monitoring-health-check
  namespace: oran-monitoring
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: health-checker
            image: curlimages/curl:latest
            command:
            - /bin/sh
            - -c
            - |
              ./deployment/kubernetes/health-checks/check-prometheus-targets.sh
              ./deployment/kubernetes/health-checks/check-grafana-dashboards.sh
          restartPolicy: OnFailure
EOF

kubectl apply -f monitoring-health-check.yaml
```

## Troubleshooting

### Common Issues

1. **Pod stuck in Pending state**
   ```bash
   kubectl describe pod <pod-name> -n oran-monitoring
   # Check resource constraints and storage availability
   ```

2. **Prometheus not scraping targets**
   ```bash
   kubectl logs -n oran-monitoring deployment/prometheus
   # Check RBAC permissions and network policies
   ```

3. **Grafana dashboards not loading**
   ```bash
   kubectl exec -n oran-monitoring deployment/grafana -- ls /var/lib/grafana/dashboards
   # Verify dashboard provisioning
   ```

### Diagnostic Commands

```bash
# Check all resources
kubectl get all -n oran-monitoring -o wide

# Check resource consumption
kubectl top pods -n oran-monitoring
kubectl top nodes

# Check storage
kubectl get pvc -n oran-monitoring
kubectl describe pvc -n oran-monitoring

# Check logs
kubectl logs -n oran-monitoring deployment/prometheus --tail=100
kubectl logs -n oran-monitoring deployment/grafana --tail=100
kubectl logs -n oran-monitoring deployment/alertmanager --tail=100
```

## Maintenance

### Regular Maintenance Tasks

1. **Weekly**
   - Review alert fatigue
   - Check storage usage
   - Verify backup completion

2. **Monthly**
   - Update dashboards
   - Review and tune alert rules
   - Performance optimization

3. **Quarterly**
   - Security patching
   - Version upgrades
   - Capacity planning

### Upgrade Procedures

```bash
# Backup before upgrade
./backup-prometheus.sh
./backup-grafana.sh

# Upgrade Prometheus
kubectl set image deployment/prometheus -n oran-monitoring \
  prometheus=prom/prometheus:v2.49.0

# Upgrade Grafana
kubectl set image deployment/grafana -n oran-monitoring \
  grafana=grafana/grafana:10.2.0

# Verify upgrades
kubectl rollout status deployment/prometheus -n oran-monitoring
kubectl rollout status deployment/grafana -n oran-monitoring
```

## Performance Tuning

### Prometheus Optimization

```yaml
# Prometheus performance tuning
args:
- '--config.file=/etc/prometheus/prometheus.yml'
- '--storage.tsdb.path=/prometheus/'
- '--storage.tsdb.retention.time=15d'
- '--storage.tsdb.retention.size=50GB'
- '--query.max-concurrency=50'
- '--query.max-samples=50000000'
- '--storage.tsdb.wal-compression'
- '--storage.tsdb.head-chunks-write-queue-size=10000'
```

### Grafana Optimization

```yaml
# Grafana performance settings
env:
- name: GF_USERS_ALLOW_SIGN_UP
  value: "false"
- name: GF_SECURITY_ADMIN_PASSWORD
  valueFrom:
    secretKeyRef:
      name: grafana-admin-credentials
      key: password
- name: GF_DATABASE_WAL
  value: "true"
- name: GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH
  value: "/var/lib/grafana/dashboards/oran/overview.json"
```

## Next Steps

After successful deployment:

1. **Configure Data Sources**: Connect to your O-RAN components
2. **Import Dashboards**: Use the provided O-RAN dashboards
3. **Setup Alerts**: Configure alerting rules for your environment
4. **Train Team**: Ensure your team understands the monitoring tools
5. **Document**: Create environment-specific documentation

## Support

For additional support:
- Check logs: `kubectl logs -n oran-monitoring <pod-name>`
- Review troubleshooting guide: [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
- Contact: oran-monitoring-team@company.com