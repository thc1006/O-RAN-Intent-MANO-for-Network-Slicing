# GitOps Validation Framework

A comprehensive validation framework for O-RAN Intent-based MANO system with Nephio/Porch integration, ensuring "GitOps packages render cleanly; kubectl resources Ready" as specified in the Definition of Done.

## Features

### Core Validation
- **Multi-cluster GitOps validation** - Validates GitOps deployments across edge01, edge02, regional, and central clusters
- **Nephio/Porch package integration** - Complete validation of Nephio packages and Porch repositories
- **ConfigSync and ArgoCD support** - Validates both ConfigSync and ArgoCD GitOps implementations
- **Resource readiness checking** - Ensures all Kubernetes resources are Ready and healthy
- **Git repository state validation** - Validates Git sync status and repository health

### Advanced Features
- **Automated rollback mechanisms** - Automatic rollback on validation failures
- **Drift detection and correction** - Detects configuration drift and can auto-correct
- **Multi-cluster package synchronization** - Orchestrates package deployment across clusters
- **Performance monitoring integration** - Collects and validates performance metrics
- **End-to-end deployment validation** - Complete E2E pipeline validation

### O-RAN Specific
- **Intent processing validation** - Validates O-RAN intent-to-QoS mapping
- **RAN/CN/TN component validation** - Validates O-RAN network function deployments
- **O2 interface validation** - Validates O2ims/O2dms interfaces
- **Performance thresholds** - Validates against O-RAN DoD requirements:
  - E2E deploy time < 10 minutes
  - DL throughput ≈ {4.57, 2.77, 0.93} Mbps
  - Ping RTT ≈ {16.1, 15.7, 6.3} ms

## Quick Start

### Prerequisites

- Go 1.21+
- Kubernetes clusters (edge01, edge02, regional, central)
- kubectl configured with cluster access
- Git repository with GitOps configurations
- Nephio/Porch (optional)
- Prometheus (optional, for metrics)

### Installation

```bash
# Clone the repository
git clone https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing.git
cd O-RAN-Intent-MANO-for-Network-Slicing/clusters/validation-framework

# Install dependencies
make deps

# Build the binary
make build

# Run validation
make run-validation
```

### Configuration

Create a `config.yaml` file with your cluster and GitOps configuration:

```yaml
clusters:
  - name: edge01
    type: edge
    kubeconfig: ${HOME}/.kube/edge01-config
    context: edge01-context
    packages:
      - ran-du-packages
      - tn-edge-packages

git:
  repoUrl: "https://github.com/your-org/gitops-repo.git"
  branch: main
  path: clusters

validation:
  readinessTimeout: 10m
  performanceThresholds:
    deploymentTime: 10m
    throughputMbps: [4.57, 2.77, 0.93]
    pingRttMs: [16.1, 15.7, 6.3]
```

## Usage

### Basic Validation

```bash
# Validate all clusters
./build/gitops-validator --config=config.yaml --validate-only

# Validate specific cluster
./build/gitops-validator --config=config.yaml --cluster=edge01 --validate-only

# Output to file
./build/gitops-validator --config=config.yaml --output-file=results.json
```

### Continuous Monitoring

```bash
# Start continuous monitoring
./build/gitops-validator --config=config.yaml --interval=5m

# Enable drift detection
./build/gitops-validator --config=config.yaml --enable-drift

# Enable metrics collection
./build/gitops-validator --config=config.yaml --enable-metrics
```

### E2E Pipeline

```bash
# Run complete E2E validation pipeline
make run-e2e

# Run with custom config
./build/gitops-validator --config=production.yaml --validate-only --enable-drift --enable-metrics
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    GitOps Validation Framework                  │
├─────────────────────────────────────────────────────────────────┤
│                     ValidationFramework                        │
│  ┌─────────────────┬─────────────────┬─────────────────────────┐ │
│  │   Cluster       │   Git Repo      │   Nephio/Porch         │ │
│  │   Clients       │   Integration   │   Integration          │ │
│  └─────────────────┴─────────────────┴─────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┬─────────────────┬─────────────────────────┐ │
│  │   ConfigSync    │   ArgoCD        │   Resource Ready        │ │
│  │   Validator     │   Validator     │   Checker              │ │
│  └─────────────────┴─────────────────┴─────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┬─────────────────┬─────────────────────────┐ │
│  │   Rollback      │   Drift         │   Sync Manager          │ │
│  │   Manager       │   Detector      │                        │ │
│  └─────────────────┴─────────────────┴─────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┬─────────────────┬─────────────────────────┐ │
│  │   Metrics       │   E2E Pipeline  │   Performance           │ │
│  │   Collector     │   Orchestrator  │   Validator            │ │
│  └─────────────────┴─────────────────┴─────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### Core Components

- **ValidationFramework**: Main orchestrator for all validation operations
- **ClusterClient**: Kubernetes API client wrapper for multi-cluster operations
- **GitRepository**: Git repository integration for GitOps validation
- **NephioValidator**: Nephio/Porch package validation and rendering

### GitOps Validators

- **ConfigSyncValidator**: Google Config Sync validation
- **ArgoCDValidator**: ArgoCD application and project validation

### Advanced Features

- **RollbackManager**: Automated rollback on validation failures
- **DriftDetector**: Configuration drift detection and correction
- **SyncManager**: Multi-cluster package synchronization
- **MetricsCollector**: Performance metrics collection and validation
- **E2EPipeline**: End-to-end deployment validation pipeline

## Validation Types

### 1. Package Validation
- Kptfile validation
- Resource rendering with kpt/kustomize
- Package structure validation
- Dependency validation

### 2. Deployment Validation
- Resource creation and readiness
- Health checks and status validation
- Service availability validation
- Custom resource validation

### 3. GitOps Validation
- Git repository sync status
- ConfigSync/ArgoCD status
- Package synchronization validation
- Drift detection

### 4. Performance Validation
- Deployment time validation (< 10 min)
- Throughput validation (4.57, 2.77, 0.93 Mbps)
- Latency validation (16.1, 15.7, 6.3 ms RTT)
- Resource utilization validation

## Configuration Reference

### Cluster Configuration

```yaml
clusters:
  - name: edge01
    type: edge
    kubeconfig: /path/to/kubeconfig
    context: cluster-context
    environment: production
    packages:
      - package-name
    capabilities:
      - ran-du
      - transport-network
    labels:
      cluster.oran.io/type: edge
      cluster.oran.io/location: edge01
```

### Validation Rules

```yaml
validation:
  readinessTimeout: 10m
  requiredResources:
    - apiVersion: mano.oran.io/v1alpha1
      kind: VNF
      namespace: ran-du
      labels:
        component: ran-du
  driftTolerance:
    maxDriftPercentage: 5.0
    checkInterval: 30s
    autoCorrect: true
  performanceThresholds:
    deploymentTime: 10m
    throughputMbps: [4.57, 2.77, 0.93]
    pingRttMs: [16.1, 15.7, 6.3]
```

### E2E Pipeline Configuration

```yaml
e2e:
  enabled: true
  stages:
    - name: git-sync
      type: git-sync
      timeout: 5m
    - name: package-validation
      type: package-validation
      timeout: 10m
      config:
        packages:
          - clusters/edge01/packages/ran-du
    - name: deployment
      type: deployment
      timeout: 15m
    - name: performance-test
      type: performance-test
      timeout: 5m
```

## Monitoring and Metrics

### Prometheus Integration

The framework integrates with Prometheus for metrics collection:

```yaml
monitoring:
  enabled: true
  prometheusUrl: "http://prometheus.monitoring.svc.cluster.local:9090"
  scrapeInterval: 30s
```

### Metrics Collected

- **Deployment metrics**: Time to deploy, success rate
- **Resource metrics**: CPU, memory, storage usage
- **Network metrics**: Throughput, latency, packet loss
- **Application metrics**: O-RAN component health and performance

### Grafana Dashboards

Pre-built Grafana dashboards are available for:
- GitOps validation status
- Multi-cluster deployment health
- O-RAN performance metrics
- Drift detection alerts

## Troubleshooting

### Common Issues

1. **Cluster Connection Issues**
   ```bash
   # Verify kubeconfig
   kubectl config current-context
   kubectl cluster-info
   ```

2. **Package Validation Failures**
   ```bash
   # Check package structure
   kpt fn render /path/to/package
   ```

3. **GitOps Sync Issues**
   ```bash
   # Check ConfigSync status
   kubectl get rootsyncs -n config-management-system

   # Check ArgoCD applications
   kubectl get applications -n argocd
   ```

### Debug Mode

```bash
# Enable debug logging
./build/gitops-validator --config=config.yaml --log-level=debug

# Verbose output
./build/gitops-validator --config=config.yaml --output=table --validate-only
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing.git
cd clusters/validation-framework

# Install dependencies
make deps

# Run tests
make test

# Build
make build

# Run linting
make lint
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make benchmark
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make check` to ensure all checks pass
6. Submit a pull request

## Docker

### Building Docker Image

```bash
# Build image
make docker-build

# Run container
make docker-run

# Push to registry
make docker-push
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitops-validator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gitops-validator
  template:
    metadata:
      labels:
        app: gitops-validator
    spec:
      containers:
      - name: validator
        image: ghcr.io/thc1006/gitops-validator:latest
        args:
          - --config=/etc/config/config.yaml
          - --interval=5m
        volumeMounts:
        - name: config
          mountPath: /etc/config
        - name: kubeconfig
          mountPath: /root/.kube
      volumes:
      - name: config
        configMap:
          name: validator-config
      - name: kubeconfig
        secret:
          secretName: kubeconfig
```

## License

Apache License 2.0 - see [LICENSE](../../LICENSE) for details.

## Support

- **Documentation**: [O-RAN Intent MANO Docs](../../docs/)
- **Issues**: [GitHub Issues](https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/issues)
- **Discussions**: [GitHub Discussions](https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/discussions)

## Related Projects

- [Nephio](https://nephio.org/) - Cloud native network function automation
- [Config Sync](https://cloud.google.com/kubernetes-engine/docs/add-on/config-sync) - GitOps for Kubernetes
- [ArgoCD](https://argo-cd.readthedocs.io/) - Declarative GitOps for Kubernetes
- [O-RAN Alliance](https://www.o-ran.org/) - Open RAN specifications