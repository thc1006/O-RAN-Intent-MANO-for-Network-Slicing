# QoSProfile Refactoring Test Plan

## Overview

This comprehensive test suite validates the QoSProfile refactoring in the O-RAN Intent MANO for Network Slicing project. The refactoring converts between two QoS profile structures:

- **VNFQoSProfile**: Simple string-based format used in VNF deployments
- **QoSProfile**: Structured format with detailed requirements for intent processing

## Test Structure

```
tests/
├── qos_profile_refactoring_test_plan.sh  # Main executable test script
├── unit/
│   └── qos_conversion_test.go             # Type conversion unit tests
├── integration/
│   └── qos_integration_test.go            # Integration and fixture tests
├── performance/
│   └── qos_performance_test.go            # Performance and benchmarks
├── security/
│   └── qos_security_test.go               # Security validation tests
├── fixtures/
│   ├── test_helpers.go                    # Test utilities and converters
│   ├── vnf_deployment_fixtures.go         # VNF deployment test data
│   └── intent_fixtures.go                # Intent processing test data
└── reports/                               # Generated test reports
```

## Quick Start

### Prerequisites

- Go 1.19 or later
- Docker (optional, for container tests)
- Git

### Running All Tests

```bash
# Make script executable (if not already)
chmod +x tests/qos_profile_refactoring_test_plan.sh

# Run all test phases
./tests/qos_profile_refactoring_test_plan.sh

# Run with verbose output
./tests/qos_profile_refactoring_test_plan.sh --verbose

# Run specific phase only
./tests/qos_profile_refactoring_test_plan.sh --phase 3

# Dry run (show what would be executed)
./tests/qos_profile_refactoring_test_plan.sh --dry-run
```

### Running Individual Test Suites

```bash
# Unit tests only
cd tests && go test -v ./unit/...

# Integration tests only
cd tests && go test -v ./integration/...

# Performance benchmarks
cd tests && go test -v -bench=. ./performance/...

# Security tests
cd tests && go test -v ./security/...
```

## Test Phases

### Phase 1: Type Conversion Tests

**Scope**: Core conversion functions between VNFQoSProfile and QoSProfile

**Tests**:
- VNFQoSProfile → QoSProfile conversion
- QoSProfile → VNFQoSProfile conversion (when implemented)
- Round-trip conversion validation
- Edge cases (nil, empty, invalid formats)
- Slice type specific conversions (eMBB, URLLC, mMTC)
- Unit conversions (latency, throughput, reliability)

**Location**: `tests/unit/qos_conversion_test.go`

**Example**:
```bash
# Run only type conversion tests
go test -v -run "TestVNFQoSProfileToQoSProfile" ./unit/...
```

### Phase 2: Fixture Integration Tests

**Scope**: Test fixture compatibility and helper functions

**Tests**:
- ValidVNFDeployment() fixture validation
- Test helper compatibility with both QoS types
- Builder pattern functionality
- Slice-specific fixture validation
- JSON serialization/deserialization
- Concurrent conversion safety

**Location**: `tests/integration/qos_integration_test.go`

**Example**:
```bash
# Run fixture tests
go test -v -run "TestValidVNFDeploymentFixture" ./integration/...
```

### Phase 3: Compilation Tests

**Scope**: Verify all packages compile with new types

**Tests**:
- orchestrator/pkg/intent compilation
- orchestrator/pkg/placement compilation
- cn-dms, ran-dms, vnf-operator compilation
- Go vet checks across all modules
- go mod tidy verification

**Example**:
```bash
# Manual compilation test
cd orchestrator && go build -v ./...
cd cn-dms && go build -v ./...
```

### Phase 4: Integration Tests

**Scope**: End-to-end integration with existing systems

**Tests**:
- Orchestrator package tests
- DMS integration tests
- API compatibility validation
- Database model compatibility
- Performance under integration load

**Location**: `tests/integration/qos_integration_test.go`

### Phase 5: Docker Build Tests

**Scope**: Container build and deployment validation

**Tests**:
- Individual service Docker builds
- Multi-stage build verification
- Container smoke tests
- Docker compose builds
- Image size validation

**Example**:
```bash
# Manual Docker test
docker build -t oran-orchestrator:test -f orchestrator/Dockerfile orchestrator/
```

### Phase 6: Performance Tests

**Scope**: Performance benchmarks and optimization validation

**Tests**:
- Conversion performance benchmarks
- Memory allocation analysis
- Concurrent conversion testing
- Large dataset validation
- CPU/Memory profiling

**Location**: `tests/performance/qos_performance_test.go`

**Example**:
```bash
# Run performance benchmarks
go test -v -bench=BenchmarkQoSConversion ./performance/...

# With memory profiling
go test -v -bench=. -memprofile=mem.prof ./performance/...
```

### Phase 7: Security and Validation Tests

**Scope**: Security validation and input sanitization

**Tests**:
- Input validation and sanitization
- SQL injection prevention
- XSS prevention
- Data integrity under attack
- Authentication/authorization integration
- Boundary value security

**Location**: `tests/security/qos_security_test.go`

**Example**:
```bash
# Run security tests
go test -v ./security/...
```

## Test Requirements

### Type Conversion Requirements

1. **Lossless Conversion**: Round-trip conversion should preserve data integrity
2. **Unit Normalization**: Consistent unit handling (ms, bps, percentage)
3. **Slice Type Validation**:
   - eMBB: Latency ≤ 100ms, High throughput
   - URLLC: Latency ≤ 1ms, Reliability ≥ 99.99%
   - mMTC: Latency ≤ 1000ms, Device density support
4. **Error Handling**: Graceful handling of invalid inputs
5. **Performance**: Conversion should complete in < 10μs per operation

### Integration Requirements

1. **API Compatibility**: Existing APIs continue to work
2. **Database Schema**: No breaking changes to stored data
3. **Fixture Validity**: All test fixtures remain valid
4. **Concurrent Safety**: Thread-safe conversion operations
5. **Memory Safety**: No memory leaks during conversion

### Security Requirements

1. **Input Sanitization**: All user inputs are sanitized
2. **Injection Prevention**: SQL, XSS, command injection prevention
3. **Data Validation**: Boundary value checking
4. **Authentication**: Proper auth integration
5. **Authorization**: Role-based access control maintained

## Expected Outcomes

### Success Criteria

- **All compilation tests pass**: No build errors across modules
- **Performance targets met**: >50k conversions/sec, <1KB memory/op
- **Security validation passes**: No injection vulnerabilities
- **Integration tests pass**: Existing functionality preserved
- **Docker builds succeed**: All containers build and run

### Pass/Fail Criteria

| Test Phase | Pass Criteria | Fail Criteria |
|------------|---------------|---------------|
| Type Conversion | All conversions work, units preserved | Conversion errors, data loss |
| Fixtures | All fixtures valid, helpers work | Fixture validation failures |
| Compilation | Clean builds, no vet issues | Build failures, type errors |
| Integration | Tests pass, APIs work | Test failures, breaking changes |
| Docker | Images build, containers run | Build failures, runtime errors |
| Performance | >10k ops/sec, <1MB heap growth | Poor performance, memory leaks |
| Security | No vulnerabilities found | Injection vulnerabilities |

## Troubleshooting

### Common Issues

#### Compilation Errors
```bash
# Problem: "undefined: QoSProfile"
# Solution: Check imports and run go mod tidy
go mod tidy && go mod download
```

#### Type Conversion Errors
```bash
# Problem: Cannot convert between types
# Solution: Implement conversion functions
# Check: tests/fixtures/test_helpers.go
```

#### Docker Build Failures
```bash
# Problem: Module not found in Docker
# Solution: Ensure go.mod is copied correctly
COPY go.mod go.sum ./
RUN go mod download
```

#### Performance Issues
```bash
# Profile performance
go test -v -bench=. -cpuprofile=cpu.prof ./performance/...
go tool pprof cpu.prof
```

### Debug Commands

```bash
# View test logs
tail -f tests/reports/qos_refactoring_test_*.log

# Debug compilation
go build -v -x ./...

# Race detection
go test -race ./...

# Memory profiling
go test -memprofile=mem.prof ./...

# CPU profiling
go test -cpuprofile=cpu.prof ./...
```

## Reports and Metrics

### Generated Reports

- **HTML Report**: `tests/reports/qos_refactoring_test_report.html`
- **Log File**: `tests/reports/qos_refactoring_test_*.log`
- **Performance Profiles**: `tests/performance/*.prof`

### Key Metrics

- **Test Coverage**: Target >90% for conversion functions
- **Performance**: >50k conversions/second
- **Memory Usage**: <1KB per conversion operation
- **Security Score**: Zero injection vulnerabilities
- **Build Time**: <5 minutes for all containers

## Key Files Created

### Main Test Script
- `/tests/qos_profile_refactoring_test_plan.sh` - Comprehensive executable test plan with 7 phases

### Test Implementation Files
- `/tests/unit/qos_conversion_test.go` - Type conversion unit tests (12 test functions)
- `/tests/integration/qos_integration_test.go` - Integration tests (12 test functions)
- `/tests/performance/qos_performance_test.go` - Performance benchmarks (8 benchmark functions)
- `/tests/security/qos_security_test.go` - Security validation tests (8 test functions)

### Enhanced Test Helpers
- Updated `/tests/fixtures/test_helpers.go` with QoS conversion utilities

### Documentation
- `/tests/QOS_TEST_README.md` - Comprehensive test documentation

## Test Commands Summary

```bash
# Execute complete test suite
./tests/qos_profile_refactoring_test_plan.sh

# Run individual phases
./tests/qos_profile_refactoring_test_plan.sh --phase 1  # Type conversion
./tests/qos_profile_refactoring_test_plan.sh --phase 2  # Fixture integration
./tests/qos_profile_refactoring_test_plan.sh --phase 3  # Compilation
./tests/qos_profile_refactoring_test_plan.sh --phase 4  # Integration
./tests/qos_profile_refactoring_test_plan.sh --phase 5  # Docker builds
./tests/qos_profile_refactoring_test_plan.sh --phase 6  # Performance
./tests/qos_profile_refactoring_test_plan.sh --phase 7  # Security

# Run with options
./tests/qos_profile_refactoring_test_plan.sh --verbose   # Detailed output
./tests/qos_profile_refactoring_test_plan.sh --dry-run   # Show commands only
```

The test plan is now ready for execution and provides comprehensive validation of the QoSProfile refactoring across all critical dimensions: functionality, performance, security, and integration compatibility.