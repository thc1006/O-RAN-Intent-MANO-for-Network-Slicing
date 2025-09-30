# End-to-End Integration Testing Guide

## Overview

This directory contains comprehensive end-to-end (E2E) integration tests for the O-RAN Intent MANO system. These tests validate the complete workflow from natural language intent to operational network slice deployment.

## Test Architecture

### Test Flow

```
User Intent (NLP)
    ↓
Claude AI Analysis
    ↓
Orchestrator Placement Calculation
    ↓
Nephio Package Generation
    ↓
VNF Deployment (Kubernetes)
    ↓
Transport Network Configuration (VXLAN)
    ↓
Network Slice Verification
```

## Directory Structure

```
tests/integration/
├── e2e/                           # End-to-end test files
│   ├── slice_provisioning_test.go # Main E2E test suite
│   ├── helpers.go                 # Test helper functions
│   └── types.go                   # Test data structures
├── mocks/                         # Mock service configurations
│   ├── orchestrator-expectations.json
│   ├── vnf-operator-expectations.json
│   ├── nephio-expectations.json
│   ├── tn-agent-expectations.json
│   ├── mockserver.properties
│   └── prometheus.yml
├── testdata/                      # Test fixtures and data
├── helpers/                       # Test utility tools
├── docker-compose.e2e.yml         # E2E test environment
├── Dockerfile.test                # Test container image
└── README.md                      # This file
```

## Prerequisites

### Required Tools

- **Go** 1.21 or later
- **Docker** 20.10 or later
- **Docker Compose** 2.0 or later
- **kubectl** (for Kubernetes integration)
- **make** (optional, for convenience commands)

### Environment Setup

1. **Install Dependencies**
   ```bash
   go mod download
   ```

2. **Verify Docker**
   ```bash
   docker --version
   docker-compose --version
   ```

3. **Configure Kubernetes** (optional for full integration)
   ```bash
   kubectl cluster-info
   ```

## Running E2E Tests

### Quick Start

```bash
# Run all E2E tests with Docker Compose
cd tests/integration
docker-compose -f docker-compose.e2e.yml up --build --abort-on-container-exit

# View test results
docker-compose -f docker-compose.e2e.yml logs e2e-test
```

### Run Specific Test Suites

```bash
# Run only the main provisioning test
docker-compose -f docker-compose.e2e.yml run e2e-test \
  go test -v -timeout=10m ./tests/integration/e2e/ \
  -run TestEndToEndSliceProvisioning

# Run error handling tests
docker-compose -f docker-compose.e2e.yml run e2e-test \
  go test -v -timeout=10m ./tests/integration/e2e/ \
  -run TestEndToEndSliceProvisioningErrorHandling

# Run edge case tests
docker-compose -f docker-compose.e2e.yml run e2e-test \
  go test -v -timeout=10m ./tests/integration/e2e/ \
  -run TestEndToEndSliceProvisioningEdgeCases

# Run component integration tests
docker-compose -f docker-compose.e2e.yml run e2e-test \
  go test -v -timeout=10m ./tests/integration/e2e/ \
  -run TestIntegrationWithMockedComponents
```

### Run Tests Locally (Without Docker)

```bash
# Set environment variables
export ORCHESTRATOR_URL=http://localhost:8080
export VNF_OPERATOR_URL=http://localhost:8081
export NEPHIO_RENDERER_URL=http://localhost:8082
export TN_AGENT_URL=http://localhost:8083

# Run tests
go test -v -timeout=10m -tags=integration ./tests/integration/e2e/...

# Run with coverage
go test -v -timeout=10m -tags=integration -coverprofile=coverage.out ./tests/integration/e2e/...
go tool cover -html=coverage.out -o coverage.html
```

### Run Tests in Short Mode (Skip Integration Tests)

```bash
go test -short ./tests/integration/e2e/...
```

## Test Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_MODE` | Test execution mode | `e2e` |
| `GO_TEST_TIMEOUT` | Test timeout duration | `10m` |
| `TEST_PARALLEL` | Number of parallel tests | `4` |
| `ORCHESTRATOR_URL` | Orchestrator service URL | `http://localhost:8080` |
| `VNF_OPERATOR_URL` | VNF Operator service URL | `http://localhost:8081` |
| `NEPHIO_RENDERER_URL` | Nephio Renderer service URL | `http://localhost:8082` |
| `TN_AGENT_URL` | TN Agent service URL | `http://localhost:8083` |
| `KUBECONFIG` | Kubernetes config path | `~/.kube/config` |
| `K8S_NAMESPACE` | Kubernetes namespace | `test-slice` |
| `LOG_LEVEL` | Logging level | `info` |
| `LOG_FORMAT` | Log format (json/text) | `json` |

### Mock Server Configuration

Mock servers use [MockServer](https://www.mock-server.com/) to simulate backend services. Configuration files are located in `tests/integration/mocks/`.

**Mock Services:**
- **Orchestrator**: Port 8080 (mocked on 1080)
- **VNF Operator**: Port 8081 (mocked on 1080)
- **Nephio Renderer**: Port 8082 (mocked on 1080)
- **TN Agent**: Port 8083 (mocked on 1080)

## Test Scenarios

### 1. Happy Path: Complete Slice Provisioning

Tests the full workflow from intent to operational slice:
- NLP intent parsing
- Claude AI analysis
- Resource placement calculation
- Nephio package generation
- VNF deployment
- Transport network configuration
- Slice verification

**Expected Result**: Network slice successfully provisioned with all QoS requirements met.

### 2. Error Handling Tests

Tests system behavior under error conditions:
- Invalid natural language intent
- Insufficient resources for placement
- VNF deployment failures
- Network configuration errors
- Rollback scenarios

**Expected Result**: Graceful error handling with appropriate rollback.

### 3. Edge Case Tests

Tests boundary conditions:
- Minimal resource requirements
- Maximum resource requirements
- Concurrent slice deployments
- Resource contention scenarios

**Expected Result**: System handles edge cases correctly.

### 4. Component Integration Tests

Tests individual component integrations:
- Nephio-O2 client integration
- Orchestrator-VNF operator integration
- TN Agent-VXLAN integration

**Expected Result**: All component integrations work correctly.

## Test Results

### Viewing Test Output

```bash
# Follow test logs in real-time
docker-compose -f docker-compose.e2e.yml logs -f e2e-test

# View test coverage
docker-compose -f docker-compose.e2e.yml run e2e-test \
  go tool cover -html=/output/coverage.out
```

### Test Metrics

Tests collect the following metrics:
- **Duration**: Total test execution time
- **Success Rate**: Percentage of successful tests
- **Component Response Times**: Individual service latencies
- **Resource Usage**: CPU, memory, network bandwidth

### Continuous Integration

Tests can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v2
      - name: Run E2E Tests
        run: |
          cd tests/integration
          docker-compose -f docker-compose.e2e.yml up --build --abort-on-container-exit
      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: tests/integration/output/
```

## Troubleshooting

### Common Issues

1. **Mock servers not responding**
   ```bash
   # Check mock server health
   curl http://localhost:8080/mockserver/status

   # Restart mock servers
   docker-compose -f docker-compose.e2e.yml restart mock-orchestrator
   ```

2. **Tests timing out**
   ```bash
   # Increase timeout
   export GO_TEST_TIMEOUT=20m

   # Or edit docker-compose.e2e.yml
   ```

3. **Port conflicts**
   ```bash
   # Check ports in use
   netstat -tuln | grep -E '8080|8081|8082|8083'

   # Stop conflicting services or change ports in docker-compose.e2e.yml
   ```

4. **Database connection errors**
   ```bash
   # Check PostgreSQL status
   docker-compose -f docker-compose.e2e.yml ps postgres-test

   # View database logs
   docker-compose -f docker-compose.e2e.yml logs postgres-test
   ```

### Debugging Tests

```bash
# Run tests with verbose output
go test -v -timeout=10m ./tests/integration/e2e/...

# Run specific test with debugging
go test -v -run TestEndToEndSliceProvisioning ./tests/integration/e2e/...

# Enable debug logging
export LOG_LEVEL=debug

# Attach to running container
docker exec -it e2e-test-runner /bin/bash
```

### Cleaning Up

```bash
# Stop all test services
docker-compose -f docker-compose.e2e.yml down

# Remove volumes
docker-compose -f docker-compose.e2e.yml down -v

# Remove test artifacts
rm -rf output/
```

## Best Practices

### Test-Driven Development (TDD)

1. **Write tests first**: Define expected behavior before implementation
2. **Red-Green-Refactor**: Follow TDD cycle strictly
3. **Maintain high coverage**: Target >90% code coverage
4. **Test all scenarios**: Happy path, errors, edge cases

### Test Organization

1. **Use test suites**: Group related tests with testify/suite
2. **Isolate tests**: Each test should be independent
3. **Clean up resources**: Use setup/teardown properly
4. **Mock external dependencies**: Use mock servers for external services

### Performance

1. **Parallel execution**: Run independent tests in parallel
2. **Optimize fixtures**: Use minimal test data
3. **Cache dependencies**: Reuse Docker layers
4. **Monitor resources**: Track CPU, memory usage

## Contributing

### Adding New Tests

1. Create test file in `tests/integration/e2e/`
2. Implement test using testify/suite pattern
3. Add mock expectations in `tests/integration/mocks/`
4. Update this README with test description
5. Ensure tests pass locally before submitting PR

### Updating Mock Expectations

1. Edit JSON files in `tests/integration/mocks/`
2. Follow MockServer expectation format
3. Test mock responses manually
4. Update related tests

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Suite](https://pkg.go.dev/github.com/stretchr/testify/suite)
- [MockServer Documentation](https://www.mock-server.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [O-RAN Architecture](https://www.o-ran.org/)
- [Nephio Documentation](https://nephio.org/)

## Support

For questions or issues:
- Open an issue in the project repository
- Contact the development team
- Review project documentation

---

**Last Updated**: 2025-09-30
**Test Framework Version**: 1.0.0
**Go Version**: 1.21+