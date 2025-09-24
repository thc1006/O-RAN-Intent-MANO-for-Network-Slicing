# O-RAN Intent-MANO Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the O-RAN Intent-MANO system locally for development and testing. The system supports both Docker Compose and Kubernetes (Kind) deployments with full monitoring and testing capabilities.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Orchestrator  │◄──►│   VNF Operator  │◄──►│   O2 Client     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TN Manager    │◄──►│    RAN DMS      │◄──►│     CN DMS      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲
         │
         ▼
┌─────────────────┐    ┌─────────────────┐
│  TN Agent E01   │◄──►│  TN Agent E02   │
└─────────────────┘    └─────────────────┘
```

## Prerequisites

### System Requirements

- **OS**: Linux/macOS/Windows (with WSL2)
- **CPU**: 4+ cores
- **Memory**: 8GB+ RAM
- **Disk**: 20GB+ available space
- **Network**: Internet access for downloading dependencies

### Required Software

#### Docker Compose Deployment
- Docker 24.0+
- Docker Compose 2.20+
- curl, wget, jq
- openssl

#### Kubernetes Deployment
- Docker 24.0+
- kubectl 1.28+
- Kind 0.20+
- Helm 3.12+

## Quick Start

### Docker Compose (Recommended for Development)

1. **Clone and navigate to project**:
   ```bash
   cd /path/to/O-RAN-Intent-MANO-for-Network-Slicing
   ```

2. **Deploy with automated script**:
   ```bash
   chmod +x deploy/scripts/deploy-local.sh
   ./deploy/scripts/deploy-local.sh start
   ```

3. **Access services**:
   - Orchestrator: http://localhost:8080
   - Grafana: http://localhost:3000 (admin/admin123)
   - Prometheus: http://localhost:9090

### Kubernetes with Kind

1. **Deploy to Kind cluster**:
   ```bash
   chmod +x deploy/scripts/deploy-kubernetes.sh
   ./deploy/scripts/deploy-kubernetes.sh deploy
   ```

2. **Access services via port-forwarding**:
   ```bash
   kubectl port-forward -n oran-mano svc/oran-orchestrator 8080:8080
   ```

## Detailed Deployment Instructions

### Docker Compose Deployment

#### Configuration Overview

The Docker Compose deployment uses three main files:

- `docker-compose.local.yml` - Main services with enhanced networking
- `docker-compose.test.yml` - Testing services and frameworks
- `docker-compose.security.yml` - Security scanning and validation

#### Network Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Docker Networks                          │
├─────────────────────────────────────────────────────────────────┤
│ mano-net (172.20.0.0/16)    - Core MANO services              │
│ oran-edge (172.21.0.0/16)   - Edge TN agents                  │
│ oran-core (172.22.0.0/16)   - Core network functions          │
│ monitoring (172.23.0.0/16)  - Monitoring stack                │
└─────────────────────────────────────────────────────────────────┘
```

#### Step-by-Step Deployment

1. **Prepare environment**:
   ```bash
   # Set environment variables
   export LOG_LEVEL=debug
   export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
   export VERSION=v1.0.0-local
   export GRAFANA_PASSWORD=admin123
   ```

2. **Build all services**:
   ```bash
   cd deploy/docker
   docker-compose -f docker-compose.local.yml build --parallel
   ```

3. **Start infrastructure services**:
   ```bash
   docker-compose -f docker-compose.local.yml up -d \
     ran-dms cn-dms prometheus grafana
   ```

4. **Start core MANO services**:
   ```bash
   docker-compose -f docker-compose.local.yml up -d \
     orchestrator vnf-operator o2-client tn-manager
   ```

5. **Start edge services**:
   ```bash
   docker-compose -f docker-compose.local.yml up -d \
     tn-agent-edge01 tn-agent-edge02
   ```

6. **Verify deployment**:
   ```bash
   ./deploy/scripts/deploy-local.sh health
   ```

#### Service Port Mapping

| Service | HTTP Port | Metrics Port | Debug Port | Notes |
|---------|-----------|--------------|------------|-------|
| Orchestrator | 8080 | 9090 | 8180 | Main API |
| VNF Operator | 8081 | 8080 | - | Metrics on 8081 |
| O2 Client | 8083 | 9093 | - | O-RAN O2 interface |
| TN Manager | 8084 | 9091 | 8184 | Transport Network |
| TN Agent E01 | 8085 | - | 8185 | iPerf3: 5201 |
| TN Agent E02 | 8086 | - | 8186 | iPerf3: 5202 |
| RAN DMS | 8087 | 9087 | - | HTTPS: 8443 |
| CN DMS | 8088 | 9088 | - | HTTPS: 8444 |
| Prometheus | 9090 | - | - | Monitoring |
| Grafana | 3000 | - | - | Dashboards |

#### Volume Management

```bash
# Data volumes are stored in:
deploy/docker/data/
├── orchestrator/
├── vnf-operator/
├── o2-client/
├── tn-manager/
├── tn-agent/
├── ran-dms/
└── cn-dms/

# Test results:
deploy/docker/test-results/

# Logs:
deploy/docker/logs/
```

#### Configuration Files

Each service has its configuration in:
```
deploy/docker/configs/
├── orchestrator/
├── vnf-operator/
├── o2-client/
├── tn-manager/
├── tn-agent/
├── prometheus/
└── grafana/
```

### Kubernetes Deployment

#### Cluster Configuration

The Kind cluster is configured with:
- Single control-plane node
- Kube-OVN CNI for advanced networking
- Port mappings for all services
- Pod security standards (restricted)
- Network policies for security

#### Prerequisites Installation

```bash
# Install Kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

#### Detailed Deployment Steps

1. **Create Kind cluster**:
   ```bash
   kind create cluster --name oran-mano \
     --config deploy/kind/kind-cluster-config.yaml \
     --wait 5m
   ```

2. **Install Kube-OVN CNI**:
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/kubeovn/kube-ovn/v1.12.0/dist/images/install.sh
   ```

3. **Setup namespaces and RBAC**:
   ```bash
   kubectl apply -f deploy/k8s/base/namespace.yaml
   kubectl apply -f deploy/k8s/base/rbac.yaml
   ```

4. **Install monitoring stack**:
   ```bash
   helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
   helm install prometheus prometheus-community/kube-prometheus-stack \
     --namespace oran-monitoring --create-namespace \
     --set grafana.adminPassword=admin123
   ```

5. **Build and load images**:
   ```bash
   # Build all service images
   for service in orchestrator vnf-operator o2-client tn-manager tn-agent ran-dms cn-dms; do
     docker build -t oran-$service:latest \
       -f deploy/docker/$service/Dockerfile .
     kind load docker-image oran-$service:latest --name oran-mano
   done
   ```

6. **Deploy services**:
   ```bash
   kubectl apply -f deploy/k8s/base/
   ```

7. **Wait for deployment**:
   ```bash
   kubectl wait --for=condition=available deployment \
     -l app.kubernetes.io/component=orchestrator \
     -n oran-mano --timeout=300s
   ```

#### Service Access

Access services through port-forwarding:

```bash
# Orchestrator
kubectl port-forward -n oran-mano svc/oran-orchestrator 8080:8080

# Grafana
kubectl port-forward -n oran-monitoring svc/prometheus-grafana 3000:80

# Prometheus
kubectl port-forward -n oran-monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
```

## Configuration

### Environment Variables

#### Core Configuration
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `METRICS_ENABLED`: Enable metrics collection (true/false)
- `BUILD_TIME`: Build timestamp
- `VERSION`: Application version

#### Service Endpoints
- `DMS_ENDPOINT`: RAN/CN DMS endpoint
- `PORCH_ENDPOINT`: Porch GitOps server
- `O2IMS_ENDPOINT`: O-RAN O2 IMS endpoint
- `ORCHESTRATOR_ENDPOINT`: Orchestrator API endpoint

#### Performance Targets
- `TARGET_THROUGHPUT_EMBB`: eMBB throughput target (4.57 Mbps)
- `TARGET_THROUGHPUT_URLLC`: URLLC throughput target (2.77 Mbps)
- `TARGET_THROUGHPUT_MMTC`: mMTC throughput target (0.93 Mbps)
- `TARGET_RTT_EMBB`: eMBB RTT target (16.1 ms)
- `TARGET_RTT_URLLC`: URLLC RTT target (15.7 ms)
- `TARGET_RTT_MMTC`: mMTC RTT target (6.3 ms)
- `MAX_DEPLOYMENT_TIME`: Maximum deployment time (600 seconds)

### Security Configuration

#### TLS Certificates

Self-signed certificates are automatically generated:
```bash
# Certificate locations:
deploy/docker/certs/
├── ca.crt          # Certificate Authority
├── ca.key          # CA private key
├── server.crt      # Default server certificate
├── server.key      # Default server private key
└── webhook/
    ├── server.crt  # Webhook server certificate
    └── server.key  # Webhook server private key
```

#### Network Security

- Pod Security Standards: restricted
- Network policies for inter-service communication
- AppArmor profiles for container runtime security
- Seccomp profiles for syscall filtering
- Non-root containers with minimal privileges

## Monitoring and Observability

### Prometheus Metrics

Key metrics exposed:
- `http_requests_total` - HTTP request counts
- `http_request_duration_seconds` - Request latency
- `intent_processing_duration_seconds` - Intent processing time
- `vnf_deployment_duration_seconds` - VNF deployment time
- `network_slice_count` - Active network slices
- `tn_bandwidth_utilization` - Transport network utilization

### Grafana Dashboards

Pre-configured dashboards:
1. **O-RAN MANO Overview** - System health and performance
2. **Intent Processing** - Intent lifecycle and processing metrics
3. **Network Slices** - Slice deployment and performance
4. **Transport Network** - TN agent metrics and connectivity
5. **Resource Utilization** - CPU, memory, and storage usage

### Alerting Rules

Configured alerts:
- Service availability
- High latency (>500ms)
- Error rate (>10%)
- Resource utilization (>80%)
- Intent processing failures
- Network slice deployment failures

## Troubleshooting

### Common Issues

#### Docker Compose Issues

1. **Port conflicts**:
   ```bash
   # Check port usage
   netstat -tulpn | grep :8080

   # Modify docker-compose.local.yml ports if needed
   ```

2. **Image build failures**:
   ```bash
   # Clean Docker cache
   docker system prune -f
   docker builder prune -f

   # Rebuild with no cache
   docker-compose build --no-cache --parallel
   ```

3. **Service health check failures**:
   ```bash
   # Check service logs
   docker-compose logs orchestrator

   # Check network connectivity
   docker exec oran-orchestrator wget -qO- http://ran-dms:8080/health
   ```

#### Kubernetes Issues

1. **Cluster creation failures**:
   ```bash
   # Delete and recreate cluster
   kind delete cluster --name oran-mano
   kind create cluster --name oran-mano --config deploy/kind/kind-cluster-config.yaml
   ```

2. **Image pull failures**:
   ```bash
   # Verify images are loaded
   docker exec -it oran-mano-control-plane crictl images

   # Reload images if missing
   kind load docker-image oran-orchestrator:latest --name oran-mano
   ```

3. **Pod startup issues**:
   ```bash
   # Check pod status
   kubectl get pods -n oran-mano

   # Describe problematic pods
   kubectl describe pod -n oran-mano <pod-name>

   # Check logs
   kubectl logs -n oran-mano <pod-name>
   ```

### Log Locations

#### Docker Compose
- Service logs: `docker-compose logs <service>`
- Persistent logs: `deploy/docker/logs/`

#### Kubernetes
- Pod logs: `kubectl logs -n oran-mano <pod>`
- Node logs: `kubectl get events -n oran-mano`

### Health Check Commands

```bash
# Docker Compose
./deploy/scripts/deploy-local.sh health

# Kubernetes
kubectl get pods -n oran-mano
kubectl get svc -n oran-mano

# Manual health checks
curl http://localhost:8080/health  # Orchestrator
curl http://localhost:8087/health  # RAN DMS
curl http://localhost:8088/health  # CN DMS
```

### Performance Tuning

#### Docker Compose
```bash
# Increase container resources
# Edit docker-compose.local.yml:
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
```

#### Kubernetes
```bash
# Scale deployments
kubectl scale deployment oran-orchestrator --replicas=2 -n oran-mano

# Update resource requests/limits
kubectl patch deployment oran-orchestrator -n oran-mano -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "orchestrator",
          "resources": {
            "requests": {"cpu": "200m", "memory": "256Mi"},
            "limits": {"cpu": "1000m", "memory": "1Gi"}
          }
        }]
      }
    }
  }
}'
```

## Cleanup and Maintenance

### Docker Compose Cleanup

```bash
# Stop all services
./deploy/scripts/deploy-local.sh stop

# Complete cleanup (removes data)
./deploy/scripts/deploy-local.sh clean

# Prune Docker system
docker system prune -a -f
```

### Kubernetes Cleanup

```bash
# Delete cluster
kind delete cluster --name oran-mano

# Clean up Docker images
docker image prune -a -f
```

### Log Rotation

```bash
# Docker logs
docker system prune -f --filter "until=24h"

# Application logs
find deploy/docker/logs/ -name "*.log" -mtime +7 -delete
```

## Next Steps

After successful deployment:

1. **Run Tests**: Execute the comprehensive test suite
   ```bash
   ./deploy/testing/performance-test.sh
   ```

2. **Explore APIs**: Access the OpenAPI documentation
   - Orchestrator: http://localhost:8080/swagger-ui
   - RAN DMS: http://localhost:8087/docs

3. **Monitor System**: Access monitoring dashboards
   - Grafana: http://localhost:3000
   - Prometheus: http://localhost:9090

4. **Deploy Intents**: Submit network slice intents via API
   ```bash
   curl -X POST http://localhost:8080/api/v1/intents \
     -H "Content-Type: application/json" \
     -d '{"intent": {"type": "network-slice", "requirements": {...}}}'
   ```

For detailed testing procedures, see [TESTING_PROCEDURES.md](TESTING_PROCEDURES.md).

For health check details, see [HEALTH_CHECKS.md](HEALTH_CHECKS.md).