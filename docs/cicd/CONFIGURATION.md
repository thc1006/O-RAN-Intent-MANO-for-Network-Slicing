# CI/CD Configuration Reference

## 🔧 Configuration Files Overview

This document provides a comprehensive reference for all CI/CD configuration files, environment variables, and settings used in the O-RAN Intent-MANO enhanced CI/CD pipeline.

## 📁 File Structure

```
.github/
├── workflows/
│   ├── enhanced-ci-v2.yml              # Main CI/CD pipeline
│   ├── deployment-automation.yml       # Multi-environment deployment
│   ├── performance-testing.yml         # Performance validation
│   └── monitoring-alerting.yml         # System monitoring
├── dependabot.yml                      # Dependency updates
└── CODEOWNERS                          # Code review assignments

docs/
├── cicd/
│   ├── README.md                       # Main documentation
│   ├── RUNBOOKS.md                     # Operational procedures
│   └── CONFIGURATION.md                # This file

.gosec.toml                             # Security scanning config
.golangci.yml                           # Go linting configuration
kind-config.yaml                        # Kubernetes test cluster
```

## 🌍 Environment Variables

### Global Pipeline Variables
```yaml
# Container Registry
REGISTRY: ghcr.io
IMAGE_PREFIX: ghcr.io/${{ github.repository_owner }}

# Language Versions
GO_VERSION: '1.24.7'
PYTHON_VERSION: '3.11'
NODE_VERSION: '20'

# Tool Versions
KIND_VERSION: 'v0.23.0'
KUBECTL_VERSION: 'v1.31.0'
HELM_VERSION: 'v3.16.2'
GOLANGCI_LINT_VERSION: 'v1.61.0'
TRIVY_VERSION: 'v0.58.0'
COSIGN_VERSION: 'v2.4.1'
```

### Quality Gates Configuration
```yaml
# Coverage and Testing
MIN_CODE_COVERAGE: '85'
MIN_TEST_SUCCESS_RATE: '95'
MAX_DEPLOYMENT_TIME_MINUTES: '8'

# Security Thresholds
MAX_CRITICAL_VULNERABILITIES: '0'
MAX_HIGH_VULNERABILITIES: '3'
MAX_MEDIUM_VULNERABILITIES: '10'
MAX_COMPLEXITY_VIOLATIONS: '2'
MAX_DUPLICATION_PERCENTAGE: '15'

# Performance Targets (Thesis Requirements)
URLLC_THROUGHPUT_TARGET: '4.57'        # Mbps
EMBB_THROUGHPUT_TARGET: '2.77'         # Mbps
MMTC_THROUGHPUT_TARGET: '0.93'         # Mbps
URLLC_LATENCY_TARGET: '6.3'            # ms
EMBB_LATENCY_TARGET: '15.7'            # ms
MMTC_LATENCY_TARGET: '16.1'            # ms
DEPLOYMENT_TIME_TARGET: '480'          # seconds
SCALING_TIME_TARGET: '60'              # seconds
```

### Feature Flags
```yaml
# Security Features
COSIGN_EXPERIMENTAL: 1
ENABLE_SBOM_GENERATION: true
ENABLE_SLSA_PROVENANCE: true
ENABLE_POLICY_ENFORCEMENT: true

# Monitoring Features
DASHBOARD_ENABLED: 'true'
PUBLISH_DASHBOARD: 'true'
GENERATE_REPORTS: 'true'
```

## 🔒 Security Configuration

### GoSec Configuration (`.gosec.toml`)
```toml
[global]
# Exclude false positives for security package
exclude = "G204"
exclude-generated = true
severity = "medium"
confidence = "medium"

[rules]
# Allow subprocess calls in security package
[[rules.G204]]
exclude = [
    "pkg/security/.*",
    "internal/security/.*"
]

[output]
format = "sarif"
stdout = true
verbose = "text"
```

### GolangCI-Lint Configuration (`.golangci.yml`)
```yaml
run:
  timeout: 15m
  modules-download-mode: readonly
  allow-parallel-runners: true

linters:
  enable:
    - gosec
    - gocritic
    - revive
    - staticcheck
    - unparam
    - unused
    - ineffassign
    - misspell
    - goconst
    - gocyclo
    - dupl
    - goimports
    - gomodguard
    - govet
    - errcheck

linters-settings:
  gosec:
    config:
      G204:
        # Allow subprocess calls in security package
        - "pkg/security/*"

  gocyclo:
    min-complexity: 10

  govet:
    check-shadowing: true

  dupl:
    threshold: 100

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl
    - path: pkg/security/
      linters:
        - gosec
      text: "G204"
```

### Trivy Configuration (`.trivy.yaml`)
```yaml
# Security scanner configuration
scan:
  security-checks:
    - vuln
    - config
    - secret

vulnerability:
  type:
    - os
    - library

severity:
    - CRITICAL
    - HIGH
    - MEDIUM

format: sarif
output: trivy-results.sarif

cache:
  backend: fs
  dir: .trivycache/
```

## 🧪 Test Configuration

### Unit Test Matrix
```yaml
strategy:
  fail-fast: false
  matrix:
    include:
      - component: orchestrator
        language: go
        path: orchestrator
        coverage-threshold: 85
        timeout: 12m

      - component: vnf-operator
        language: go
        path: adapters/vnf-operator
        coverage-threshold: 80
        timeout: 15m

      - component: o2-client
        language: go
        path: o2-client
        coverage-threshold: 85
        timeout: 10m

      - component: tn-manager
        language: go
        path: tn
        coverage-threshold: 75
        timeout: 12m

      - component: nlp
        language: python
        path: nlp
        coverage-threshold: 80
        timeout: 8m
```

### Integration Test Configuration
```yaml
strategy:
  fail-fast: false
  matrix:
    test-suite:
      - name: orchestrator-integration
        components: [orchestrator, o2-client]
        clusters: [central]
        timeout: 20m

      - name: vnf-operator-integration
        components: [vnf-operator, orchestrator]
        clusters: [central, edge01]
        timeout: 25m

      - name: network-slicing-integration
        components: [orchestrator, tn-manager, tn-agent]
        clusters: [central, edge01, edge02]
        timeout: 30m
```

### Performance Test Profiles
```yaml
load_profiles:
  light:
    virtual_users: 10
    requests_per_second: 50
    duration: 5m
    ramp_up_time: 1m

  moderate:
    virtual_users: 50
    requests_per_second: 200
    duration: 10m
    ramp_up_time: 2m

  heavy:
    virtual_users: 100
    requests_per_second: 500
    duration: 15m
    ramp_up_time: 3m

  stress:
    virtual_users: 200
    requests_per_second: 1000
    duration: 20m
    ramp_up_time: 5m
```

## 🚀 Deployment Configuration

### Environment-Specific Settings
```yaml
environments:
  dev:
    namespace: oran-dev
    replicas: 1
    resources:
      requests:
        cpu: 250m
        memory: 256Mi
      limits:
        cpu: 500m
        memory: 512Mi
    monitoring: basic

  staging:
    namespace: oran-staging
    replicas: 2
    resources:
      requests:
        cpu: 500m
        memory: 512Mi
      limits:
        cpu: 1000m
        memory: 1Gi
    monitoring: enhanced

  prod:
    namespace: oran-prod
    replicas: 3
    resources:
      requests:
        cpu: 1000m
        memory: 1Gi
      limits:
        cpu: 2000m
        memory: 2Gi
    monitoring: full
```

### Deployment Strategies
```yaml
strategies:
  rolling:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
    progressDeadlineSeconds: 600

  canary:
    initial_traffic_percent: 10
    monitoring_duration: 300  # 5 minutes
    promotion_criteria:
      - error_rate < 1%
      - response_time_p95 < baseline + 10%
      - cpu_usage < 80%

  blue_green:
    switch_traffic: instant
    keep_old_version: 300  # 5 minutes
    rollback_timeout: 60   # 1 minute
```

## 📊 Monitoring Configuration

### Prometheus Metrics
```yaml
prometheus:
  scrape_configs:
    - job_name: 'oran-components'
      kubernetes_sd_configs:
        - role: endpoints
          namespaces:
            names: [oran-dev, oran-staging, oran-prod]

      relabel_configs:
        - source_labels: [__meta_kubernetes_service_name]
          action: keep
          regex: oran-.*

      scrape_interval: 15s
      scrape_timeout: 10s

  rule_files:
    - "cicd_alerts.yml"
    - "performance_alerts.yml"
    - "security_alerts.yml"
```

### Alert Rules
```yaml
groups:
  - name: cicd_alerts
    rules:
      - alert: WorkflowFailureRate
        expr: rate(github_workflow_failures_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate"
          description: "Workflow failure rate is {{ $value }} failures per second"

      - alert: CriticalVulnerability
        expr: github_security_alerts{severity="critical"} > 0
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "Critical security vulnerability detected"
          description: "{{ $value }} critical vulnerabilities found"

      - alert: PerformanceDegradation
        expr: avg(deployment_duration_seconds) > 480
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Deployment performance degraded"
          description: "Average deployment time is {{ $value }}s (target: 480s)"
```

### Grafana Dashboard Configuration
```json
{
  "dashboard": {
    "title": "O-RAN Intent-MANO CI/CD Dashboard",
    "panels": [
      {
        "title": "Workflow Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(github_workflow_success_total[24h]) / rate(github_workflow_total[24h]) * 100"
          }
        ],
        "thresholds": [
          {"color": "red", "value": 0},
          {"color": "yellow", "value": 90},
          {"color": "green", "value": 95}
        ]
      },
      {
        "title": "Performance Metrics",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95 Latency"
          },
          {
            "expr": "rate(network_throughput_mbps[5m])",
            "legendFormat": "Throughput"
          }
        ]
      }
    ]
  }
}
```

## 🔄 Dependabot Configuration

### Dependabot Settings (`.github/dependabot.yml`)
```yaml
version: 2
updates:
  # Go modules
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
    open-pull-requests-limit: 5
    reviewers:
      - "ci-cd-team"
    assignees:
      - "security-team"
    labels:
      - "dependencies"
      - "go"
    commit-message:
      prefix: "build(deps)"
      include: "scope"

  # Python dependencies
  - package-ecosystem: "pip"
    directory: "/nlp"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 3
    labels:
      - "dependencies"
      - "python"

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "monthly"
    labels:
      - "dependencies"
      - "github-actions"

  # Docker base images
  - package-ecosystem: "docker"
    directory: "/deploy/docker"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "docker"
```

## 🏗️ Kind Cluster Configuration

### Test Cluster Setup (`kind-config.yaml`)
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: oran-test

# Multi-node cluster for realistic testing
nodes:
  # Control plane node
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"

    extraPortMappings:
      # Expose common ports for testing
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
      - containerPort: 30080
        hostPort: 30080
        protocol: TCP
      - containerPort: 30443
        hostPort: 30443
        protocol: TCP

  # Worker nodes with labels for component placement
  - role: worker
    labels:
      component-type: "core"
      test-node: "true"

  - role: worker
    labels:
      component-type: "edge"
      test-node: "true"

  - role: worker
    labels:
      component-type: "performance"
      test-node: "true"

# Network configuration for multi-cluster testing
networking:
  # Use different CIDR to avoid conflicts
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/16"

  # Disable default CNI for custom network testing
  disableDefaultCNI: false

  # Enable IPv6 for future testing
  ipFamily: ipv4

# Feature gates for testing
featureGates:
  "EphemeralContainers": true
  "GracefulNodeShutdown": true
```

## 📋 Workflow Dispatch Inputs

### Enhanced CI Pipeline Inputs
```yaml
workflow_dispatch:
  inputs:
    run_performance_tests:
      description: 'Run comprehensive performance tests'
      required: false
      default: false
      type: boolean

    run_e2e_tests:
      description: 'Run end-to-end integration tests'
      required: false
      default: false
      type: boolean

    run_chaos_tests:
      description: 'Run chaos engineering tests'
      required: false
      default: false
      type: boolean

    target_architecture:
      description: 'Target architectures for builds'
      required: false
      default: 'linux/amd64,linux/arm64'
      type: string

    skip_quality_gates:
      description: 'Skip quality gate validation (emergency only)'
      required: false
      default: false
      type: boolean

    deploy_environment:
      description: 'Target deployment environment'
      required: false
      default: 'dev'
      type: choice
      options:
        - 'dev'
        - 'staging'
        - 'prod'

    notification_channel:
      description: 'Notification channel for results'
      required: false
      default: 'slack'
      type: choice
      options:
        - 'slack'
        - 'email'
        - 'github'
        - 'none'
```

## 🔐 Secrets Management

### Required GitHub Secrets
```yaml
# Container Registry
GITHUB_TOKEN: # Automatically provided by GitHub

# Deployment Environments
DEV_KUBECONFIG: # Base64 encoded kubeconfig for dev
STAGING_KUBECONFIG: # Base64 encoded kubeconfig for staging
PROD_KUBECONFIG: # Base64 encoded kubeconfig for production

# External Services
SLACK_WEBHOOK: # Slack webhook URL for notifications
CODECOV_TOKEN: # Token for code coverage reporting
SONAR_TOKEN: # SonarQube authentication token

# Security Tools
SNYK_TOKEN: # Snyk security scanning token
COSIGN_PRIVATE_KEY: # Container signing private key
COSIGN_PASSWORD: # Container signing key password
```

### Secret Rotation Schedule
```yaml
rotation_schedule:
  high_priority:
    - COSIGN_PRIVATE_KEY: quarterly
    - PROD_KUBECONFIG: quarterly

  medium_priority:
    - STAGING_KUBECONFIG: bi-annually
    - SNYK_TOKEN: annually

  low_priority:
    - SLACK_WEBHOOK: annually
    - CODECOV_TOKEN: annually
```

## 📊 Performance Baselines

### Benchmark Targets
```yaml
performance_baselines:
  build_time:
    target: 480  # seconds (8 minutes)
    warning_threshold: 600  # 10 minutes
    critical_threshold: 900  # 15 minutes

  test_execution:
    unit_tests: 300  # 5 minutes
    integration_tests: 900  # 15 minutes
    performance_tests: 1800  # 30 minutes

  network_performance:
    urllc:
      latency_ms: 6.3
      throughput_mbps: 4.57
      reliability_percent: 99.999

    embb:
      latency_ms: 15.7
      throughput_mbps: 2.77
      reliability_percent: 99.9

    mmtc:
      latency_ms: 16.1
      throughput_mbps: 0.93
      reliability_percent: 99.0

  system_performance:
    deployment_time_seconds: 480
    scaling_time_seconds: 60
    recovery_time_seconds: 120
```

---

**Document Version**: 2.0
**Last Updated**: 2025-09-26
**Configuration Schema Version**: v2.1
**Compatibility**: GitHub Actions, Kubernetes 1.31+, Go 1.24+

For configuration updates or questions, please refer to the [main documentation](README.md) or create an issue with the `configuration` label.