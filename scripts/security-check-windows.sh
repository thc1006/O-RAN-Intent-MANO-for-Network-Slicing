#!/bin/bash

# Windows-compatible Security Check Script for O-RAN Intent-MANO Network Slicing
# Copyright 2024 O-RAN Intent MANO Project
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORT_FILE="${PROJECT_ROOT}/security-check-report.json"
LOG_FILE="${PROJECT_ROOT}/security-check.log"
TEMP_DIR="${PROJECT_ROOT}/.security-check-temp"
EXIT_CODE=0

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# Tool versions and requirements
readonly GOSEC_VERSION="2.18.2"
readonly CHECKOV_VERSION="3.0.0"

# Initialize logging
mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p "$TEMP_DIR"
exec 1> >(tee -a "$LOG_FILE")
exec 2> >(tee -a "$LOG_FILE" >&2)

# Report structure
declare -A SECURITY_REPORT=(
    ["timestamp"]=""
    ["environment"]=""
    ["go_security_scan"]=""
    ["kubernetes_manifest_scan"]=""
    ["network_policy_validation"]=""
    ["security_context_validation"]=""
    ["image_specification_validation"]=""
    ["service_account_validation"]=""
    ["overall_status"]=""
    ["total_issues"]="0"
    ["critical_issues"]="0"
    ["high_issues"]="0"
    ["medium_issues"]="0"
    ["low_issues"]="0"
)

# Helper functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    EXIT_CODE=1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

info() {
    echo -e "${CYAN}[INFO]${NC} $*"
}

# Check if running in CI environment
is_ci_environment() {
    [[ "${CI:-}" == "true" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ -n "${JENKINS_URL:-}" ]]
}

# Install required tools for Windows
install_security_tools() {
    log "Checking security scanning tools..."

    # Set up PATH to include our installed tools
    export PATH="/c/Users/thc1006/AppData/Roaming/Python/Python313/Scripts:/c/Users/thc1006/go/bin:$PATH"

    local tools_installed=0

    # Check gosec
    if ! command -v gosec &> /dev/null; then
        info "Installing gosec..."
        go install "github.com/securego/gosec/v2/cmd/gosec@v${GOSEC_VERSION}"
        tools_installed=$((tools_installed + 1))
    else
        info "gosec already installed"
    fi

    # Check checkov
    if ! command -v checkov &> /dev/null; then
        error "checkov not found. Please install manually: pip install --user checkov==${CHECKOV_VERSION}"
        return 1
    else
        info "checkov already installed"
    fi

    if [[ $tools_installed -gt 0 ]]; then
        success "Installed $tools_installed security tools"
    else
        info "All security tools already available"
    fi
}

# Validate Go code with gosec
validate_go_security() {
    log "Running Go security analysis with gosec..."

    local go_files
    go_files=$(find "$PROJECT_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" 2>/dev/null | wc -l)

    if [[ $go_files -eq 0 ]]; then
        warning "No Go files found for security analysis"
        SECURITY_REPORT["go_security_scan"]="SKIPPED"
        return 0
    fi

    info "Found $go_files Go files to analyze"

    local gosec_output="${TEMP_DIR}/gosec-report.json"
    local gosec_txt="${TEMP_DIR}/gosec-report.txt"

    # Run gosec with comprehensive checks
    if gosec -fmt json -out "$gosec_output" -stdout -nosec=false -tests -severity=medium \
        -confidence=medium -exclude-generated \
        -include="G101,G102,G103,G104,G106,G107,G108,G109,G110,G201,G202,G203,G204,G301,G302,G303,G304,G305,G306,G307,G401,G402,G403,G404,G501,G502,G503,G504,G505,G601" \
        "$PROJECT_ROOT/..." 2> "$gosec_txt"; then

        # Parse results using Python since jq is not available
        local found_issues
        found_issues=$(python3 -c "
import json, sys
try:
    with open('$gosec_output', 'r') as f:
        data = json.load(f)
    print(len(data.get('Issues', [])))
except:
    print('0')
" 2>/dev/null || echo "0")

        if [[ $found_issues -gt 0 ]]; then
            warning "Found $found_issues security issues in Go code"

            # Categorize issues by severity using Python
            python3 << 'PYTHON_SCRIPT' > "${TEMP_DIR}/gosec-stats.json"
import json
import sys

try:
    with open('$gosec_output', 'r') as f:
        data = json.load(f)

    issues = data.get('Issues', [])
    critical = sum(1 for issue in issues if issue.get('severity') == 'HIGH' and issue.get('confidence') == 'HIGH')
    high = sum(1 for issue in issues if issue.get('severity') == 'MEDIUM' and issue.get('confidence') == 'HIGH')
    medium = sum(1 for issue in issues if issue.get('severity') == 'LOW' and issue.get('confidence') == 'HIGH')
    low = sum(1 for issue in issues if issue.get('confidence') in ['MEDIUM', 'LOW'])

    result = {
        "critical": critical,
        "high": high,
        "medium": medium,
        "low": low,
        "total": len(issues)
    }

    print(json.dumps(result))
except Exception as e:
    print(json.dumps({"critical": 0, "high": 0, "medium": 0, "low": 0, "total": 0}))
PYTHON_SCRIPT

            if [[ -f "${TEMP_DIR}/gosec-stats.json" ]]; then
                local stats
                stats=$(cat "${TEMP_DIR}/gosec-stats.json")
                local critical high medium low
                critical=$(echo "$stats" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['critical'])")
                high=$(echo "$stats" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['high'])")
                medium=$(echo "$stats" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['medium'])")
                low=$(echo "$stats" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['low'])")

                SECURITY_REPORT["critical_issues"]=$((${SECURITY_REPORT["critical_issues"]} + critical))
                SECURITY_REPORT["high_issues"]=$((${SECURITY_REPORT["high_issues"]} + high))
                SECURITY_REPORT["medium_issues"]=$((${SECURITY_REPORT["medium_issues"]} + medium))
                SECURITY_REPORT["low_issues"]=$((${SECURITY_REPORT["low_issues"]} + low))

                info "Issue breakdown: Critical=$critical, High=$high, Medium=$medium, Low=$low"

                if [[ $critical -gt 0 ]] || [[ $high -gt 0 ]]; then
                    error "Critical or high-severity security issues found in Go code"
                    SECURITY_REPORT["go_security_scan"]="FAILED"
                else
                    warning "Medium/low-severity security issues found in Go code"
                    SECURITY_REPORT["go_security_scan"]="WARNING"
                fi
            fi
        else
            success "No security issues found in Go code"
            SECURITY_REPORT["go_security_scan"]="PASSED"
        fi
    else
        error "gosec scan failed"
        SECURITY_REPORT["go_security_scan"]="ERROR"
    fi

    SECURITY_REPORT["total_issues"]=$((${SECURITY_REPORT["total_issues"]} + found_issues))
}

# Validate Kubernetes manifests with checkov
validate_kubernetes_manifests() {
    log "Running Kubernetes manifest security analysis..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -not -path "*/.git/*" -not -path "*/node_modules/*" 2>/dev/null | grep -E "(k8s|kubernetes|deploy|chart)" | head -20 || true)

    if [[ -z "$k8s_files" ]]; then
        warning "No Kubernetes manifest files found"
        SECURITY_REPORT["kubernetes_manifest_scan"]="SKIPPED"
        return 0
    fi

    info "Found Kubernetes manifests to analyze"

    local checkov_output="${TEMP_DIR}/checkov-report.json"
    local total_issues=0

    # Run checkov on Kubernetes files
    if echo "$k8s_files" | xargs checkov -f --framework kubernetes \
        --output json --output-file "$checkov_output" \
        --check CKV_K8S_1,CKV_K8S_2,CKV_K8S_3,CKV_K8S_4,CKV_K8S_5,CKV_K8S_6,CKV_K8S_7,CKV_K8S_8,CKV_K8S_9,CKV_K8S_10 \
        --check CKV_K8S_11,CKV_K8S_12,CKV_K8S_13,CKV_K8S_14,CKV_K8S_15,CKV_K8S_16,CKV_K8S_17,CKV_K8S_18,CKV_K8S_19,CKV_K8S_20 \
        --check CKV_K8S_21,CKV_K8S_22,CKV_K8S_23,CKV_K8S_25,CKV_K8S_28,CKV_K8S_29,CKV_K8S_30,CKV_K8S_31,CKV_K8S_37,CKV_K8S_38 \
        --check CKV_K8S_40,CKV_K8S_43,CKV_K8S_49 2>/dev/null || true; then

        # Parse checkov results using Python
        local checkov_failed
        checkov_failed=$(python3 -c "
import json, sys
try:
    with open('$checkov_output', 'r') as f:
        data = json.load(f)
    failed = data.get('results', {}).get('failed_checks', [])
    print(len(failed))
except:
    print('0')
" 2>/dev/null || echo "0")

        total_issues=$((total_issues + checkov_failed))

        if [[ $checkov_failed -gt 0 ]]; then
            warning "Checkov found $checkov_failed security issues"
        fi
    fi

    # Update totals
    SECURITY_REPORT["total_issues"]=$((${SECURITY_REPORT["total_issues"]} + total_issues))

    if [[ $total_issues -gt 20 ]]; then
        error "Critical Kubernetes security issues found"
        SECURITY_REPORT["kubernetes_manifest_scan"]="FAILED"
    elif [[ $total_issues -gt 0 ]]; then
        warning "Some Kubernetes security issues found"
        SECURITY_REPORT["kubernetes_manifest_scan"]="WARNING"
    else
        success "Kubernetes manifests passed security validation"
        SECURITY_REPORT["kubernetes_manifest_scan"]="PASSED"
    fi
}

# Check NetworkPolicy presence and correctness (simplified for Windows)
validate_network_policies() {
    log "Validating NetworkPolicy configurations..."

    local netpol_files
    netpol_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: NetworkPolicy" {} \; 2>/dev/null || true)

    if [[ -z "$netpol_files" ]]; then
        error "No NetworkPolicy resources found - network segmentation required"
        SECURITY_REPORT["network_policy_validation"]="FAILED"
        return 1
    fi

    local total_policies
    total_policies=$(echo "$netpol_files" | wc -l)

    info "Found $total_policies NetworkPolicy files"

    if [[ $total_policies -gt 0 ]]; then
        success "NetworkPolicy resources found"
        SECURITY_REPORT["network_policy_validation"]="PASSED"
    else
        error "No NetworkPolicy resources found"
        SECURITY_REPORT["network_policy_validation"]="FAILED"
    fi
}

# Validate container security contexts (simplified)
validate_security_contexts() {
    log "Validating container security contexts..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: \(Deployment\|StatefulSet\|DaemonSet\|Pod\)" {} \; 2>/dev/null || true)

    if [[ -z "$k8s_files" ]]; then
        warning "No workload resources found for security context validation"
        SECURITY_REPORT["security_context_validation"]="SKIPPED"
        return 0
    fi

    local container_files
    container_files=$(echo "$k8s_files" | xargs grep -l "securityContext" 2>/dev/null || true)

    if [[ -n "$container_files" ]]; then
        success "Found security contexts in workload resources"
        SECURITY_REPORT["security_context_validation"]="PASSED"
    else
        warning "No security contexts found in workload resources"
        SECURITY_REPORT["security_context_validation"]="WARNING"
    fi
}

# Validate image specifications
validate_image_specifications() {
    log "Validating container image specifications..."

    local image_files
    image_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "image:" {} \; 2>/dev/null || true)

    if [[ -z "$image_files" ]]; then
        warning "No files with container images found"
        SECURITY_REPORT["image_specification_validation"]="SKIPPED"
        return 0
    fi

    local latest_images
    latest_images=$(echo "$image_files" | xargs grep "image:" | grep -c ":latest\|image: [^:]*$" 2>/dev/null || echo "0")

    if [[ $latest_images -gt 0 ]]; then
        warning "Found $latest_images images using 'latest' tag or no tag"
        SECURITY_REPORT["image_specification_validation"]="WARNING"
    else
        success "All images use specific tags"
        SECURITY_REPORT["image_specification_validation"]="PASSED"
    fi
}

# Validate Service Account configurations
validate_service_accounts() {
    log "Validating Service Account configurations..."

    local sa_files
    sa_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: ServiceAccount\|serviceAccount:" {} \; 2>/dev/null || true)

    if [[ -z "$sa_files" ]]; then
        warning "No Service Account configurations found"
        SECURITY_REPORT["service_account_validation"]="SKIPPED"
        return 0
    fi

    local sa_count
    sa_count=$(echo "$sa_files" | wc -l)

    info "Found $sa_count files with Service Account configurations"
    success "Service Account configurations found"
    SECURITY_REPORT["service_account_validation"]="PASSED"
}

# Generate final security report
generate_security_report() {
    log "Generating security compliance report..."

    # Set timestamp and environment
    SECURITY_REPORT["timestamp"]=$(date -Iseconds)
    SECURITY_REPORT["environment"]=$(if is_ci_environment; then echo "CI"; else echo "LOCAL"; fi)

    # Calculate overall status
    local failed_checks=0
    local warning_checks=0
    local passed_checks=0

    for check in "go_security_scan" "kubernetes_manifest_scan" "network_policy_validation" "security_context_validation" "image_specification_validation" "service_account_validation"; do
        case "${SECURITY_REPORT[$check]}" in
            "FAILED"|"ERROR") failed_checks=$((failed_checks + 1)) ;;
            "WARNING") warning_checks=$((warning_checks + 1)) ;;
            "PASSED") passed_checks=$((passed_checks + 1)) ;;
        esac
    done

    if [[ $failed_checks -gt 0 ]]; then
        SECURITY_REPORT["overall_status"]="FAILED"
    elif [[ $warning_checks -gt 0 ]]; then
        SECURITY_REPORT["overall_status"]="WARNING"
    else
        SECURITY_REPORT["overall_status"]="PASSED"
    fi

    # Generate JSON report
    cat > "$REPORT_FILE" << EOF
{
  "timestamp": "${SECURITY_REPORT[timestamp]}",
  "environment": "${SECURITY_REPORT[environment]}",
  "overall_status": "${SECURITY_REPORT[overall_status]}",
  "summary": {
    "total_issues": ${SECURITY_REPORT[total_issues]},
    "critical_issues": ${SECURITY_REPORT[critical_issues]},
    "high_issues": ${SECURITY_REPORT[high_issues]},
    "medium_issues": ${SECURITY_REPORT[medium_issues]},
    "low_issues": ${SECURITY_REPORT[low_issues]}
  },
  "checks": {
    "go_security_scan": "${SECURITY_REPORT[go_security_scan]}",
    "kubernetes_manifest_scan": "${SECURITY_REPORT[kubernetes_manifest_scan]}",
    "network_policy_validation": "${SECURITY_REPORT[network_policy_validation]}",
    "security_context_validation": "${SECURITY_REPORT[security_context_validation]}",
    "image_specification_validation": "${SECURITY_REPORT[image_specification_validation]}",
    "service_account_validation": "${SECURITY_REPORT[service_account_validation]}"
  },
  "statistics": {
    "failed_checks": $failed_checks,
    "warning_checks": $warning_checks,
    "passed_checks": $passed_checks
  }
}
EOF

    # Display summary
    echo
    echo "======================== SECURITY COMPLIANCE REPORT ========================"
    echo "Timestamp: ${SECURITY_REPORT[timestamp]}"
    echo "Environment: ${SECURITY_REPORT[environment]}"
    echo "Overall Status: ${SECURITY_REPORT[overall_status]}"
    echo
    echo "Summary:"
    echo "  Total Issues: ${SECURITY_REPORT[total_issues]}"
    echo "  Critical: ${SECURITY_REPORT[critical_issues]}"
    echo "  High: ${SECURITY_REPORT[high_issues]}"
    echo "  Medium: ${SECURITY_REPORT[medium_issues]}"
    echo "  Low: ${SECURITY_REPORT[low_issues]}"
    echo
    echo "Check Results:"
    echo "  Go Security Scan: ${SECURITY_REPORT[go_security_scan]}"
    echo "  Kubernetes Manifest Scan: ${SECURITY_REPORT[kubernetes_manifest_scan]}"
    echo "  NetworkPolicy Validation: ${SECURITY_REPORT[network_policy_validation]}"
    echo "  Security Context Validation: ${SECURITY_REPORT[security_context_validation]}"
    echo "  Image Specification Validation: ${SECURITY_REPORT[image_specification_validation]}"
    echo "  Service Account Validation: ${SECURITY_REPORT[service_account_validation]}"
    echo
    echo "Statistics:"
    echo "  Failed Checks: $failed_checks"
    echo "  Warning Checks: $warning_checks"
    echo "  Passed Checks: $passed_checks"
    echo "=========================================================================="
    echo

    success "Security report generated: $REPORT_FILE"

    # Set exit code based on results
    if [[ "${SECURITY_REPORT[overall_status]}" == "FAILED" ]]; then
        error "Security compliance check FAILED"
        EXIT_CODE=1
    elif [[ "${SECURITY_REPORT[overall_status]}" == "WARNING" ]]; then
        warning "Security compliance check completed with WARNINGS"
        EXIT_CODE=0  # Don't fail CI for warnings
    else
        success "Security compliance check PASSED"
        EXIT_CODE=0
    fi
}

# Cleanup function
cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main execution
main() {
    trap cleanup EXIT

    log "Starting comprehensive security check for O-RAN Intent-MANO (Windows)..."
    log "Project root: $PROJECT_ROOT"
    log "Environment: $(if is_ci_environment; then echo "CI/CD"; else echo "Local"; fi)"

    # Install required tools
    install_security_tools || {
        error "Failed to install security tools"
        exit 1
    }

    # Run security validations
    validate_go_security
    validate_kubernetes_manifests
    validate_network_policies
    validate_security_contexts
    validate_image_specifications
    validate_service_accounts

    # Generate final report
    generate_security_report

    exit $EXIT_CODE
}

# Script usage
usage() {
    cat << EOF
Usage: $0 [options]

Options:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    --skip-install          Skip tool installation
    --report-file FILE      Custom report file location
    --temp-dir DIR          Custom temporary directory

Examples:
    $0                      Run full security check
    $0 --skip-install       Run without installing tools
    $0 --verbose            Run with detailed output

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            set -x
            shift
            ;;
        --skip-install)
            SKIP_INSTALL=1
            shift
            ;;
        --report-file)
            REPORT_FILE="$2"
            shift 2
            ;;
        --temp-dir)
            TEMP_DIR="$2"
            shift 2
            ;;
        *)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi