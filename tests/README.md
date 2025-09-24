# O-RAN Intent-MANO Testing Infrastructure

## Overview

This comprehensive testing framework provides end-to-end validation for the O-RAN Intent-Based MANO system. The framework is designed to validate all thesis requirements, including deployment times (<10 minutes), throughput targets (4.57/2.77/0.93 Mbps), and RTT latencies (16.1/15.7/6.3 ms).

## Architecture

```
tests/
├── cmd/
│   └── test-runner/           # Test orchestration engine
│       └── main.go
├── integration/               # Cross-service integration tests
│   ├── orchestrator_integration_test.go
│   ├── ran_dms_integration_test.go
│   ├── cn_dms_integration_test.go
│   └── vnf_operator_integration_test.go
├── performance/               # Performance and thesis validation
│   ├── thesis_metrics_validation_test.go
│   ├── benchmark_test.go
│   └── network_performance_test.go
├── e2e/                      # End-to-end workflow tests
│   ├── complete_intent_flow_test.go
│   └── deployment_timing_test.go
├── healthcheck/              # Service health monitoring
│   └── services_health_test.go
├── security/                 # Security validation tests
├── utils/                    # Test utilities and helpers
│   ├── test_helpers.go
│   └── metrics.go
├── mocks/                    # Mock API types and services
│   └── api_types.go
├── go.mod                    # Go module configuration
├── test.config.yaml          # Test configuration
├── run-tests.sh             # Comprehensive test runner
└── README.md                # This file
```

## Key Features

### 🎯 Thesis Validation
- **Deployment Time**: Validates E2E deployment < 10 minutes
- **Throughput Targets**: Measures and validates 4.57/2.77/0.93 Mbps for eMBB/URLLC/mMTC
- **Latency Requirements**: Validates RTT of 16.1/15.7/6.3 ms (including TC overhead)
- **Comprehensive Reporting**: JSON/HTML reports with thesis compliance metrics

### 🔧 Test Types
- **Unit Tests**: Component-level validation
- **Integration Tests**: Cross-service interaction testing
- **E2E Tests**: Complete workflow validation
- **Performance Tests**: Throughput, latency, and resource utilization
- **Security Tests**: Security vulnerability scanning
- **Health Check Tests**: Service availability and health monitoring
- **Benchmark Tests**: Performance regression testing

### 🏗️ Infrastructure Features
- **Go 1.24.7 Compatibility**: Enforced across all tests
- **Kubernetes Integration**: Native K8s testing with envtest
- **Mock Services**: O2, Nephio, and other external service mocks
- **Parallel Execution**: Configurable parallel test execution
- **Docker Support**: Containerized test execution
- **CI/CD Ready**: Integrated with GitHub Actions workflows

## Quick Start

### Prerequisites

- Go 1.24.7+
- kubectl (for integration/E2E tests)
- Docker (optional, for containerized testing)
- Kubernetes cluster (kind/k3s/real cluster)

### Basic Usage

```bash
# Run all tests
make test

# Run specific test suites
make test-unit           # Unit tests only
make test-integration    # Integration tests
make test-e2e           # End-to-end tests
make test-performance   # Performance tests
make test-thesis        # Thesis validation
make test-security      # Security tests
make test-healthcheck   # Health checks

# Advanced usage
./tests/run-tests.sh --help                    # Show all options
./tests/run-tests.sh all --verbose --clean     # Clean run with verbose output
./tests/run-tests.sh thesis --timeout=45m      # Thesis validation with extended timeout
./tests/run-tests.sh performance --docker      # Performance tests in Docker
```

### Configuration

Tests can be configured via `tests/test.config.yaml`:

```yaml
test_suite: "all"
parallel: true
verbose: true
timeout: "30m"
coverage: true
environment:
  THESIS_VALIDATION: "true"
  INTEGRATION_TESTS: "true"

# Thesis validation thresholds
thesis_requirements:
  deployment_time_max: "10m"
  throughput_targets:
    embb: 4.57  # Mbps
    urllc: 2.77 # Mbps
    mmtc: 0.93  # Mbps
  latency_targets:
    embb: 16.1  # ms
    urllc: 15.7 # ms
    mmtc: 6.3   # ms
```

## Test Execution

### Common Commands

```bash
# Development workflow
./tests/run-tests.sh unit --fast                # Quick unit tests
./tests/run-tests.sh integration --parallel     # Parallel integration tests
./tests/run-tests.sh e2e --timeout=45m         # Extended E2E tests

# CI/CD pipeline
./tests/run-tests.sh all --docker --clean      # Complete CI test run
./tests/run-tests.sh thesis --no-parallel      # Thesis validation

# Debugging
./tests/run-tests.sh unit --verbose            # Verbose output
./tests/run-tests.sh healthcheck --config=custom.yaml  # Custom configuration
```

### Environment Variables

Key environment variables for test execution:

```bash
# Test control
THESIS_VALIDATION=true     # Enable thesis validation
INTEGRATION_TESTS=true     # Enable integration tests
PERFORMANCE_TESTS=true     # Enable performance tests

# Kubernetes
KUBECONFIG=/path/to/config # Kubernetes configuration
TEST_NAMESPACE=test-ns     # Test namespace

# Go configuration
GO_VERSION=1.24.7         # Enforced Go version
CGO_ENABLED=0             # Disable CGO for tests
```

## Test Framework Components

### 1. Test Runner (`cmd/test-runner/main.go`)
Comprehensive test orchestration engine that:
- Manages test suite execution
- Handles parallel/sequential execution
- Collects and aggregates results
- Generates comprehensive reports
- Validates thesis requirements

### 2. Integration Tests (`integration/`)
Cross-service interaction tests:
- **Orchestrator**: Intent processing, QoS mapping, placement decisions
- **RAN-DMS**: Radio resource management, slice configuration
- **CN-DMS**: Core network deployment, slice management
- **VNF-Operator**: VNF lifecycle management

### 3. Performance Tests (`performance/`)
Thesis validation and performance benchmarking:
- **Deployment Time**: E2E slice deployment timing
- **Throughput**: Network throughput measurement using iperf3
- **Latency**: RTT measurement with ping and custom tools
- **Resource Efficiency**: CPU, memory, and storage utilization

### 4. E2E Tests (`e2e/`)
Complete workflow validation:
- **Intent Flow**: Natural language → QoS → Deployment
- **Multi-Domain**: RAN + TN + CN coordination
- **Multi-Site**: Edge/cloud deployment scenarios
- **Failure Recovery**: Resilience and self-healing

### 5. Health Check Tests (`healthcheck/`)
Service monitoring and availability:
- **Service Health**: HTTP health endpoints
- **Database Connectivity**: PostgreSQL/Redis health
- **External Dependencies**: O2/Nephio/Prometheus connectivity
- **Resource Health**: Pod status and resource utilization

### 6. Test Utilities (`utils/`)
Common test infrastructure:
- **Test Helpers**: Environment setup, mocking, assertions
- **Metrics Collection**: Performance measurement and validation
- **Mock Services**: O2, Nephio, and external service mocks

## Thesis Validation

The framework specifically validates thesis requirements:

### Deployment Time Requirement
- **Target**: E2E deployment < 10 minutes
- **Measurement**: Complete intent-to-deployment timing
- **Validation**: Automated pass/fail with detailed reporting

### Throughput Requirements
- **eMBB**: 4.57 Mbps target (±10% tolerance)
- **URLLC**: 2.77 Mbps target (±10% tolerance)
- **mMTC**: 0.93 Mbps target (±10% tolerance)
- **Measurement**: iperf3-based network throughput testing

### Latency Requirements (including TC overhead)
- **eMBB**: 16.1 ms RTT target
- **URLLC**: 15.7 ms RTT target
- **mMTC**: 6.3 ms RTT target
- **Measurement**: ping-based RTT measurement with TC consideration

### Compliance Reporting
Comprehensive thesis compliance reports generated in JSON/HTML format:
```json
{
  "compliance_report": {
    "overall_compliance": true,
    "deployment_compliance": true,
    "throughput_compliance": true,
    "latency_compliance": true,
    "violations": [],
    "violation_count": 0
  }
}
```

## Docker Build Fix

This testing infrastructure resolves Docker build failures by:

1. **Creating missing `/tests` directory** with proper Go module structure
2. **Providing comprehensive test framework** for all services
3. **Including test dependencies** in go.mod/go.sum
4. **Supporting CI/CD integration** with proper artifact generation
5. **Enabling thesis validation** with automated compliance checking

## Quick Test Execution

```bash
# Validate Docker build fix
cd tests && go mod verify && go mod tidy

# Run basic test suite to verify framework
./tests/run-tests.sh unit --verbose

# Full thesis validation
make test-thesis

# CI/CD compatible test run
./tests/run-tests.sh all --docker --clean
```

## Support and Troubleshooting

For common issues:
1. **Go Module Issues**: `cd tests && go mod tidy && go mod verify`
2. **Kubernetes Connectivity**: `kubectl cluster-info && kubectl get nodes`
3. **Test Failures**: `./tests/run-tests.sh unit --verbose`
4. **Performance Issues**: `THESIS_VALIDATION=true ./tests/run-tests.sh thesis --timeout=60m`

Enable debug mode: `export TEST_DEBUG=true`

## Version Compatibility
- **Go**: 1.24.7 (enforced)
- **Kubernetes**: 1.31.0+
- **Ginkgo**: v2.25.3+
- **Testify**: v1.11.1+