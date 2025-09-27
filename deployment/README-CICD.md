# O-RAN MANO CI/CD Pipeline Documentation

## Overview

This document provides comprehensive documentation for the O-RAN MANO CI/CD pipeline, including automated deployment, monitoring validation, performance regression detection, and failure reporting mechanisms.

## 🚀 Pipeline Components

### 1. GitHub Workflows

#### **deploy-monitoring.yml**
**Purpose**: Automated deployment testing and validation
**Triggers**:
- Push to `main`/`develop` branches
- Changes to `monitoring/` or `deployment/` directories
- Manual workflow dispatch

**Key Features**:
- ✅ Kubernetes YAML linting and validation
- ✅ Prometheus rules validation with promtool
- ✅ Grafana dashboard JSON validation
- ✅ Kind cluster deployment testing
- ✅ E2E testing with real monitoring stack
- ✅ Performance benchmarking
- ✅ Automatic promotion to staging

**Stages**:
1. **Lint & Validate** - Syntax and configuration validation
2. **Prometheus Rules** - PromQL and AlertManager validation
3. **Grafana Dashboards** - Dashboard JSON and datasource validation
4. **Deploy Test Cluster** - Kind cluster with full monitoring stack
5. **Performance Benchmarks** - Load testing and performance validation
6. **Promote to Staging** - Automatic promotion on success

#### **validate-metrics.yml**
**Purpose**: Scheduled monitoring health validation
**Triggers**:
- Nightly at 2 AM UTC
- Every 6 hours for critical monitoring
- Manual workflow dispatch

**Key Features**:
- 🔍 Prometheus target health validation
- 📊 Metric cardinality and format checking
- 🌐 Grafana datasource connectivity testing
- 📈 Metric drift and anomaly detection
- 📧 Comprehensive notification system (Slack, Teams, Email)

**Validation Areas**:
- Prometheus target health and scrape success rates
- Metric naming conventions and cardinality limits
- Grafana datasource health and connectivity
- Historical data analysis for drift detection
- Alert rule syntax and evaluation

### 2. Deployment Scripts

#### **ci-deploy.sh**
**Location**: `/deployment/scripts/ci-deploy.sh`
**Purpose**: Complete automation of monitoring stack deployment

**Features**:
- 🏗️ **Kind cluster creation** with optimized configuration
- 📦 **Dependency installation** (cert-manager, NGINX ingress)
- ⚙️ **Prometheus Operator** deployment via Helm
- 🔧 **O-RAN ServiceMonitors** for component monitoring
- ✅ **Deployment verification** and health checks
- 📊 **Performance validation** after deployment
- 🧹 **Automatic cleanup** on failure

**Usage**:
```bash
# Deploy complete monitoring stack
./ci-deploy.sh

# Clean up test cluster
./ci-deploy.sh cleanup

# Verify existing deployment
./ci-deploy.sh verify
```

#### **ci-validation.sh**
**Location**: `/monitoring/tests/ci-validation.sh`
**Purpose**: Comprehensive monitoring stack validation

**Validation Tests**:
- 🎯 Prometheus target health (>95% success rate required)
- 🔗 Grafana datasource connectivity
- 📏 Alert rule syntax and evaluation
- 🏷️ ServiceMonitor selector validation
- 💾 Resource usage and storage validation
- 🌐 Network connectivity testing
- 🔍 Metric query performance testing

**Output**: Detailed test report with pass/fail status for each component

### 3. Infrastructure as Code

#### **Terraform Modules**
**Location**: `/deployment/terraform/`

**Files**:
- `main.tf` - Core infrastructure and namespace management
- `prometheus.tf` - Monitoring stack with Helm charts
- `variables.tf` - Comprehensive configuration variables
- `outputs.tf` - Service endpoints and access information

**Features**:
- 🏗️ **Kubernetes resource management** (namespaces, RBAC, storage)
- 📊 **Prometheus Operator** deployment with custom values
- 🎨 **Grafana configuration** with custom dashboards
- 🚨 **AlertManager** setup with routing rules
- 🔒 **Security configurations** (NetworkPolicies, RBAC)
- 🌐 **Ingress management** with TLS termination

**Usage**:
```bash
cd deployment/terraform
terraform init
terraform plan
terraform apply
```

#### **Kustomize Overlays**
**Location**: `/monitoring/kustomize/`

**Structure**:
```
kustomize/
├── base/                    # Base configurations
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   └── servicemonitors/
└── overlays/
    ├── dev/                 # Development environment
    ├── staging/             # Staging environment
    └── prod/                # Production environment
```

**Environment-Specific Features**:
- **Development**: Minimal resources, latest images, debugging enabled
- **Staging**: Production-like setup, stable images, performance monitoring
- **Production**: High availability, pinned versions, comprehensive alerting

### 4. Rollback Procedures

#### **rollback-monitoring.sh**
**Location**: `/deployment/scripts/rollback/rollback-monitoring.sh`
**Purpose**: Comprehensive rollback capabilities

**Rollback Methods**:
- 🔄 **Helm rollback** to previous successful revision
- 📄 **Kubectl manifest** rollback from backup
- 🔧 **Kustomize overlay** rollback to previous state
- 💾 **Backup restoration** from specific backup point

**Features**:
- 📦 **Automatic backup** before rollback operations
- ✅ **Rollback verification** with health checks
- 📊 **Detailed reporting** of rollback operations
- 🧹 **Cleanup procedures** for failed rollbacks

#### **emergency-rollback.sh**
**Location**: `/deployment/scripts/rollback/emergency-rollback.sh`
**Purpose**: Rapid emergency recovery procedures

**Emergency Actions**:
- ⚠️ **Immediate backup** of current state
- 🛑 **Stop all monitoring** components
- 🗑️ **Delete problematic** resources and finalizers
- 📦 **Uninstall Helm** releases safely
- 🧹 **Namespace cleanup** with data preservation options
- 🚀 **Minimal monitoring** restoration

### 5. Performance Testing

#### **performance-regression-test.sh**
**Location**: `/deployment/scripts/performance/performance-regression-test.sh`
**Purpose**: Automated performance regression detection

**Performance Metrics**:
- ⏱️ **Response times** (Prometheus <2s, Grafana <1s)
- 💾 **Resource usage** (CPU <80%, Memory <80%)
- 📊 **Query performance** (PromQL queries <5s)
- 🎯 **Scrape success rate** (>95% target health)
- 🔄 **Load testing** with concurrent requests

**Baseline Management**:
- 📈 **Baseline creation** from known good performance
- 📊 **Regression detection** with configurable thresholds
- 📋 **Detailed reporting** with performance comparisons
- 🚨 **Automated alerting** on performance degradation

### 6. Failure Reporting

#### **failure-reporter.sh**
**Location**: `/deployment/scripts/reporting/failure-reporter.sh`
**Purpose**: Comprehensive failure analysis and notification

**Failure Analysis**:
- 🔍 **System information** collection (OS, K8s, cluster state)
- 📦 **Pod failure analysis** with restart counts and events
- 🌐 **Service issue detection** (missing endpoints, connectivity)
- 📊 **Monitoring health** assessment
- 📝 **Log collection** from all monitoring components

**Notification Channels**:
- 💬 **Slack integration** with rich formatting
- 📧 **Email notifications** with detailed reports
- 🔗 **Microsoft Teams** webhook support
- 🐛 **GitHub issue** creation with automation

**Report Formats**:
- 📊 **JSON reports** for programmatic analysis
- 📝 **Markdown reports** for human readability
- 📈 **Performance data** with trend analysis

### 7. Validation Testing

#### **validate-cicd.sh**
**Location**: `/deployment/tests/validate-cicd.sh`
**Purpose**: Complete CI/CD pipeline validation

**Validation Areas**:
- ✅ **GitHub workflows** - YAML syntax, required fields, action validation
- 🔧 **Deployment scripts** - Bash syntax, executable permissions, error handling
- 🏗️ **Terraform configuration** - Syntax validation, best practices, security
- 🔧 **Kustomize overlays** - Build validation, environment-specific configs
- 📊 **Monitoring configuration** - YAML/JSON syntax, component validation
- 📦 **Dependencies** - Required tools availability and versions

## 🔧 Configuration

### Environment Variables

```bash
# Core Configuration
NAMESPACE=monitoring                    # Kubernetes namespace
PROMETHEUS_URL=http://localhost:9090   # Prometheus endpoint
GRAFANA_URL=http://localhost:3000      # Grafana endpoint

# Notification Configuration
SLACK_WEBHOOK=https://hooks.slack.com/...
TEAMS_WEBHOOK=https://outlook.office.com/...
EMAIL_SMTP_HOST=smtp.company.com
EMAIL_FROM=monitoring@company.com

# Performance Thresholds
MAX_RESPONSE_TIME_MS=2000              # Maximum acceptable response time
MAX_CPU_USAGE_PERCENT=80               # CPU usage threshold
MAX_MEMORY_USAGE_PERCENT=80            # Memory usage threshold
MIN_SCRAPE_SUCCESS_RATE=95             # Minimum scrape success rate

# GitHub Integration
GITHUB_TOKEN=ghp_...                   # GitHub API token
GITHUB_REPOSITORY=org/repo             # Repository for issue creation
```

### Customization

#### Monitoring Stack Configuration
```yaml
# Custom Prometheus values
prometheus:
  retention: 15d
  storage: 50Gi
  replicas: 2

# Custom Grafana configuration
grafana:
  adminPassword: ${GRAFANA_PASSWORD}
  persistence:
    enabled: true
    size: 10Gi

# AlertManager configuration
alertmanager:
  enabled: true
  retention: 120h
  replicas: 2
```

#### Performance Thresholds
```bash
# Response time limits (milliseconds)
PROMETHEUS_MAX_RESPONSE=2000
GRAFANA_MAX_RESPONSE=1000
QUERY_MAX_DURATION=5000

# Resource usage limits (percentage)
MAX_CPU_THRESHOLD=80
MAX_MEMORY_THRESHOLD=80
MAX_DISK_USAGE=85

# Availability requirements
MIN_UPTIME_PERCENT=99.9
MIN_SCRAPE_SUCCESS=95
MAX_ERROR_RATE=0.1
```

## 📊 Monitoring and Alerting

### Key Metrics Monitored

1. **Infrastructure Metrics**:
   - Node CPU, memory, and disk usage
   - Network connectivity and latency
   - Kubernetes cluster health

2. **Application Metrics**:
   - VNF Operator performance and errors
   - Intent Management latency and throughput
   - O-RAN component availability

3. **Monitoring Stack Metrics**:
   - Prometheus query performance
   - Grafana dashboard load times
   - AlertManager notification delivery

### Alert Conditions

#### Critical Alerts
- 🚨 **Monitoring stack down** (>5 minutes)
- 🚨 **High error rate** (>5% for 2 minutes)
- 🚨 **Memory usage critical** (>90% for 5 minutes)
- 🚨 **Disk space critical** (>95% usage)

#### Warning Alerts
- ⚠️ **Performance degradation** (response time >2s)
- ⚠️ **Resource usage high** (>80% for 10 minutes)
- ⚠️ **Scrape targets failing** (<95% success rate)
- ⚠️ **Certificate expiring** (<30 days)

## 🚀 Quick Start Guide

### 1. Prerequisites
```bash
# Install required tools
kubectl version --client
helm version
docker --version
kind --version

# Verify cluster access
kubectl cluster-info
```

### 2. Deploy Monitoring Stack
```bash
# Clone repository
git clone <repository-url>
cd O-RAN-Intent-MANO-for-Network-Slicing

# Deploy using automation script
./deployment/scripts/ci-deploy.sh

# Verify deployment
./monitoring/tests/ci-validation.sh
```

### 3. Access Monitoring Services
```bash
# Port forward to access services
kubectl port-forward -n monitoring svc/prometheus-operator-kube-p-prometheus 9090:9090 &
kubectl port-forward -n monitoring svc/prometheus-operator-grafana 3000:80 &

# Access URLs
open http://localhost:9090    # Prometheus
open http://localhost:3000    # Grafana (admin/admin123)
```

### 4. Run Performance Tests
```bash
# Create performance baseline
./deployment/scripts/performance/performance-regression-test.sh baseline

# Run performance validation
./deployment/scripts/performance/performance-regression-test.sh full
```

### 5. Test Failure Scenarios
```bash
# Generate failure report
./deployment/scripts/reporting/failure-reporter.sh general

# Test rollback procedures
./deployment/scripts/rollback/rollback-monitoring.sh help
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Deployment Failures
```bash
# Check cluster state
kubectl get pods -n monitoring
kubectl describe pods -n monitoring

# View deployment logs
kubectl logs -l app.kubernetes.io/name=prometheus -n monitoring

# Run diagnostics
./deployment/scripts/reporting/failure-reporter.sh deployment
```

#### 2. Performance Issues
```bash
# Run performance tests
./deployment/scripts/performance/performance-regression-test.sh prometheus

# Check resource usage
kubectl top nodes
kubectl top pods -n monitoring

# Analyze metrics
curl http://localhost:9090/api/v1/query?query=up
```

#### 3. Validation Failures
```bash
# Validate CI/CD components
./deployment/tests/validate-cicd.sh full

# Check specific components
./deployment/tests/validate-cicd.sh workflows
./deployment/tests/validate-cicd.sh scripts
```

### Recovery Procedures

#### 1. Standard Rollback
```bash
# Rollback to previous Helm revision
./deployment/scripts/rollback/rollback-monitoring.sh helm

# Rollback using backup
./deployment/scripts/rollback/rollback-monitoring.sh backup /path/to/backup
```

#### 2. Emergency Recovery
```bash
# Emergency rollback (destructive)
./deployment/scripts/rollback/emergency-rollback.sh full

# Restore minimal monitoring
./deployment/scripts/rollback/emergency-rollback.sh restore
```

## 📈 Performance Benchmarks

### Expected Performance Metrics

| Component | Metric | Target | Critical |
|-----------|--------|--------|----------|
| Prometheus | Response Time | <1s | <2s |
| Grafana | Dashboard Load | <2s | <5s |
| Query Performance | PromQL Queries | <1s | <5s |
| Scrape Success | Target Health | >99% | >95% |
| CPU Usage | Monitoring Pods | <60% | <80% |
| Memory Usage | Monitoring Pods | <70% | <80% |

### Load Testing Results

- **Concurrent Users**: 50 simultaneous requests
- **Test Duration**: 5 minutes
- **Success Rate**: >99.5%
- **Average Response Time**: <500ms
- **P95 Response Time**: <1s
- **P99 Response Time**: <2s

## 🛡️ Security Considerations

### Security Features Implemented

1. **RBAC Configuration**:
   - Least privilege access for monitoring components
   - Service account isolation
   - Cluster-wide role restrictions

2. **Network Policies**:
   - Traffic segmentation between namespaces
   - Ingress/egress rules for monitoring stack
   - External access restrictions

3. **Secret Management**:
   - Kubernetes secrets for sensitive data
   - External secret management integration
   - Automatic secret rotation capabilities

4. **TLS/Encryption**:
   - TLS termination at ingress
   - Inter-service communication encryption
   - Certificate management automation

### Security Validation

```bash
# Security scan
./deployment/scripts/reporting/failure-reporter.sh security

# Validate RBAC
kubectl auth can-i --list --as=system:serviceaccount:monitoring:prometheus

# Check network policies
kubectl get networkpolicies -n monitoring
```

## 📚 Additional Resources

### Documentation
- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [Grafana Administration Guide](https://grafana.com/docs/grafana/latest/administration/)
- [Kubernetes Monitoring Best Practices](https://kubernetes.io/docs/concepts/cluster-administration/monitoring/)

### Monitoring Dashboards
- O-RAN Overview Dashboard
- VNF Operator Performance
- Intent Management Metrics
- Infrastructure Health
- SLA Monitoring

### Alert Runbooks
- [Prometheus Down Runbook](./runbooks/prometheus-down.md)
- [High CPU Usage Runbook](./runbooks/high-cpu.md)
- [Disk Space Critical Runbook](./runbooks/disk-space.md)
- [Network Connectivity Runbook](./runbooks/network-issues.md)

---

**Note**: This CI/CD pipeline is designed for production use and includes comprehensive testing, monitoring, and recovery mechanisms. Always test changes in development environments before applying to production systems.