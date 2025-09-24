#!/bin/bash
# Comprehensive test runner script for O-RAN Intent-MANO system

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_CONFIG="${SCRIPT_DIR}/test.config.yaml"
TEST_REPORTS_DIR="${SCRIPT_DIR}/test-reports"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TEST_SUITE="all"
VERBOSE=false
COVERAGE=true
PARALLEL=true
TIMEOUT="30m"
CLEAN=false
DOCKER=false

# Function to print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

print_success() { print_color "$GREEN" "$@"; }
print_error() { print_color "$RED" "$@"; }
print_warning() { print_color "$YELLOW" "$@"; }
print_info() { print_color "$BLUE" "$@"; }

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [TEST_SUITE]

Comprehensive test runner for O-RAN Intent-MANO system.

TEST_SUITE options:
    all                 Run all test suites (default)
    unit               Run unit tests only
    integration        Run integration tests only
    e2e                Run end-to-end tests only
    performance        Run performance tests only
    security           Run security tests only
    healthcheck        Run health check tests only
    thesis             Run thesis validation tests only
    benchmark          Run benchmark tests only

OPTIONS:
    -h, --help         Show this help message
    -v, --verbose      Enable verbose output
    -c, --coverage     Enable coverage reporting (default: true)
    --no-coverage      Disable coverage reporting
    -p, --parallel     Enable parallel execution (default: true)
    --no-parallel      Disable parallel execution
    -t, --timeout      Set test timeout (default: 30m)
    --clean            Clean test artifacts before running
    --docker           Run tests in Docker containers
    --config           Specify custom test configuration file
    --reports-dir      Specify custom test reports directory

Examples:
    $0                           # Run all tests with default settings
    $0 unit                      # Run only unit tests
    $0 integration --verbose     # Run integration tests with verbose output
    $0 performance --timeout=45m # Run performance tests with 45min timeout
    $0 thesis --no-parallel      # Run thesis validation sequentially
    $0 --clean --docker all      # Clean and run all tests in Docker

Environment Variables:
    THESIS_VALIDATION     Set to 'true' to enable thesis validation
    INTEGRATION_TESTS     Set to 'true' to enable integration tests
    PERFORMANCE_TESTS     Set to 'true' to enable performance tests
    KUBECONFIG           Path to Kubernetes configuration file
    TEST_NAMESPACE       Kubernetes namespace for testing
EOF
}

# Function to validate dependencies
check_dependencies() {
    print_info "🔍 Checking dependencies..."

    local missing_deps=()

    # Check Go
    if ! command -v go >/dev/null 2>&1; then
        missing_deps+=("go")
    else
        local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+' | sed 's/go//')
        if [[ "$go_version" < "1.24.0" ]]; then
            print_warning "⚠️  Go version $go_version found, recommend 1.24.7+"
        fi
    fi

    # Check kubectl if running integration/e2e tests
    if [[ "$TEST_SUITE" =~ ^(all|integration|e2e|healthcheck)$ ]]; then
        if ! command -v kubectl >/dev/null 2>&1; then
            missing_deps+=("kubectl")
        fi
    fi

    # Check ginkgo for BDD tests
    if ! command -v ginkgo >/dev/null 2>&1; then
        print_info "📦 Installing ginkgo..."
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    fi

    # Check Docker if requested
    if [[ "$DOCKER" == "true" ]]; then
        if ! command -v docker >/dev/null 2>&1; then
            missing_deps+=("docker")
        fi
    fi

    if [[ ${#missing_deps[@]} -ne 0 ]]; then
        print_error "❌ Missing dependencies: ${missing_deps[*]}"
        print_info "Please install the missing dependencies and try again."
        exit 1
    fi

    print_success "✅ All dependencies satisfied"
}

# Function to setup test environment
setup_test_environment() {
    print_info "🛠️  Setting up test environment..."

    # Create reports directory
    mkdir -p "$TEST_REPORTS_DIR"

    # Set Go environment variables
    export GO_VERSION="1.24.7"
    export CGO_ENABLED="0"
    export GOOS="linux"
    export GOARCH="amd64"

    # Set test environment variables
    export THESIS_VALIDATION="${THESIS_VALIDATION:-true}"
    export INTEGRATION_TESTS="${INTEGRATION_TESTS:-true}"
    export PERFORMANCE_TESTS="${PERFORMANCE_TESTS:-true}"
    export TEST_TIMEOUT="$TIMEOUT"

    # Set Kubernetes environment if available
    if command -v kubectl >/dev/null 2>&1; then
        export KUBERNETES_VERSION="${KUBERNETES_VERSION:-$(kubectl version --client -o yaml | grep gitVersion | cut -d'"' -f4)}"
    fi

    print_success "✅ Test environment ready"
}

# Function to clean test artifacts
clean_artifacts() {
    print_info "🧹 Cleaning test artifacts..."

    # Clean test reports
    rm -rf "$TEST_REPORTS_DIR"
    mkdir -p "$TEST_REPORTS_DIR"

    # Clean coverage files
    find "$SCRIPT_DIR" -name "*.out" -type f -delete
    find "$SCRIPT_DIR" -name "coverage.html" -type f -delete

    # Clean test binaries
    find "$SCRIPT_DIR" -name "*.test" -type f -delete

    print_success "✅ Artifacts cleaned"
}

# Function to run specific test suite
run_test_suite() {
    local suite="$1"
    print_info "🧪 Running $suite tests..."

    local test_cmd="go run ./cmd/test-runner"
    local test_args=()

    # Add suite-specific arguments
    test_args+=("-suite=$suite")

    if [[ "$VERBOSE" == "true" ]]; then
        test_args+=("-verbose")
    fi

    if [[ "$COVERAGE" == "true" ]]; then
        test_args+=("-coverage")
    fi

    if [[ "$PARALLEL" == "true" ]]; then
        test_args+=("-parallel")
    fi

    test_args+=("-timeout=$TIMEOUT")
    test_args+=("-output=$TEST_REPORTS_DIR/${suite}-results.json")

    # Add configuration file if exists
    if [[ -f "$TEST_CONFIG" ]]; then
        test_args+=("-config=$TEST_CONFIG")
    fi

    # Execute test command
    cd "$SCRIPT_DIR"

    if [[ "$DOCKER" == "true" ]]; then
        run_tests_in_docker "$suite" "${test_args[@]}"
    else
        if $test_cmd "${test_args[@]}"; then
            print_success "✅ $suite tests passed"
            return 0
        else
            print_error "❌ $suite tests failed"
            return 1
        fi
    fi
}

# Function to run tests in Docker
run_tests_in_docker() {
    local suite="$1"
    shift
    local test_args=("$@")

    print_info "🐳 Running $suite tests in Docker..."

    local docker_image="golang:1.24.7-alpine"
    local docker_args=(
        "run" "--rm"
        "-v" "$PROJECT_ROOT:/workspace"
        "-w" "/workspace/tests"
        "-e" "THESIS_VALIDATION=$THESIS_VALIDATION"
        "-e" "INTEGRATION_TESTS=$INTEGRATION_TESTS"
        "-e" "PERFORMANCE_TESTS=$PERFORMANCE_TESTS"
        "$docker_image"
    )

    # Install dependencies in container and run tests
    docker "${docker_args[@]}" sh -c "
        apk add --no-cache git &&
        go mod download &&
        go run ./cmd/test-runner ${test_args[*]}
    "
}

# Function to generate comprehensive report
generate_report() {
    print_info "📊 Generating test report..."

    local report_file="$TEST_REPORTS_DIR/comprehensive-test-report.html"
    local json_files=("$TEST_REPORTS_DIR"/*.json)

    if [[ ${#json_files[@]} -eq 0 ]]; then
        print_warning "⚠️  No test results found to generate report"
        return
    fi

    # Create HTML report
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>O-RAN Intent-MANO Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f4f4f4; padding: 20px; border-radius: 5px; }
        .summary { display: flex; gap: 20px; margin: 20px 0; }
        .metric { background: #e8f5e8; padding: 15px; border-radius: 5px; flex: 1; }
        .passed { color: #28a745; }
        .failed { color: #dc3545; }
        .warning { color: #ffc107; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f8f9fa; }
    </style>
</head>
<body>
    <div class="header">
        <h1>O-RAN Intent-MANO Test Report</h1>
        <p>Generated on: $(date)</p>
        <p>Test Suite: $TEST_SUITE</p>
    </div>

    <div class="summary">
        <div class="metric">
            <h3>Test Results</h3>
            <p>Total Suites: <span id="total-suites">-</span></p>
            <p>Passed: <span class="passed" id="passed-suites">-</span></p>
            <p>Failed: <span class="failed" id="failed-suites">-</span></p>
        </div>
        <div class="metric">
            <h3>Coverage</h3>
            <p>Overall: <span id="coverage">-</span></p>
            <p>Threshold: 80%</p>
        </div>
        <div class="metric">
            <h3>Performance</h3>
            <p>Total Duration: <span id="duration">-</span></p>
            <p>Thesis Compliance: <span id="thesis-compliance">-</span></p>
        </div>
    </div>

    <h2>Detailed Results</h2>
    <table>
        <thead>
            <tr>
                <th>Test Suite</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Tests</th>
                <th>Coverage</th>
            </tr>
        </thead>
        <tbody id="results-table">
        </tbody>
    </table>
</body>
</html>
EOF

    print_success "✅ Test report generated: $report_file"
}

# Function to display summary
display_summary() {
    print_info "📋 Test Summary:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test Suite:     $TEST_SUITE"
    echo "Timeout:        $TIMEOUT"
    echo "Coverage:       $COVERAGE"
    echo "Parallel:       $PARALLEL"
    echo "Reports Dir:    $TEST_REPORTS_DIR"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main execution function
main() {
    local failed_suites=()
    local total_start_time=$(date +%s)

    print_info "🚀 O-RAN Intent-MANO Test Runner Started"

    # Setup
    check_dependencies
    setup_test_environment

    if [[ "$CLEAN" == "true" ]]; then
        clean_artifacts
    fi

    display_summary

    # Run tests based on suite selection
    case "$TEST_SUITE" in
        "all")
            local suites=("unit" "integration" "e2e" "performance" "security" "healthcheck")
            for suite in "${suites[@]}"; do
                if ! run_test_suite "$suite"; then
                    failed_suites+=("$suite")
                fi
            done
            ;;
        "thesis")
            # Special thesis validation suite
            export THESIS_VALIDATION="true"
            export PERFORMANCE_TESTS="true"
            if ! run_test_suite "performance"; then
                failed_suites+=("thesis-validation")
            fi
            ;;
        "benchmark")
            # Run benchmark tests
            cd "$SCRIPT_DIR"
            go test -bench=. -benchmem ./performance/ | tee "$TEST_REPORTS_DIR/benchmark-results.txt"
            ;;
        *)
            if ! run_test_suite "$TEST_SUITE"; then
                failed_suites+=("$TEST_SUITE")
            fi
            ;;
    esac

    # Generate report
    generate_report

    # Calculate total duration
    local total_end_time=$(date +%s)
    local total_duration=$((total_end_time - total_start_time))

    # Print final results
    echo ""
    print_info "🏁 Test Execution Completed"
    print_info "Total Duration: ${total_duration}s"

    if [[ ${#failed_suites[@]} -eq 0 ]]; then
        print_success "🎉 All tests passed successfully!"
        exit 0
    else
        print_error "💥 Failed test suites: ${failed_suites[*]}"
        print_info "Check test reports in: $TEST_REPORTS_DIR"
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        --no-coverage)
            COVERAGE=false
            shift
            ;;
        -p|--parallel)
            PARALLEL=true
            shift
            ;;
        --no-parallel)
            PARALLEL=false
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --timeout=*)
            TIMEOUT="${1#*=}"
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        --docker)
            DOCKER=true
            shift
            ;;
        --config)
            TEST_CONFIG="$2"
            shift 2
            ;;
        --config=*)
            TEST_CONFIG="${1#*=}"
            shift
            ;;
        --reports-dir)
            TEST_REPORTS_DIR="$2"
            shift 2
            ;;
        --reports-dir=*)
            TEST_REPORTS_DIR="${1#*=}"
            shift
            ;;
        unit|integration|e2e|performance|security|healthcheck|thesis|benchmark|all)
            TEST_SUITE="$1"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Execute main function
main "$@"