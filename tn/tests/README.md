# O-RAN Intent MANO Testing Framework

This document provides comprehensive guidance for testing the O-RAN Intent-Based MANO system components, focusing on robust error handling, security validation, and system reliability.

## Overview

The testing framework covers multiple layers of validation:

- **Unit Tests**: Component-level testing with comprehensive error scenarios
- **Integration Tests**: End-to-end testing with real system interactions
- **Security Tests**: Kubernetes manifest validation and security policy compliance
- **Coverage Tests**: Ensuring >80% test coverage with quality metrics

## Test Structure

```
tn/tests/
├── README.md                          # This documentation
├── coverage/
│   └── coverage_test.go               # Coverage analysis and reporting
├── integration/
│   ├── http_integration_test.go       # HTTP endpoint integration tests
│   ├── iperf_integration_test.go      # Iperf network testing integration
│   └── vxlan_integration_test.go      # VXLAN tunnel integration tests
└── security/
    └── kubernetes_manifest_test.go    # K8s security validation
```

## Running Tests

### Prerequisites

1. **Go Environment** (1.19+)
2. **Network Tools** (for integration tests):
   - `iperf3` - Network performance testing
   - `ip` command - Network interface management
   - `ping` - Connectivity testing
3. **Kubernetes Tools** (for security tests):
   - `kubectl` - Kubernetes CLI
   - Access to Kubernetes manifests

### Basic Test Execution

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Run specific test suites
go test ./tn/tests/integration/
go test ./tn/tests/security/
go test ./tn/tests/coverage/
```

### Environment-Specific Testing

#### CI/CD Environment
```bash
# Short tests for CI/CD (skip long-running integration tests)
go test -short ./...

# Security-focused testing
go test ./tn/tests/security/ -v

# Coverage validation
go test ./tn/tests/coverage/ -v
```

#### Local Development
```bash
# Full test suite with integration tests
go test -v ./...

# Run with race detection
go test -race ./...

# Benchmark tests
go test -bench=. ./...
```

#### Production-like Environment
```bash
# Integration tests with real network interfaces
sudo go test ./tn/tests/integration/ -v

# VXLAN tests (requires CAP_NET_ADMIN)
sudo go test ./tn/tests/integration/vxlan_integration_test.go -v
```

## Test Categories

### Unit Tests

Located alongside source files (`*_test.go`), these tests focus on:

#### HTTP Handler Tests (`tn/agent/pkg/http_test.go`)
- **Error Handling**: All HTTP endpoints with error conditions
- **Security**: Input validation and sanitization
- **Edge Cases**: Malformed requests, large payloads, concurrent access
- **Mock Integration**: External dependencies properly mocked

Example test patterns:
```go
func TestHTTPHandlers_ErrorConditions(t *testing.T) {
    // Test invalid JSON input
    // Test missing required fields
    // Test network timeouts
    // Test resource unavailability
    // Test concurrent request handling
}
```

#### Iperf Manager Tests (`tn/agent/pkg/iperf_test.go`)
- **Network Failures**: Connection timeouts, refused connections
- **Security Validation**: Command injection prevention
- **Resource Management**: Port conflicts, process lifecycle
- **Error Recovery**: Graceful failure handling

#### VXLAN Manager Tests (`tn/agent/pkg/vxlan/optimized_manager_test.go`)
- **System-Level Errors**: Interface creation failures
- **Security**: Command injection prevention in network commands
- **Performance**: Concurrent tunnel operations
- **Resource Cleanup**: Proper resource deallocation

### Integration Tests

Comprehensive end-to-end testing with real system interactions:

#### HTTP Integration (`tn/tests/integration/http_integration_test.go`)
- **Real HTTP Server**: Full request/response cycle testing
- **Error Scenarios**: Network failures, service unavailability
- **Load Testing**: Concurrent request handling
- **Security**: CORS, content negotiation, large payloads

#### Iperf Integration (`tn/tests/integration/iperf_integration_test.go`)
- **Network Performance**: Real iperf3 server/client interactions
- **Error Handling**: Network timeouts, port conflicts
- **Concurrent Operations**: Multiple simultaneous tests
- **Resource Management**: Server lifecycle and cleanup

#### VXLAN Integration (`tn/tests/integration/vxlan_integration_test.go`)
- **Network Interface Creation**: Real VXLAN tunnel creation
- **System Interaction**: Actual `ip` command execution
- **Permission Handling**: Graceful degradation without privileges
- **Cleanup Validation**: Interface removal verification

### Security Tests

Comprehensive security validation framework:

#### Kubernetes Manifest Validation (`tn/tests/security/kubernetes_manifest_test.go`)
- **Security Policies**: 15+ security checks per manifest
- **Severity Classification**: HIGH/MEDIUM/LOW risk categorization
- **Compliance**: Pod Security Standards validation
- **Best Practices**: Resource limits, security contexts, RBAC

Security policies include:
- No privileged containers
- No root user execution
- Resource limits defined
- Security contexts configured
- Network policies present
- Read-only root filesystems
- Capability dropping
- No privilege escalation

### Coverage Tests

Quality assurance through coverage analysis:

#### Coverage Analysis (`tn/tests/coverage/coverage_test.go`)
- **Minimum Thresholds**: >80% overall coverage required
- **Critical Components**: >90% coverage for security-critical code
- **Package-Level Analysis**: Per-package coverage reporting
- **HTML Reports**: Visual coverage analysis

## Test Quality Standards

### Test Design Principles

1. **Isolation**: Tests don't depend on external state
2. **Reliability**: Tests pass consistently across environments
3. **Fast Execution**: Unit tests complete quickly (<100ms each)
4. **Clear Naming**: Test names describe the scenario being tested
5. **Comprehensive Coverage**: All error paths tested

### Error Handling Testing

All tests follow the pattern:
```go
func TestComponent_ErrorScenario(t *testing.T) {
    // Setup test environment
    // Configure failure condition
    // Execute operation
    // Verify error handling
    // Check system state consistency
}
```

### Security Testing Standards

Security tests validate:
- **Input Sanitization**: All user inputs properly validated
- **Command Injection Prevention**: System commands properly escaped
- **Privilege Escalation**: No unnecessary elevated permissions
- **Resource Limits**: Proper resource constraints applied

## Environment Setup

### Docker Test Environment

```bash
# Build test environment
docker build -f deploy/docker/Dockerfile.test .

# Run tests in container
docker run --rm -v $(pwd):/app test-image go test ./...
```

### Local Development Setup

```bash
# Install test dependencies
go mod download

# Install network tools (Ubuntu/Debian)
sudo apt-get install iperf3 iproute2 iputils-ping

# Install security scanning tools
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

### CI/CD Integration

```yaml
# .github/workflows/test.yml example
- name: Run Tests
  run: |
    go test -short -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

- name: Security Tests
  run: |
    go test ./tn/tests/security/ -v

- name: Coverage Check
  run: |
    go test ./tn/tests/coverage/ -v
```

## Troubleshooting

### Common Issues

#### Permission Errors
```bash
# VXLAN tests require network admin capabilities
sudo setcap cap_net_admin+ep $(which go)
# Or run specific tests with sudo
sudo go test ./tn/tests/integration/vxlan_integration_test.go
```

#### Network Tool Dependencies
```bash
# Install missing tools
sudo apt-get install iperf3 iproute2 bridge-utils
```

#### Coverage Report Issues
```bash
# Generate coverage first
go test -coverprofile=coverage.out ./...
# Then run coverage tests
go test ./tn/tests/coverage/
```

### Test Environment Validation

```bash
# Validate test environment
go test ./tn/tests/integration/ -v -run TestValidation
```

## Best Practices

### Writing New Tests

1. **Follow Naming Conventions**:
   - `TestFunctionName_Scenario`
   - `TestComponent_ErrorHandling`
   - `TestIntegration_EndToEndWorkflow`

2. **Use Test Helpers**:
   ```go
   func setupTestEnvironment(t *testing.T) *TestSuite {
       // Common setup logic
   }

   func (suite *TestSuite) cleanup() {
       // Common cleanup logic
   }
   ```

3. **Mock External Dependencies**:
   ```go
   type MockService struct {
       mock.Mock
   }

   func (m *MockService) ExternalCall() error {
       args := m.Called()
       return args.Error(0)
   }
   ```

4. **Test Error Paths**:
   ```go
   // Test both success and failure cases
   func TestFunction_Success(t *testing.T) { /* ... */ }
   func TestFunction_NetworkFailure(t *testing.T) { /* ... */ }
   func TestFunction_InvalidInput(t *testing.T) { /* ... */ }
   ```

### Performance Testing

```go
func BenchmarkComponent_Operation(b *testing.B) {
    // Setup
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Operation to benchmark
    }
}
```

### Security Testing

```go
func TestComponent_SecurityValidation(t *testing.T) {
    maliciousInputs := []string{
        "'; DROP TABLE users; --",
        "<script>alert('xss')</script>",
        "$(rm -rf /)",
    }

    for _, input := range maliciousInputs {
        err := component.ProcessInput(input)
        assert.Error(t, err, "Should reject malicious input")
    }
}
```

## Metrics and Reporting

### Coverage Metrics
- **Overall Coverage**: >80% (enforced)
- **Critical Components**: >90% (enforced)
- **Security Code**: >95% (recommended)

### Test Metrics
- **Test Execution Time**: <5 minutes total
- **Unit Test Speed**: <100ms per test
- **Integration Test Speed**: <30 seconds per test

### Quality Gates
- All HIGH severity security violations must be resolved
- No failing tests in CI/CD pipeline
- Coverage thresholds must be met
- No race conditions detected

## Contributing

When adding new functionality:

1. **Write Tests First** (TDD approach)
2. **Include Error Scenarios** in test cases
3. **Add Security Validation** for new inputs
4. **Update Coverage Requirements** if adding critical code
5. **Document Test Scenarios** in commit messages

For questions or issues with the testing framework, refer to the project documentation or create an issue in the repository.