#!/bin/bash

# ==============================================================================
# O-RAN Intent MANO - QoSProfile Refactoring Test Plan
# ==============================================================================
# This script provides comprehensive testing for QoSProfile refactoring
# which converts between VNFQoSProfile (simple strings) and QoSProfile (structured)
#
# TEST SCOPE:
# - Type conversion functions (VNFQoSProfile ↔ QoSProfile)
# - Integration tests for fixtures usage
# - Compilation tests for all packages
# - Docker build tests
# - Round-trip conversion validation
# - Edge cases and error conditions
# ==============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
TEST_OUTPUT_DIR="$PROJECT_ROOT/tests/reports"
LOG_FILE="$TEST_OUTPUT_DIR/qos_refactoring_test_$(date +%Y%m%d_%H%M%S).log"
VERBOSE=${VERBOSE:-false}
DRY_RUN=${DRY_RUN:-false}

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# ==============================================================================
# UTILITY FUNCTIONS
# ==============================================================================

log() {
    local level=$1; shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    case $level in
        INFO) echo -e "${BLUE}[INFO]${NC} $message" | tee -a "$LOG_FILE" ;;
        WARN) echo -e "${YELLOW}[WARN]${NC} $message" | tee -a "$LOG_FILE" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} $message" | tee -a "$LOG_FILE" ;;
        SUCCESS) echo -e "${GREEN}[SUCCESS]${NC} $message" | tee -a "$LOG_FILE" ;;
        DEBUG) [ "$VERBOSE" = true ] && echo -e "[DEBUG] $message" | tee -a "$LOG_FILE" ;;
    esac
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_exit_code="${3:-0}"

    ((TOTAL_TESTS++))
    log INFO "Running test: $test_name"

    if [ "$DRY_RUN" = true ]; then
        log INFO "DRY RUN: Would execute: $test_command"
        ((PASSED_TESTS++))
        return 0
    fi

    local start_time=$(date +%s)
    if eval "$test_command" >> "$LOG_FILE" 2>&1; then
        local actual_exit_code=0
    else
        local actual_exit_code=$?
    fi
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [ $actual_exit_code -eq $expected_exit_code ]; then
        log SUCCESS "✓ $test_name (${duration}s)"
        ((PASSED_TESTS++))
        return 0
    else
        log ERROR "✗ $test_name failed (exit code: $actual_exit_code, expected: $expected_exit_code) (${duration}s)"
        ((FAILED_TESTS++))
        return 1
    fi
}

skip_test() {
    local test_name="$1"
    local reason="$2"

    ((TOTAL_TESTS++))
    ((SKIPPED_TESTS++))
    log WARN "⚠ Skipping $test_name: $reason"
}

setup_test_environment() {
    log INFO "Setting up test environment..."

    # Create output directory
    mkdir -p "$TEST_OUTPUT_DIR"

    # Change to project root
    cd "$PROJECT_ROOT"

    # Verify Go installation
    if ! command -v go &> /dev/null; then
        log ERROR "Go is not installed or not in PATH"
        exit 1
    fi

    log INFO "Go version: $(go version)"
    log INFO "Project root: $PROJECT_ROOT"
    log INFO "Test output: $TEST_OUTPUT_DIR"
    log INFO "Log file: $LOG_FILE"
}

cleanup_test_environment() {
    log INFO "Cleaning up test environment..."

    # Clean up any temporary files
    find "$PROJECT_ROOT" -name "*.tmp" -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "*.test" -delete 2>/dev/null || true

    # Reset any modified files (if needed)
    git checkout -- . 2>/dev/null || true
}

# ==============================================================================
# PHASE 1: TYPE CONVERSION TESTS
# ==============================================================================

test_type_conversion_functions() {
    log INFO "=== PHASE 1: Type Conversion Tests ==="

    # Test 1.1: VNFQoSProfile to QoSProfile conversion
    run_test "VNFQoSProfile → QoSProfile conversion" \
        "cd tests && go test -v -run 'TestVNFQoSProfileToQoSProfile' ./fixtures/..."

    # Test 1.2: QoSProfile to VNFQoSProfile conversion
    run_test "QoSProfile → VNFQoSProfile conversion" \
        "cd tests && go test -v -run 'TestQoSProfileToVNFQoSProfile' ./fixtures/..."

    # Test 1.3: Round-trip conversion (lossless)
    run_test "Round-trip conversion validation" \
        "cd tests && go test -v -run 'TestRoundTripConversion' ./fixtures/..."

    # Test 1.4: Edge cases - nil values
    run_test "Edge case: nil pointer handling" \
        "cd tests && go test -v -run 'TestConversionNilPointers' ./fixtures/..."

    # Test 1.5: Edge cases - empty values
    run_test "Edge case: empty value handling" \
        "cd tests && go test -v -run 'TestConversionEmptyValues' ./fixtures/..."

    # Test 1.6: Edge cases - invalid formats
    run_test "Edge case: invalid format handling" \
        "cd tests && go test -v -run 'TestConversionInvalidFormats' ./fixtures/..."

    # Test 1.7: eMBB slice type conversion
    run_test "eMBB slice type conversion" \
        "cd tests && go test -v -run 'TestEMBBConversion' ./fixtures/..."

    # Test 1.8: URLLC slice type conversion
    run_test "URLLC slice type conversion" \
        "cd tests && go test -v -run 'TestURLLCConversion' ./fixtures/..."

    # Test 1.9: mMTC slice type conversion
    run_test "mMTC slice type conversion" \
        "cd tests && go test -v -run 'TestMmTCConversion' ./fixtures/..."

    # Test 1.10: Latency unit conversion
    run_test "Latency unit conversion (ms, μs, ns)" \
        "cd tests && go test -v -run 'TestLatencyUnitConversion' ./fixtures/..."

    # Test 1.11: Throughput unit conversion
    run_test "Throughput unit conversion (bps, Kbps, Mbps, Gbps)" \
        "cd tests && go test -v -run 'TestThroughputUnitConversion' ./fixtures/..."

    # Test 1.12: Reliability format conversion
    run_test "Reliability format conversion (percentage, nines)" \
        "cd tests && go test -v -run 'TestReliabilityFormatConversion' ./fixtures/..."
}

# ==============================================================================
# PHASE 2: FIXTURE INTEGRATION TESTS
# ==============================================================================

test_fixture_integration() {
    log INFO "=== PHASE 2: Fixture Integration Tests ==="

    # Test 2.1: ValidVNFDeployment() creates valid objects
    run_test "ValidVNFDeployment fixture validation" \
        "cd tests && go test -v -run 'TestValidVNFDeploymentFixture' ./fixtures/..."

    # Test 2.2: Test helpers work with both types
    run_test "Test helpers compatibility" \
        "cd tests && go test -v -run 'TestHelpersWithBothTypes' ./fixtures/..."

    # Test 2.3: Builder pattern functions correctly
    run_test "VNF builder pattern validation" \
        "cd tests && go test -v -run 'TestVNFBuilderPattern' ./fixtures/..."

    # Test 2.4: eMBB fixture validation
    run_test "eMBB fixture with new QoS structure" \
        "cd tests && go test -v -run 'TestEMBBFixtureValidation' ./fixtures/..."

    # Test 2.5: URLLC fixture validation
    run_test "URLLC fixture with new QoS structure" \
        "cd tests && go test -v -run 'TestURLLCFixtureValidation' ./fixtures/..."

    # Test 2.6: mMTC fixture validation
    run_test "mMTC fixture with new QoS structure" \
        "cd tests && go test -v -run 'TestMmTCFixtureValidation' ./fixtures/..."

    # Test 2.7: QoS profile validation for each slice type
    run_test "QoS profile validation across slice types" \
        "cd tests && go test -v -run 'TestQoSProfileValidation' ./fixtures/..."

    # Test 2.8: Resource profile compatibility
    run_test "Resource profile compatibility" \
        "cd tests && go test -v -run 'TestResourceProfileCompatibility' ./fixtures/..."

    # Test 2.9: Placement profile compatibility
    run_test "Placement profile compatibility" \
        "cd tests && go test -v -run 'TestPlacementProfileCompatibility' ./fixtures/..."

    # Test 2.10: JSON serialization/deserialization
    run_test "JSON marshal/unmarshal with new types" \
        "cd tests && go test -v -run 'TestJSONSerialization' ./fixtures/..."
}

# ==============================================================================
# PHASE 3: COMPILATION TESTS
# ==============================================================================

test_compilation() {
    log INFO "=== PHASE 3: Compilation Tests ==="

    # Test 3.1: orchestrator/pkg/intent compilation
    run_test "orchestrator/pkg/intent compilation" \
        "cd orchestrator && go build -v ./pkg/intent/..."

    # Test 3.2: orchestrator/pkg/placement compilation
    run_test "orchestrator/pkg/placement compilation" \
        "cd orchestrator && go build -v ./pkg/placement/..."

    # Test 3.3: cn-dms compilation
    run_test "cn-dms module compilation" \
        "cd cn-dms && go build -v ./..."

    # Test 3.4: ran-dms compilation
    run_test "ran-dms module compilation" \
        "cd ran-dms && go build -v ./..."

    # Test 3.5: adapters/vnf-operator compilation
    run_test "vnf-operator compilation" \
        "cd adapters/vnf-operator && go build -v ./..."

    # Test 3.6: tests module compilation
    run_test "tests module compilation" \
        "cd tests && go build -v ./..."

    # Test 3.7: All orchestrator packages
    run_test "All orchestrator packages compilation" \
        "cd orchestrator && go build -v ./..."

    # Test 3.8: Root module compilation
    run_test "Root module compilation" \
        "go build -v ./..."

    # Test 3.9: Go vet checks - orchestrator
    run_test "Go vet - orchestrator" \
        "cd orchestrator && go vet ./..."

    # Test 3.10: Go vet checks - cn-dms
    run_test "Go vet - cn-dms" \
        "cd cn-dms && go vet ./..."

    # Test 3.11: Go vet checks - ran-dms
    run_test "Go vet - ran-dms" \
        "cd ran-dms && go vet ./..."

    # Test 3.12: Go vet checks - vnf-operator
    run_test "Go vet - vnf-operator" \
        "cd adapters/vnf-operator && go vet ./..."

    # Test 3.13: Go vet checks - tests
    run_test "Go vet - tests" \
        "cd tests && go vet ./..."

    # Test 3.14: Go mod tidy verification
    run_test "Go mod tidy verification" \
        "go mod tidy && git diff --exit-code go.mod go.sum"
}

# ==============================================================================
# PHASE 4: INTEGRATION TESTS
# ==============================================================================

test_integration() {
    log INFO "=== PHASE 4: Integration Tests ==="

    # Test 4.1: orchestrator/pkg/intent tests
    run_test "orchestrator intent package tests" \
        "cd orchestrator && go test -v ./pkg/intent/..."

    # Test 4.2: orchestrator/pkg/placement tests
    run_test "orchestrator placement package tests" \
        "cd orchestrator && go test -v ./pkg/placement/..."

    # Test 4.3: cn-dms tests
    run_test "cn-dms integration tests" \
        "cd cn-dms && go test -v ./..."

    # Test 4.4: ran-dms tests
    run_test "ran-dms integration tests" \
        "cd ran-dms && go test -v ./..."

    # Test 4.5: vnf-operator tests
    run_test "vnf-operator integration tests" \
        "cd adapters/vnf-operator && go test -v ./..."

    # Test 4.6: End-to-end intent flow
    run_test "E2E intent processing flow" \
        "cd tests && go test -v -run 'TestE2EIntentFlow' ./integration/..."

    # Test 4.7: CN-DMS integration with new QoS
    run_test "CN-DMS QoS integration" \
        "cd tests && go test -v -run 'TestCNDMSQoSIntegration' ./integration/..."

    # Test 4.8: RAN-DMS integration with new QoS
    run_test "RAN-DMS QoS integration" \
        "cd tests && go test -v -run 'TestRANDMSQoSIntegration' ./integration/..."

    # Test 4.9: TN network integration
    run_test "TN network QoS integration" \
        "cd tests && go test -v -run 'TestTNNetworkQoSIntegration' ./integration/..."

    # Test 4.10: QoS validation performance test
    run_test "QoS validation performance" \
        "cd tests && go test -v -run 'TestQoSValidationPerformance' ./performance/..."

    # Test 4.11: API compatibility test
    run_test "API compatibility with new QoS types" \
        "cd tests && go test -v -run 'TestAPICompatibility' ./integration/..."

    # Test 4.12: Database model compatibility
    run_test "Database model compatibility" \
        "cd tests && go test -v -run 'TestDBModelCompatibility' ./integration/..."
}

# ==============================================================================
# PHASE 5: DOCKER BUILD TESTS
# ==============================================================================

test_docker_builds() {
    log INFO "=== PHASE 5: Docker Build Tests ==="

    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        skip_test "Docker build tests" "Docker not installed"
        return 0
    fi

    # Test 5.1: Orchestrator Docker build
    run_test "Orchestrator Docker build" \
        "docker build -t oran-orchestrator:test -f orchestrator/Dockerfile orchestrator/"

    # Test 5.2: CN-DMS Docker build
    run_test "CN-DMS Docker build" \
        "docker build -t oran-cn-dms:test -f cn-dms/Dockerfile cn-dms/"

    # Test 5.3: RAN-DMS Docker build
    run_test "RAN-DMS Docker build" \
        "docker build -t oran-ran-dms:test -f ran-dms/Dockerfile ran-dms/"

    # Test 5.4: VNF Operator Docker build
    run_test "VNF Operator Docker build" \
        "docker build -t oran-vnf-operator:test -f adapters/vnf-operator/Dockerfile adapters/vnf-operator/"

    # Test 5.5: Multi-stage build verification
    run_test "Multi-stage build verification" \
        "docker build --target builder -t oran-builder:test -f orchestrator/Dockerfile orchestrator/"

    # Test 5.6: Docker compose build
    if [ -f docker-compose.yml ]; then
        run_test "Docker compose build" \
            "docker-compose build --no-cache"
    else
        skip_test "Docker compose build" "docker-compose.yml not found"
    fi

    # Test 5.7: Container smoke test - orchestrator
    run_test "Orchestrator container smoke test" \
        "docker run --rm oran-orchestrator:test --version || docker run --rm oran-orchestrator:test --help"

    # Test 5.8: Container smoke test - cn-dms
    run_test "CN-DMS container smoke test" \
        "docker run --rm oran-cn-dms:test --version || docker run --rm oran-cn-dms:test --help"

    # Test 5.9: Container smoke test - ran-dms
    run_test "RAN-DMS container smoke test" \
        "docker run --rm oran-ran-dms:test --version || docker run --rm oran-ran-dms:test --help"

    # Test 5.10: Image size verification
    run_test "Docker image size verification" \
        "docker images --format 'table {{.Repository}}:{{.Tag}}\t{{.Size}}' | grep oran-"

    # Cleanup test images
    log INFO "Cleaning up Docker test images..."
    docker rmi -f oran-orchestrator:test oran-cn-dms:test oran-ran-dms:test oran-vnf-operator:test oran-builder:test 2>/dev/null || true
}

# ==============================================================================
# PHASE 6: PERFORMANCE AND STRESS TESTS
# ==============================================================================

test_performance() {
    log INFO "=== PHASE 6: Performance Tests ==="

    # Test 6.1: QoS conversion performance
    run_test "QoS conversion performance benchmark" \
        "cd tests && go test -v -bench=BenchmarkQoSConversion -run=^$ ./performance/..."

    # Test 6.2: Memory allocation test
    run_test "Memory allocation during conversion" \
        "cd tests && go test -v -bench=BenchmarkQoSConversionMemory -benchmem -run=^$ ./performance/..."

    # Test 6.3: Concurrent conversion test
    run_test "Concurrent QoS conversion test" \
        "cd tests && go test -v -run 'TestConcurrentQoSConversion' ./performance/..."

    # Test 6.4: Large dataset validation
    run_test "Large dataset QoS validation" \
        "cd tests && go test -v -run 'TestLargeDatasetValidation' ./performance/..."

    # Test 6.5: CPU profiling test
    run_test "CPU profiling during conversion" \
        "cd tests && go test -v -cpuprofile=cpu.prof -run 'TestQoSConversionProfiling' ./performance/..."

    # Test 6.6: Memory profiling test
    run_test "Memory profiling during conversion" \
        "cd tests && go test -v -memprofile=mem.prof -run 'TestQoSConversionProfiling' ./performance/..."
}

# ==============================================================================
# PHASE 7: SECURITY AND VALIDATION TESTS
# ==============================================================================

test_security_validation() {
    log INFO "=== PHASE 7: Security and Validation Tests ==="

    # Test 7.1: Input validation tests
    run_test "QoS input validation" \
        "cd tests && go test -v -run 'TestQoSInputValidation' ./security/..."

    # Test 7.2: SQL injection prevention
    run_test "SQL injection prevention in QoS queries" \
        "cd tests && go test -v -run 'TestQoSSQLInjectionPrevention' ./security/..."

    # Test 7.3: XSS prevention in QoS display
    run_test "XSS prevention in QoS data display" \
        "cd tests && go test -v -run 'TestQoSXSSPrevention' ./security/..."

    # Test 7.4: Data sanitization
    run_test "QoS data sanitization" \
        "cd tests && go test -v -run 'TestQoSDataSanitization' ./security/..."

    # Test 7.5: Authentication integration
    run_test "Authentication with QoS endpoints" \
        "cd tests && go test -v -run 'TestQoSAuthentication' ./security/..."

    # Test 7.6: Authorization checks
    run_test "Authorization for QoS modifications" \
        "cd tests && go test -v -run 'TestQoSAuthorization' ./security/..."
}

# ==============================================================================
# TROUBLESHOOTING GUIDE
# ==============================================================================

print_troubleshooting_guide() {
    log INFO "=== TROUBLESHOOTING GUIDE ==="
    cat << 'EOF'

COMMON ISSUES AND SOLUTIONS:

1. COMPILATION ERRORS:
   Problem: "undefined: QoSProfile" or "undefined: VNFQoSProfile"
   Solution:
   - Check import statements in affected files
   - Verify type definitions are correctly imported
   - Run: go mod tidy && go mod download

2. TYPE CONVERSION ERRORS:
   Problem: "cannot convert" between QoS types
   Solution:
   - Implement conversion functions in types.go
   - Add unit conversion utilities (ms, μs, bps, Mbps)
   - Validate format before conversion

3. FIXTURE VALIDATION FAILURES:
   Problem: Test fixtures fail validation with new types
   Solution:
   - Update fixture functions to use structured QoSProfile
   - Verify slice type compatibility with QoS requirements
   - Check latency/throughput/reliability bounds

4. DOCKER BUILD FAILURES:
   Problem: Docker build fails with "module not found"
   Solution:
   - Ensure go.mod and go.sum are up to date
   - Check Dockerfile COPY instructions
   - Verify Docker build context includes all necessary files

5. INTEGRATION TEST FAILURES:
   Problem: Integration tests fail with database errors
   Solution:
   - Update database schema for new QoS structure
   - Migrate existing data to new format
   - Verify database connection strings

6. PERFORMANCE ISSUES:
   Problem: QoS conversion is too slow
   Solution:
   - Implement caching for frequently converted values
   - Use sync.Pool for object reuse
   - Profile and optimize hot paths

7. MEMORY LEAKS:
   Problem: Memory usage increases during tests
   Solution:
   - Check for proper resource cleanup
   - Verify goroutine termination
   - Use pprof to identify leaks

8. CONCURRENT ACCESS ISSUES:
   Problem: Race conditions in QoS validation
   Solution:
   - Add proper mutex protection
   - Use atomic operations where appropriate
   - Test with -race flag

DEBUGGING COMMANDS:
- View test logs: tail -f $LOG_FILE
- Debug compilation: go build -v -x ./...
- Race detection: go test -race ./...
- Memory profiling: go test -memprofile=mem.prof
- CPU profiling: go test -cpuprofile=cpu.prof
- Trace execution: go test -trace=trace.out

EOF
}

# ==============================================================================
# EXECUTION SUMMARY AND REPORTING
# ==============================================================================

generate_test_report() {
    local report_file="$TEST_OUTPUT_DIR/qos_refactoring_test_report.html"

    log INFO "Generating test report: $report_file"

    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>QoSProfile Refactoring Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 10px; border-radius: 5px; }
        .summary { background-color: #e7f3ff; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .passed { color: green; font-weight: bold; }
        .failed { color: red; font-weight: bold; }
        .skipped { color: orange; font-weight: bold; }
        .section { margin: 20px 0; }
        .logs { background-color: #f5f5f5; padding: 10px; border-radius: 5px; white-space: pre-wrap; font-family: monospace; }
    </style>
</head>
<body>
    <div class="header">
        <h1>QoSProfile Refactoring Test Report</h1>
        <p>Generated: $(date)</p>
        <p>Project: O-RAN Intent MANO for Network Slicing</p>
    </div>

    <div class="summary">
        <h2>Test Summary</h2>
        <p><span class="passed">Passed: $PASSED_TESTS</span></p>
        <p><span class="failed">Failed: $FAILED_TESTS</span></p>
        <p><span class="skipped">Skipped: $SKIPPED_TESTS</span></p>
        <p><strong>Total: $TOTAL_TESTS</strong></p>
        <p><strong>Success Rate: $(( PASSED_TESTS * 100 / TOTAL_TESTS ))%</strong></p>
    </div>

    <div class="section">
        <h2>Test Phases</h2>
        <ul>
            <li>Phase 1: Type Conversion Tests</li>
            <li>Phase 2: Fixture Integration Tests</li>
            <li>Phase 3: Compilation Tests</li>
            <li>Phase 4: Integration Tests</li>
            <li>Phase 5: Docker Build Tests</li>
            <li>Phase 6: Performance Tests</li>
            <li>Phase 7: Security and Validation Tests</li>
        </ul>
    </div>

    <div class="section">
        <h2>Detailed Logs</h2>
        <div class="logs">$(cat "$LOG_FILE")</div>
    </div>
</body>
</html>
EOF
}

print_execution_summary() {
    log INFO "=== EXECUTION SUMMARY ==="
    echo "Total Tests:   $TOTAL_TESTS"
    echo "Passed Tests:  $PASSED_TESTS"
    echo "Failed Tests:  $FAILED_TESTS"
    echo "Skipped Tests: $SKIPPED_TESTS"

    if [ $FAILED_TESTS -eq 0 ]; then
        log SUCCESS "🎉 All tests passed successfully!"
        echo "✅ QoSProfile refactoring is ready for deployment"
    else
        log ERROR "❌ $FAILED_TESTS test(s) failed"
        echo "⚠️  Please review the failures before proceeding"
        echo "📋 Check the troubleshooting guide for common solutions"
    fi

    echo ""
    echo "📊 Test Report: $TEST_OUTPUT_DIR/qos_refactoring_test_report.html"
    echo "📝 Full Logs: $LOG_FILE"
}

# ==============================================================================
# MAIN EXECUTION FLOW
# ==============================================================================

main() {
    log INFO "Starting QoSProfile Refactoring Test Plan..."
    log INFO "================================================"

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --dry-run|-d)
                DRY_RUN=true
                shift
                ;;
            --phase|-p)
                PHASE="$2"
                shift 2
                ;;
            --help|-h)
                cat << 'EOF'
QoSProfile Refactoring Test Plan

Usage: ./qos_profile_refactoring_test_plan.sh [OPTIONS]

Options:
  -v, --verbose     Enable verbose output
  -d, --dry-run     Show what would be executed without running
  -p, --phase N     Run only specific phase (1-7)
  -h, --help        Show this help message

Phases:
  1. Type Conversion Tests
  2. Fixture Integration Tests
  3. Compilation Tests
  4. Integration Tests
  5. Docker Build Tests
  6. Performance Tests
  7. Security and Validation Tests

Environment Variables:
  VERBOSE=true      Enable verbose output
  DRY_RUN=true      Enable dry run mode

Examples:
  ./qos_profile_refactoring_test_plan.sh --verbose
  ./qos_profile_refactoring_test_plan.sh --phase 3
  VERBOSE=true ./qos_profile_refactoring_test_plan.sh

EOF
                exit 0
                ;;
            *)
                log ERROR "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Setup
    setup_test_environment

    # Execute test phases
    if [ -z "${PHASE:-}" ]; then
        # Run all phases
        test_type_conversion_functions
        test_fixture_integration
        test_compilation
        test_integration
        test_docker_builds
        test_performance
        test_security_validation
    else
        # Run specific phase
        case $PHASE in
            1) test_type_conversion_functions ;;
            2) test_fixture_integration ;;
            3) test_compilation ;;
            4) test_integration ;;
            5) test_docker_builds ;;
            6) test_performance ;;
            7) test_security_validation ;;
            *) log ERROR "Invalid phase: $PHASE (must be 1-7)"; exit 1 ;;
        esac
    fi

    # Generate reports
    generate_test_report
    print_execution_summary
    print_troubleshooting_guide

    # Cleanup
    cleanup_test_environment

    # Exit with appropriate code
    if [ $FAILED_TESTS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# ==============================================================================
# SCRIPT ENTRY POINT
# ==============================================================================

# Handle script interruption
trap cleanup_test_environment EXIT

# Execute main function with all arguments
main "$@"