# O-RAN Intent-MANO Testing Procedures

## Overview

This document provides comprehensive testing procedures for the O-RAN Intent-MANO system, including unit tests, integration tests, end-to-end tests, performance validation, and security testing.

## Testing Strategy

### Test Pyramid

```
         /\
        /E2E\      <- Few, comprehensive scenarios
       /------\
      /Integr.\   <- Component integration
     /----------\
    /    Unit    \ <- Many, fast, isolated
   /--------------\
```

### Testing Levels

1. **Unit Tests** - Individual component testing
2. **Integration Tests** - Service-to-service interaction
3. **End-to-End Tests** - Complete workflow validation
4. **Performance Tests** - Throughput, latency, scalability
5. **Security Tests** - Vulnerability and compliance
6. **Chaos Tests** - Resilience and fault tolerance

## Prerequisites

### Test Environment Setup

```bash
# Install testing dependencies
sudo apt-get update
sudo apt-get install -y curl jq iperf3 netcat-openbsd bc

# Python testing tools
pip3 install pytest requests pyyaml numpy pandas

# Go testing tools (if running Go tests directly)
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install github.com/onsi/gomega/...@latest
```

### Environment Preparation

```bash
# Deploy the system first
cd /path/to/O-RAN-Intent-MANO-for-Network-Slicing

# Docker Compose deployment
./deploy/scripts/deploy-local.sh start

# OR Kubernetes deployment
./deploy/scripts/deploy-kubernetes.sh deploy
```

## Unit Testing

### Go Unit Tests

#### Running All Unit Tests

```bash
# From project root
make test-unit

# Or directly with go
go test -v ./...

# With coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Component-Specific Tests

```bash
# Orchestrator tests
cd orchestrator
go test -v ./pkg/...

# VNF Operator tests
cd adapters/vnf-operator
go test -v ./controllers/...

# TN Manager tests
cd tn/manager
go test -v ./pkg/...
```

#### Example Unit Test Execution

```bash
# Test orchestrator intent processing
cd orchestrator
go test -v ./pkg/intents -run TestIntentProcessing

# Expected output:
# === RUN   TestIntentProcessing
# === RUN   TestIntentProcessing/valid_embb_intent
# === RUN   TestIntentProcessing/valid_urllc_intent
# === RUN   TestIntentProcessing/invalid_intent
# --- PASS: TestIntentProcessing (0.05s)
#     --- PASS: TestIntentProcessing/valid_embb_intent (0.01s)
#     --- PASS: TestIntentProcessing/valid_urllc_intent (0.01s)
#     --- PASS: TestIntentProcessing/invalid_intent (0.01s)
# PASS
```

### Python Unit Tests

#### Running Python Tests

```bash
# From experiments directory
cd experiments
python -m pytest tests/ -v

# With coverage
python -m pytest tests/ -v --cov=. --cov-report=html
```

## Integration Testing

### Service Integration Tests

#### Test Service Health and Connectivity

```bash
# Run integration test script
cd deploy/testing
chmod +x integration-test.sh
./integration-test.sh

# Manual service connectivity test
curl -f http://localhost:8080/health  # Orchestrator
curl -f http://localhost:8087/health  # RAN DMS
curl -f http://localhost:8088/health  # CN DMS
curl -f http://localhost:8083/health  # O2 Client
curl -f http://localhost:8084/health  # TN Manager
curl -f http://localhost:8085/health  # TN Agent E01
curl -f http://localhost:8086/health  # TN Agent E02
```

#### API Integration Tests

```bash
# Test intent submission workflow
./deploy/testing/test-intent-workflow.sh

# Expected workflow:
# 1. Submit eMBB intent -> HTTP 201
# 2. Check intent processing -> Status: processing
# 3. Wait for completion -> Status: deployed
# 4. Verify network slice -> Active slice found
# 5. Test slice connectivity -> Connectivity OK
```

#### Database Integration Tests

```bash
# Test DMS data persistence
./deploy/testing/test-dms-persistence.sh

# Tests:
# - RAN DMS data storage and retrieval
# - CN DMS configuration persistence
# - Intent state management
# - VNF lifecycle data consistency
```

### Inter-Service Communication Tests

```bash
# Test orchestrator -> DMS communication
docker exec oran-orchestrator curl -f http://ran-dms:8080/api/v1/nodes

# Test VNF operator -> DMS communication
docker exec oran-vnf-operator curl -f http://ran-dms:8080/api/v1/vnfs

# Test TN agents -> TN manager communication
docker exec oran-tn-agent-edge01 curl -f http://tn-manager:8080/api/v1/topology

# Test O2 client -> DMS sync
docker exec oran-o2-client curl -f http://ran-dms:8080/api/v1/inventory
```

## End-to-End Testing

### Complete Workflow Tests

#### Network Slice Deployment Test

```bash
#!/bin/bash
# E2E test for complete network slice deployment

# 1. Submit eMBB slice intent
INTENT_ID=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "intent": {
      "type": "network-slice",
      "slice_type": "eMBB",
      "requirements": {
        "throughput": "5Mbps",
        "latency": "20ms",
        "coverage_area": ["site01", "site02"]
      }
    }
  }' | jq -r '.intent_id')

echo "Submitted intent: $INTENT_ID"

# 2. Monitor deployment progress
for i in {1..120}; do
  STATUS=$(curl -s http://localhost:8080/api/v1/intents/$INTENT_ID | jq -r '.status')
  echo "Status: $STATUS"

  if [[ "$STATUS" == "deployed" ]]; then
    echo "✓ Intent deployed successfully"
    break
  elif [[ "$STATUS" == "failed" ]]; then
    echo "✗ Intent deployment failed"
    exit 1
  fi

  sleep 5
done

# 3. Validate network slice
SLICE_ID=$(curl -s http://localhost:8080/api/v1/intents/$INTENT_ID | jq -r '.slice_id')
SLICE_STATUS=$(curl -s http://localhost:8080/api/v1/slices/$SLICE_ID | jq -r '.status')

if [[ "$SLICE_STATUS" == "active" ]]; then
  echo "✓ Network slice is active"
else
  echo "✗ Network slice not active: $SLICE_STATUS"
  exit 1
fi

# 4. Test slice connectivity
iperf3 -c localhost -p 5201 -t 10 -f m > /tmp/throughput_test.txt
THROUGHPUT=$(grep 'receiver' /tmp/throughput_test.txt | awk '{print $7}')

echo "Measured throughput: ${THROUGHPUT} Mbps"

if (( $(echo "$THROUGHPUT >= 4.0" | bc -l) )); then
  echo "✓ Throughput test passed"
else
  echo "✗ Throughput test failed: expected >= 4.0 Mbps, got $THROUGHPUT Mbps"
fi

echo "E2E test completed successfully"
```

#### Multi-Slice Orchestration Test

```bash
# Test concurrent slice deployment
./deploy/testing/test-multi-slice.sh

# Tests:
# 1. Deploy 3 concurrent slices (eMBB, URLLC, mMTC)
# 2. Verify resource allocation and isolation
# 3. Test slice-specific QoS parameters
# 4. Validate inter-slice isolation
# 5. Test slice lifecycle (update, delete)
```

### GitOps Integration Test

```bash
# Test Porch integration and GitOps workflow
./deploy/testing/test-gitops.sh

# Workflow:
# 1. Intent submission triggers package generation
# 2. VNF operator creates Porch packages
# 3. Packages are committed to Git repository
# 4. ConfigSync deploys manifests to target clusters
# 5. Verify deployment across multiple clusters
```

## Performance Testing

### Comprehensive Performance Test Suite

```bash
# Run complete performance test suite
cd deploy/testing
chmod +x performance-test.sh
./performance-test.sh

# Test results saved to: deploy/results/performance_test_<timestamp>/
```

### Individual Performance Tests

#### Throughput Testing

```bash
# eMBB throughput test (target: 4.57 Mbps)
iperf3 -c localhost -p 5201 -t 60 -f m -J > embb_throughput.json

# URLLC throughput test (target: 2.77 Mbps)
iperf3 -c localhost -p 5202 -t 60 -f m -J > urllc_throughput.json

# mMTC throughput test (target: 0.93 Mbps)
iperf3 -c localhost -p 5201 -t 60 -b 1M -f m -J > mmtc_throughput.json

# Analyze results
python3 -c "
import json
import sys

for test_type, file in [('eMBB', 'embb_throughput.json'), ('URLLC', 'urllc_throughput.json'), ('mMTC', 'mmtc_throughput.json')]:
    try:
        with open(file) as f:
            data = json.load(f)
        throughput = data['end']['sum_received']['bits_per_second'] / 1e6
        print(f'{test_type}: {throughput:.2f} Mbps')
    except:
        print(f'{test_type}: Test failed')
"
```

#### Latency Testing

```bash
# RTT measurements
ping -c 100 localhost > ping_results.txt

# Extract RTT statistics
RTT_AVG=$(grep 'rtt min/avg/max/mdev' ping_results.txt | cut -d'/' -f5)
echo "Average RTT: ${RTT_AVG}ms"

# Target comparisons:
# eMBB: ${RTT_AVG} <= 16.1ms
# URLLC: ${RTT_AVG} <= 15.7ms
# mMTC: ${RTT_AVG} <= 6.3ms
```

#### Deployment Time Testing

```bash
# Test E2E deployment time (target: <10 minutes)
start_time=$(date +%s)

# Submit intent and wait for completion
INTENT_ID=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"intent":{"type":"network-slice","slice_type":"eMBB"}}' | jq -r '.intent_id')

# Poll until deployed
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/intents/$INTENT_ID | jq -r '.status')
  if [[ "$STATUS" == "deployed" ]]; then
    break
  elif [[ "$STATUS" == "failed" ]]; then
    echo "Deployment failed"
    exit 1
  fi
  sleep 5
done

end_time=$(date +%s)
deployment_time=$((end_time - start_time))

echo "Deployment time: ${deployment_time}s (target: ≤600s)"

if [[ $deployment_time -le 600 ]]; then
  echo "✓ Deployment time test PASSED"
else
  echo "✗ Deployment time test FAILED"
fi
```

#### Load Testing

```bash
# Concurrent intent submission test
./deploy/testing/load-test.sh

# Parameters:
# - Concurrent users: 10
# - Test duration: 300s
# - Intent types: eMBB, URLLC, mMTC
# - Success rate target: >95%
```

### Performance Monitoring

```bash
# Monitor system resources during tests
./deploy/testing/monitor-resources.sh &
MONITOR_PID=$!

# Run performance tests
./deploy/testing/performance-test.sh

# Stop monitoring
kill $MONITOR_PID

# Generate performance report
./deploy/testing/generate-perf-report.sh
```

## Security Testing

### Container Security Scan

```bash
# Trivy security scanning
trivy image oran-orchestrator:latest
trivy image oran-vnf-operator:latest
trivy image oran-ran-dms:latest
trivy image oran-cn-dms:latest

# Generate security report
trivy image --format json oran-orchestrator:latest > security-scan-orchestrator.json
```

### Network Security Testing

```bash
# Test network policies
kubectl exec -n oran-mano deployment/oran-orchestrator -- nc -zv oran-ran-dms 8080
kubectl exec -n oran-mano deployment/oran-orchestrator -- nc -zv oran-cn-dms 8080

# Should fail (blocked by network policy):
kubectl exec -n oran-edge deployment/oran-tn-agent-edge01 -- nc -zv oran-orchestrator 8080
```

### API Security Testing

```bash
# Test authentication and authorization
# (Assuming authentication is implemented)

# Test without token - should fail
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"intent":{}}' \
  -w "\nHTTP Status: %{http_code}\n"

# Test with invalid token - should fail
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{"intent":{}}' \
  -w "\nHTTP Status: %{http_code}\n"

# Test input validation
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"intent":{"type":"<script>alert(\"xss\")</script>"}}' \
  -w "\nHTTP Status: %{http_code}\n"
```

### Compliance Testing

```bash
# Run compliance checks
./deploy/testing/compliance-test.sh

# Tests:
# - Pod Security Standards compliance
# - Network policy enforcement
# - RBAC configuration validation
# - Secret management verification
# - Resource limit enforcement
```

## Chaos Testing

### Service Resilience Tests

```bash
# Chaos testing framework
./deploy/testing/chaos-test.sh

# Tests:
# 1. Random service restarts
# 2. Network partitioning
# 3. Resource exhaustion
# 4. Disk space limitations
# 5. Memory pressure
```

#### Service Failure Simulation

```bash
# Stop orchestrator and verify recovery
docker-compose stop orchestrator
sleep 30
docker-compose start orchestrator

# Wait for health check
timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done'

# Verify system recovery
curl http://localhost:8080/api/v1/status
```

#### Network Partition Simulation

```bash
# Simulate network partition between orchestrator and RAN DMS
docker exec oran-orchestrator iptables -A OUTPUT -d ran-dms -j DROP

# Wait and observe system behavior
sleep 60

# Restore connectivity
docker exec oran-orchestrator iptables -D OUTPUT -d ran-dms -j DROP

# Verify recovery
curl http://localhost:8080/api/v1/status
```

## Test Automation

### Continuous Integration Tests

```yaml
# .github/workflows/test.yml
name: Comprehensive Testing
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24.7'
      - run: make test-unit

  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - run: |
          ./deploy/scripts/deploy-local.sh start
          ./deploy/testing/integration-test.sh

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - run: |
          ./deploy/scripts/deploy-kubernetes.sh deploy
          ./deploy/testing/e2e-test.sh

  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - run: |
          ./deploy/scripts/deploy-local.sh start
          ./deploy/testing/performance-test.sh
```

### Test Results Reporting

```bash
# Generate comprehensive test report
./deploy/testing/generate-test-report.sh

# Output: deploy/results/test-report-<timestamp>.html
```

## Test Data Management

### Test Data Setup

```bash
# Create test data sets
./deploy/testing/setup-test-data.sh

# Test data includes:
# - Sample intents for each slice type
# - VNF descriptors and packages
# - Network topology configurations
# - Performance baseline data
```

### Test Environment Cleanup

```bash
# Clean test data
./deploy/testing/cleanup-test-data.sh

# Reset services to clean state
./deploy/scripts/deploy-local.sh restart
```

## Troubleshooting Test Issues

### Common Test Failures

#### Service Connectivity Issues

```bash
# Check service health
docker-compose ps

# Check network connectivity
docker exec oran-orchestrator ping ran-dms

# Check port accessibility
docker exec oran-orchestrator nc -zv ran-dms 8080
```

#### Performance Test Failures

```bash
# Check system resources
docker stats

# Verify network configuration
docker network ls
docker network inspect deploy_mano-net

# Check iPerf3 server status
docker exec oran-tn-agent-edge01 ss -tlnp | grep 5201
```

#### Timeout Issues

```bash
# Increase timeout values
export TEST_TIMEOUT=300
export HEALTH_CHECK_TIMEOUT=60

# Check service startup time
docker-compose logs orchestrator | grep "Server started"
```

### Log Analysis

```bash
# Collect logs for analysis
./deploy/testing/collect-logs.sh

# Logs are saved to: deploy/logs/test-logs-<timestamp>.tar.gz

# Extract and analyze
tar -xzf deploy/logs/test-logs-*.tar.gz
grep -r "ERROR" extracted-logs/
grep -r "FAILED" extracted-logs/
```

## Test Metrics and KPIs

### Key Performance Indicators

| Metric | Target | Measurement |
|--------|---------|-------------|
| Unit Test Coverage | >80% | Go test coverage |
| Integration Test Pass Rate | >95% | CI pipeline |
| E2E Test Success Rate | >90% | Full workflow tests |
| eMBB Throughput | ≥4.57 Mbps | iPerf3 measurement |
| URLLC Throughput | ≥2.77 Mbps | iPerf3 measurement |
| mMTC Throughput | ≥0.93 Mbps | iPerf3 measurement |
| eMBB RTT | ≤16.1ms | Ping measurement |
| URLLC RTT | ≤15.7ms | Ping measurement |
| mMTC RTT | ≤6.3ms | Ping measurement |
| Deployment Time | ≤600s | E2E workflow |
| Service Availability | >99% | Health check uptime |
| Security Scan | 0 High/Critical | Trivy scan results |

### Test Reporting

```bash
# Generate metrics dashboard
./deploy/testing/generate-metrics.sh

# Metrics are available at:
# - Grafana: http://localhost:3000/d/test-metrics
# - JSON report: deploy/results/test-metrics.json
# - HTML report: deploy/results/test-report.html
```

This comprehensive testing strategy ensures the O-RAN Intent-MANO system meets all functional, performance, and security requirements while maintaining high reliability and availability standards.