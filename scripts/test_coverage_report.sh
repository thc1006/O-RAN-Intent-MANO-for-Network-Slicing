#!/bin/bash
# Test Coverage Report Generator for O-RAN Intent MANO Project
# This script generates comprehensive test coverage reports with modern testing standards

set -euo pipefail

# Configuration
PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
REPORT_DIR="${PROJECT_ROOT}/test-reports"
COVERAGE_FILE="${REPORT_DIR}/coverage.out"
HTML_REPORT="${REPORT_DIR}/coverage.html"
JSON_REPORT="${REPORT_DIR}/coverage.json"
SUMMARY_REPORT="${REPORT_DIR}/coverage_summary.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create report directory
mkdir -p "${REPORT_DIR}"

echo -e "${BLUE}🧪 O-RAN Intent MANO - Test Coverage Report Generator${NC}"
echo -e "${BLUE}===============================================${NC}"
echo ""

# Function to print colored status
print_status() {
    local status=$1
    local message=$2
    case $status in
        "SUCCESS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Function to check Go version and test support
check_prerequisites() {
    print_status "INFO" "Checking prerequisites..."

    if ! command -v go &> /dev/null; then
        print_status "ERROR" "Go is not installed or not in PATH"
        exit 1
    fi

    local go_version
    go_version=$(go version | cut -d' ' -f3)
    print_status "SUCCESS" "Go version: $go_version"

    # Check if go modules are available
    if [[ ! -f "${PROJECT_ROOT}/go.mod" ]]; then
        print_status "ERROR" "go.mod not found. Please run from project root."
        exit 1
    fi

    print_status "SUCCESS" "Prerequisites check passed"
}

# Function to run basic tests
run_basic_tests() {
    print_status "INFO" "Running basic test suite..."

    cd "${PROJECT_ROOT}"

    # Run tests with timeout and coverage
    if timeout 300s go test -short -cover -coverprofile="${COVERAGE_FILE}" ./... 2>&1 | tee "${REPORT_DIR}/test_output.log"; then
        print_status "SUCCESS" "Basic tests completed"
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            print_status "WARNING" "Tests timed out after 5 minutes"
        else
            print_status "WARNING" "Some tests failed or encountered issues (exit code: $exit_code)"
        fi
    fi
}

# Function to generate HTML coverage report
generate_html_report() {
    print_status "INFO" "Generating HTML coverage report..."

    if [[ -f "${COVERAGE_FILE}" ]]; then
        go tool cover -html="${COVERAGE_FILE}" -o="${HTML_REPORT}"
        print_status "SUCCESS" "HTML report generated: ${HTML_REPORT}"
    else
        print_status "WARNING" "Coverage file not found, skipping HTML report"
    fi
}

# Function to analyze coverage data
analyze_coverage() {
    print_status "INFO" "Analyzing test coverage..."

    if [[ ! -f "${COVERAGE_FILE}" ]]; then
        print_status "WARNING" "Coverage file not found, skipping analysis"
        return
    fi

    # Generate summary statistics
    {
        echo "# Test Coverage Summary Report"
        echo "Generated on: $(date)"
        echo "Project: O-RAN Intent MANO for Network Slicing"
        echo ""

        # Overall coverage
        local total_coverage
        total_coverage=$(go tool cover -func="${COVERAGE_FILE}" | tail -1 | grep -o '[0-9.]*%' || echo "0.0%")
        echo "## Overall Coverage: $total_coverage"
        echo ""

        # Coverage threshold check
        local coverage_num
        coverage_num=$(echo "$total_coverage" | sed 's/%//')
        local threshold=80.0

        if (( $(echo "$coverage_num >= $threshold" | bc -l) )); then
            echo "✅ Coverage meets target threshold ($threshold%)"
        else
            echo "⚠️  Coverage below target threshold ($threshold%)"
            echo "   Current: $coverage_num% | Target: $threshold%"
        fi
        echo ""

        # Package-wise coverage
        echo "## Package Coverage Details"
        echo ""
        go tool cover -func="${COVERAGE_FILE}" | grep -v "total:" | while read -r line; do
            local package_info
            package_info=$(echo "$line" | awk '{print $1 "\t" $3}')
            echo "$package_info"
        done
        echo ""

        # Test statistics
        echo "## Test Statistics"
        echo ""

        # Count test files
        local test_files
        test_files=$(find "${PROJECT_ROOT}" -name "*_test.go" | wc -l)
        echo "Total test files: $test_files"

        # Count test functions
        local test_functions
        test_functions=$(grep -r "func Test\|func Benchmark\|func Fuzz" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "Total test functions: $test_functions"

        # Identify packages with low coverage
        echo ""
        echo "## Packages Needing Attention (< 80% coverage)"
        echo ""
        go tool cover -func="${COVERAGE_FILE}" | awk '$3 != "total:" && $3 < "80.0%" {print $1 "\t" $3}' | head -10

    } > "${SUMMARY_REPORT}"

    print_status "SUCCESS" "Coverage analysis completed"
    print_status "INFO" "Summary report: ${SUMMARY_REPORT}"
}

# Function to run security tests
run_security_tests() {
    print_status "INFO" "Running security validation tests..."

    cd "${PROJECT_ROOT}"

    # Run security-specific tests
    if go test -run "TestSecurityValidation\|TestValidation.*Security\|Fuzz.*" ./... -timeout 60s 2>&1 | tee "${REPORT_DIR}/security_tests.log"; then
        print_status "SUCCESS" "Security tests completed"
    else
        print_status "WARNING" "Some security tests encountered issues"
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    print_status "INFO" "Running benchmark tests..."

    cd "${PROJECT_ROOT}"

    # Run benchmarks with limited time
    if timeout 120s go test -bench=. -benchtime=1s -timeout=2m ./... 2>&1 | tee "${REPORT_DIR}/benchmark_results.log"; then
        print_status "SUCCESS" "Benchmark tests completed"
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            print_status "WARNING" "Benchmark tests timed out"
        else
            print_status "WARNING" "Some benchmark tests encountered issues"
        fi
    fi
}

# Function to check test modernization status
check_modernization_status() {
    print_status "INFO" "Checking test modernization status..."

    {
        echo ""
        echo "## Test Modernization Status"
        echo ""

        # Check for modern testing patterns
        echo "### Modern Testing Patterns Usage"
        echo ""

        # Count t.Parallel() usage
        local parallel_usage
        parallel_usage=$(grep -r "t\.Parallel()" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ t.Parallel() usage: $parallel_usage tests"

        # Count t.Cleanup() usage
        local cleanup_usage
        cleanup_usage=$(grep -r "t\.Cleanup(" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ t.Cleanup() usage: $cleanup_usage tests"

        # Count testify usage
        local testify_usage
        testify_usage=$(grep -r "github\.com/stretchr/testify" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ testify usage: $testify_usage imports"

        # Count table-driven tests
        local table_tests
        table_tests=$(grep -r "tests := \[\]struct" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ Table-driven tests: $table_tests implementations"

        # Count benchmark tests
        local benchmarks
        benchmarks=$(grep -r "func Benchmark" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ Benchmark tests: $benchmarks functions"

        # Count fuzz tests
        local fuzz_tests
        fuzz_tests=$(grep -r "func Fuzz" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ Fuzz tests: $fuzz_tests functions"

        # Count test suites
        local test_suites
        test_suites=$(grep -r "suite\.Suite" "${PROJECT_ROOT}" --include="*_test.go" | wc -l)
        echo "✅ Test suites: $test_suites implementations"

        echo ""
        echo "### Modernization Completeness"
        echo ""

        # Calculate modernization score
        local total_patterns=7
        local patterns_found=0

        [[ $parallel_usage -gt 0 ]] && ((patterns_found++))
        [[ $cleanup_usage -gt 0 ]] && ((patterns_found++))
        [[ $testify_usage -gt 0 ]] && ((patterns_found++))
        [[ $table_tests -gt 0 ]] && ((patterns_found++))
        [[ $benchmarks -gt 0 ]] && ((patterns_found++))
        [[ $fuzz_tests -gt 0 ]] && ((patterns_found++))
        [[ $test_suites -gt 0 ]] && ((patterns_found++))

        local modernization_score
        modernization_score=$(echo "scale=1; $patterns_found * 100 / $total_patterns" | bc)
        echo "Modernization Score: ${modernization_score}%"

        if (( $(echo "$modernization_score >= 80" | bc -l) )); then
            echo "✅ Test modernization is comprehensive"
        else
            echo "⚠️  Test modernization needs improvement"
        fi

    } >> "${SUMMARY_REPORT}"

    print_status "SUCCESS" "Modernization status checked"
}

# Function to generate comprehensive report
generate_final_report() {
    print_status "INFO" "Generating final comprehensive report..."

    local final_report="${REPORT_DIR}/COMPREHENSIVE_TEST_REPORT.md"

    {
        echo "# Comprehensive Test Report - O-RAN Intent MANO"
        echo ""
        echo "**Generated**: $(date)"
        echo "**Report Directory**: ${REPORT_DIR}"
        echo ""

        # Include summary content
        if [[ -f "${SUMMARY_REPORT}" ]]; then
            cat "${SUMMARY_REPORT}"
        fi

        echo ""
        echo "## Available Reports"
        echo ""

        if [[ -f "${HTML_REPORT}" ]]; then
            echo "- 📊 [HTML Coverage Report]($(basename "${HTML_REPORT}"))"
        fi

        if [[ -f "${REPORT_DIR}/test_output.log" ]]; then
            echo "- 📝 [Test Output Log](test_output.log)"
        fi

        if [[ -f "${REPORT_DIR}/security_tests.log" ]]; then
            echo "- 🔒 [Security Test Results](security_tests.log)"
        fi

        if [[ -f "${REPORT_DIR}/benchmark_results.log" ]]; then
            echo "- ⚡ [Benchmark Results](benchmark_results.log)"
        fi

        echo ""
        echo "## Test Execution Summary"
        echo ""

        # Check for test failures in log
        if [[ -f "${REPORT_DIR}/test_output.log" ]]; then
            local failed_tests
            failed_tests=$(grep -c "FAIL" "${REPORT_DIR}/test_output.log" || echo "0")

            local passed_tests
            passed_tests=$(grep -c "PASS" "${REPORT_DIR}/test_output.log" || echo "0")

            echo "- Tests Passed: $passed_tests"
            echo "- Tests Failed: $failed_tests"

            if [[ $failed_tests -eq 0 ]]; then
                echo "- Status: ✅ All tests passing"
            else
                echo "- Status: ⚠️  Some tests failed"
            fi
        fi

        echo ""
        echo "## Recommendations"
        echo ""

        # Coverage-based recommendations
        if [[ -f "${COVERAGE_FILE}" ]]; then
            local total_coverage
            total_coverage=$(go tool cover -func="${COVERAGE_FILE}" | tail -1 | grep -o '[0-9.]*%' | sed 's/%//' || echo "0")

            if (( $(echo "$total_coverage < 80" | bc -l) )); then
                echo "1. **Improve Test Coverage**: Current coverage is ${total_coverage}%. Add tests for uncovered code paths."
            fi

            if (( $(echo "$total_coverage >= 80" | bc -l) )); then
                echo "1. **Maintain Coverage**: Excellent coverage at ${total_coverage}%. Continue maintaining high standards."
            fi
        fi

        echo "2. **Continue Modernization**: Leverage the new test framework utilities in \`test/framework/\`"
        echo "3. **Performance Testing**: Run benchmark tests regularly to catch performance regressions"
        echo "4. **Security Testing**: Include fuzz testing in CI/CD pipeline"

        echo ""
        echo "---"
        echo ""
        echo "*Report generated by the O-RAN Intent MANO test coverage script*"

    } > "${final_report}"

    print_status "SUCCESS" "Comprehensive report generated: ${final_report}"
}

# Function to display final summary
display_summary() {
    echo ""
    echo -e "${BLUE}📊 Test Coverage Report Summary${NC}"
    echo -e "${BLUE}==============================${NC}"

    if [[ -f "${COVERAGE_FILE}" ]]; then
        local total_coverage
        total_coverage=$(go tool cover -func="${COVERAGE_FILE}" | tail -1 | grep -o '[0-9.]*%' || echo "0.0%")
        echo -e "Overall Coverage: ${GREEN}$total_coverage${NC}"
    fi

    echo ""
    echo "📁 Reports available in: ${REPORT_DIR}"

    if [[ -f "${HTML_REPORT}" ]]; then
        echo "🌐 Open HTML report: file://${HTML_REPORT}"
    fi

    if [[ -f "${REPORT_DIR}/COMPREHENSIVE_TEST_REPORT.md" ]]; then
        echo "📄 Comprehensive report: ${REPORT_DIR}/COMPREHENSIVE_TEST_REPORT.md"
    fi

    echo ""
    print_status "SUCCESS" "Test coverage analysis completed!"
}

# Main execution
main() {
    local start_time
    start_time=$(date +%s)

    check_prerequisites
    run_basic_tests
    generate_html_report
    analyze_coverage
    run_security_tests
    run_benchmark_tests
    check_modernization_status
    generate_final_report
    display_summary

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    print_status "INFO" "Total execution time: ${duration} seconds"
}

# Run main function
main "$@"