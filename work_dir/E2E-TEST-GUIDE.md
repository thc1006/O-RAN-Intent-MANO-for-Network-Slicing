# O-RAN Intent MANO - End-to-End Testing Guide

## Overview

This guide walks through deploying and testing the complete O-RAN Intent MANO system using Docker and K3s.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Docker Network                          │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │  K3s Server  │───▶│    Porch     │◀───│    Gitea     │  │
│  │  (Kubernetes)│    │  (GitOps)    │    │  (Git Repo)  │  │
│  └──────┬───────┘    └──────────────┘    └──────────────┘  │
│         │                                                    │
│         ▼                                                    │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ Orchestrator │───▶│   Nephio     │───▶│  E2E Tests   │  │
│  │  (Intent)    │    │  Generator   │    │   (Runner)   │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. K3s Server (Kubernetes)
- Lightweight Kubernetes cluster
- Runs in privileged Docker container
- Exposes API on port 6443
- Hosts Porch and applications

### 2. Porch (Package Orchestration)
- Nephio's GitOps engine
- Manages PackageRevisions
- Integrates with Git repositories
- API exposed on port 8080

### 3. Gitea (Git Server)
- Hosts Nephio package repository
- Stores rendered packages
- Web UI on port 3000

### 4. O-RAN Orchestrator
- Receives intent requests
- Performs QoS mapping
- Triggers package generation
- API on port 8081

### 5. Nephio Generator
- Renders Kpt packages
- Executes function pipelines
- Runs Kustomize builds
- Pushes to Porch

### 6. E2E Test Runner
- Submits test intents
- Validates package creation
- Checks deployment status
- Generates test reports

## Prerequisites

### Required Software
```bash
# Docker and Docker Compose
docker --version  # 20.10+
docker compose version  # 2.0+

# Optional (for local testing)
kubectl  # 1.28+
k9s  # For cluster visualization
```

### System Requirements
- **RAM**: 8GB minimum (16GB recommended)
- **CPU**: 4 cores minimum
- **Disk**: 20GB free space
- **OS**: Windows 10/11, Linux, macOS

## Quick Start

### 1. Start the Environment

```bash
# Navigate to work_dir
cd work_dir

# Start all services
docker compose -f docker-compose.e2e.yml up -d

# Check status
docker compose -f docker-compose.e2e.yml ps
```

Expected output:
```
NAME                STATUS              PORTS
o-ran-k3s           Up (healthy)        6443, 8080, 30000-30100
porch-installer     Up                  -
o-ran-orchestrator  Up                  8081
nephio-generator    Up                  8082
o-ran-gitea         Up                  3000, 2222
e2e-test            Up                  -
```

### 2. Wait for Services to Initialize

```bash
# Watch logs (in separate terminals)
docker logs -f o-ran-k3s
docker logs -f porch-installer
docker logs -f o-ran-orchestrator
```

⏳ **Initialization Time**: ~2-3 minutes

### 3. Verify K3s and Porch

```bash
# Export kubeconfig
export KUBECONFIG=./kubeconfig/kubeconfig.yaml

# Check nodes
kubectl get nodes

# Check Porch
kubectl get pods -n porch-system
kubectl get deployments -n porch-system
```

Expected output:
```
NAME                           READY   STATUS    RESTARTS   AGE
porch-server-xxxx-xxxx         1/1     Running   0          2m
```

### 4. Run E2E Test

```bash
# Execute test script inside K3s container
docker exec o-ran-k3s /scripts/test-deployment.sh
```

Or run manually:

```bash
# Submit test intent
curl -X POST http://localhost:8081/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "intent_id": "test-slice-001",
    "service_type": "eMBB",
    "coverage_area": "zone-001",
    "latency_ms": 10,
    "throughput_mbps": 1000,
    "availability": 99.99
  }'

# Wait for processing (30-60 seconds)
sleep 30

# Check PackageRevisions
kubectl get packagerevisions -A

# Get package details
kubectl get packagerevision <name> -n default -o yaml
```

## Validation Checkpoints

### ✅ Checkpoint 1: K3s Cluster
```bash
kubectl get nodes
# Expected: 1 node in Ready state

kubectl get pods -A
# Expected: All system pods Running
```

### ✅ Checkpoint 2: Porch Installation
```bash
kubectl get crds | grep porch
# Expected:
# packagerevisions.porch.kpt.dev
# repositories.porch.kpt.dev

kubectl get pods -n porch-system
# Expected: porch-server Running
```

### ✅ Checkpoint 3: Gitea Repository
```bash
curl http://localhost:3000
# Expected: Gitea web UI loads

# Check repository
curl http://localhost:3000/api/v1/repos/nephio/packages
# Expected: Repository details
```

### ✅ Checkpoint 4: Orchestrator
```bash
curl http://localhost:8081/health
# Expected: {"status": "healthy"}

docker logs o-ran-orchestrator | grep "Started"
# Expected: "Orchestrator started successfully"
```

### ✅ Checkpoint 5: Nephio Generator
```bash
docker logs nephio-generator | grep "Renderer initialized"
# Expected: Success message
```

### ✅ Checkpoint 6: Intent Processing
```bash
# Submit intent (see above)

# Check logs
docker logs o-ran-orchestrator | grep "test-slice-001"
# Expected: Intent received and processed

docker logs nephio-generator | grep "test-slice-001"
# Expected: Package generated
```

### ✅ Checkpoint 7: Package Creation
```bash
kubectl get packagerevisions -A
# Expected: PackageRevision created

kubectl get packagerevision <name> -n default -o yaml
# Expected: Full package with resources
```

## Test Scenarios

### Scenario 1: eMBB Slice
```json
{
  "intent_id": "embb-test-001",
  "service_type": "eMBB",
  "latency_ms": 10,
  "throughput_mbps": 1000,
  "availability": 99.99
}
```

**Expected Results**:
- ✅ ConfigMap created with QoS mapping
- ✅ PackageRevision created
- ✅ Resources include: Deployment, Service, NetworkPolicy

### Scenario 2: URLLC Slice
```json
{
  "intent_id": "urllc-test-001",
  "service_type": "URLLC",
  "latency_ms": 1,
  "throughput_mbps": 100,
  "availability": 99.999
}
```

**Expected Results**:
- ✅ Ultra-low latency configuration
- ✅ High availability settings
- ✅ Priority scheduling policies

### Scenario 3: mMTC Slice
```json
{
  "intent_id": "mmtc-test-001",
  "service_type": "mMTC",
  "latency_ms": 1000,
  "throughput_mbps": 1,
  "availability": 99.9
}
```

**Expected Results**:
- ✅ High connection density configuration
- ✅ Low bandwidth per device
- ✅ Resource-efficient deployment

## Troubleshooting

### Issue 1: K3s Not Starting

**Symptoms**:
```bash
docker logs o-ran-k3s
# Error: Failed to start k3s
```

**Solution**:
```bash
# Check privileged mode
docker inspect o-ran-k3s | grep Privileged
# Should be: "Privileged": true

# Restart with clean slate
docker compose -f docker-compose.e2e.yml down -v
docker compose -f docker-compose.e2e.yml up -d
```

### Issue 2: Porch Installation Fails

**Symptoms**:
```bash
kubectl get pods -n porch-system
# CrashLoopBackOff or ImagePullBackOff
```

**Solution**:
```bash
# Check CRDs
kubectl get crds | grep porch

# Reinstall
kubectl delete -f ./porch-config/porch-deployment.yaml
kubectl apply -f ./porch-config/porch-deployment.yaml

# Check logs
kubectl logs -n porch-system -l app=porch-server
```

### Issue 3: Orchestrator Can't Connect to Porch

**Symptoms**:
```bash
docker logs o-ran-orchestrator
# Error: connection refused to k3s-server:8080
```

**Solution**:
```bash
# Check network connectivity
docker exec o-ran-orchestrator curl http://k3s-server:8080

# Check Porch service
kubectl get svc -n porch-system

# Verify NodePort
kubectl get svc porch-server -n porch-system -o yaml | grep nodePort
```

### Issue 4: No PackageRevision Created

**Symptoms**:
```bash
kubectl get packagerevisions -A
# No resources found
```

**Solution**:
```bash
# Check orchestrator logs
docker logs o-ran-orchestrator | grep ERROR

# Check nephio-generator logs
docker logs nephio-generator | grep ERROR

# Verify Porch repository
kubectl get repositories -A

# Check intent submission
curl http://localhost:8081/api/v1/intents/<intent-id>
```

### Issue 5: Gitea Not Accessible

**Symptoms**:
```bash
curl http://localhost:3000
# Connection refused
```

**Solution**:
```bash
# Check Gitea status
docker logs o-ran-gitea

# Restart Gitea
docker restart o-ran-gitea

# Wait for initialization
sleep 30
```

## Cleanup

### Stop All Services
```bash
cd work_dir
docker compose -f docker-compose.e2e.yml down
```

### Remove Volumes (Clean Slate)
```bash
docker compose -f docker-compose.e2e.yml down -v
```

### Remove Images
```bash
docker rmi $(docker images -q "work_dir*")
```

## Advanced Testing

### Run Specific Test Suite
```bash
docker exec e2e-test go test -v ./e2e -run TestIntentProcessing
docker exec e2e-test go test -v ./e2e -run TestPackageRendering
docker exec e2e-test go test -v ./e2e -run TestGitOpsDeployment
```

### Enable Debug Logging
```bash
# Edit docker-compose.e2e.yml
# Change LOG_LEVEL=debug for all services

docker compose -f docker-compose.e2e.yml up -d --force-recreate
```

### Performance Testing
```bash
# Submit multiple intents
for i in {1..10}; do
  curl -X POST http://localhost:8081/api/v1/intents \
    -H "Content-Type: application/json" \
    -d "{\"intent_id\": \"load-test-$i\", \"service_type\": \"eMBB\", \"latency_ms\": 10}"
done

# Monitor processing
watch -n 1 'kubectl get packagerevisions -A | wc -l'
```

### Export Test Results
```bash
# Copy results from container
docker cp e2e-test:/test-results ./test-results

# View results
cat ./test-results/e2e-results.log
```

## Success Criteria

### ✅ All Tests Pass When:

1. **K3s Cluster**: 1 node Ready
2. **Porch**: Deployed and API responding
3. **Gitea**: Repository accessible
4. **Orchestrator**: Receives and processes intents
5. **Nephio Generator**: Renders packages successfully
6. **PackageRevision**: Created with all resources
7. **GitOps**: Package pushed to Git repository
8. **No Errors**: Clean logs across all components

### 📊 Expected Metrics:

- **Intent to Package Time**: < 30 seconds
- **Package Rendering Time**: < 10 seconds
- **GitOps Push Time**: < 5 seconds
- **Total E2E Time**: < 60 seconds

## Next Steps

After successful E2E testing:

1. **Production Deployment**: Deploy to real Kubernetes cluster
2. **Integration Testing**: Connect to real O2 endpoints
3. **Performance Tuning**: Optimize for scale
4. **Monitoring**: Add Prometheus/Grafana
5. **CI/CD**: Integrate with GitHub Actions

## Resources

- **Nephio Docs**: https://nephio.org/docs
- **Porch Guide**: https://github.com/nephio-project/porch
- **K3s Docs**: https://k3s.io
- **Kpt Functions**: https://kpt.dev/book/04-using-functions/

---

**Document Version**: 1.0
**Last Updated**: 2025-09-30
**Status**: ✅ Ready for Testing