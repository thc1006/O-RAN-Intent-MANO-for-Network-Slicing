#!/bin/bash

# Test script for security-check.sh validation
# This script runs a lightweight version of security checks for testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

log() {
    echo -e "${BLUE}[TEST]${NC} $*"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $*"
}

error() {
    echo -e "${RED}[FAIL]${NC} $*"
}

warning() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Test 1: Check if Go files exist
test_go_files() {
    log "Testing Go file detection..."

    local go_files
    go_files=$(find "$PROJECT_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" | wc -l)

    if [[ $go_files -gt 0 ]]; then
        success "Found $go_files Go files for security analysis"
        return 0
    else
        error "No Go files found"
        return 1
    fi
}

# Test 2: Check Kubernetes manifests
test_k8s_manifests() {
    log "Testing Kubernetes manifest detection..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -not -path "*/.git/*" | grep -E "(k8s|kubernetes|deploy|chart)" | wc -l)

    if [[ $k8s_files -gt 0 ]]; then
        success "Found $k8s_files Kubernetes manifest files"
        return 0
    else
        error "No Kubernetes manifest files found"
        return 1
    fi
}

# Test 3: Check NetworkPolicy presence
test_network_policies() {
    log "Testing NetworkPolicy detection..."

    local netpol_files
    netpol_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: NetworkPolicy" {} \; 2>/dev/null | wc -l)

    if [[ $netpol_files -gt 0 ]]; then
        success "Found $netpol_files files with NetworkPolicy resources"

        # Check for specific network policies
        if [[ -f "$PROJECT_ROOT/deploy/k8s/base/network-policies.yaml" ]]; then
            success "Main network policies file exists"
        fi

        if [[ -f "$PROJECT_ROOT/deploy/k8s/base/networkpolicy-orchestrator.yaml" ]]; then
            success "Orchestrator network policy exists"
        fi

        return 0
    else
        error "No NetworkPolicy resources found"
        return 1
    fi
}

# Test 4: Check container images in manifests
test_container_images() {
    log "Testing container image detection..."

    local image_count=0
    local secure_images=0
    local insecure_images=0

    while IFS= read -r k8s_file; do
        if [[ -f "$k8s_file" ]]; then
            # Extract image references
            local images
            images=$(grep -o 'image: .*' "$k8s_file" 2>/dev/null | sed 's/image: //' | tr -d '"' || true)

            while IFS= read -r image; do
                if [[ -n "$image" ]]; then
                    image_count=$((image_count + 1))

                    # Check image format
                    if [[ "$image" =~ @sha256:[a-f0-9]{64}$ ]]; then
                        secure_images=$((secure_images + 1))
                    elif [[ "$image" =~ :[a-zA-Z0-9_.-]+$ ]] && [[ ! "$image" =~ :latest$ ]]; then
                        secure_images=$((secure_images + 1))
                    else
                        insecure_images=$((insecure_images + 1))
                        warning "Potentially insecure image: $image"
                    fi
                fi
            done <<< "$images"
        fi
    done < <(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "image:" {} \; 2>/dev/null || true)

    if [[ $image_count -gt 0 ]]; then
        success "Found $image_count container images ($secure_images secure, $insecure_images potentially insecure)"
        return 0
    else
        warning "No container images found in manifests"
        return 0
    fi
}

# Test 5: Check Service Account configurations
test_service_accounts() {
    log "Testing Service Account detection..."

    local sa_files
    sa_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: ServiceAccount\|serviceAccount:" {} \; 2>/dev/null | wc -l || echo "0")

    if [[ $sa_files -gt 0 ]]; then
        success "Found $sa_files files with Service Account configurations"
        return 0
    else
        warning "No Service Account configurations found"
        return 0
    fi
}

# Test 6: Validate security-check.sh script syntax
test_script_syntax() {
    log "Testing security-check.sh script syntax..."

    if bash -n "$SCRIPT_DIR/security-check.sh"; then
        success "security-check.sh has valid syntax"
        return 0
    else
        error "security-check.sh has syntax errors"
        return 1
    fi
}

# Test 7: Check required tools availability (basic check)
test_basic_tools() {
    log "Testing basic tool availability..."

    local tools_available=0
    local tools_missing=0

    # Check Go
    if command -v go &> /dev/null; then
        success "Go is available"
        tools_available=$((tools_available + 1))
    else
        error "Go is not available"
        tools_missing=$((tools_missing + 1))
    fi

    # Check Python (try different names)
    if command -v python3 &> /dev/null || command -v python &> /dev/null; then
        success "Python is available"
        tools_available=$((tools_available + 1))
    else
        warning "Python is not available - some features may not work"
        tools_missing=$((tools_missing + 1))
    fi

    # Check jq
    if command -v jq &> /dev/null; then
        success "jq is available"
        tools_available=$((tools_available + 1))
    else
        warning "jq is not available - JSON processing may not work"
        tools_missing=$((tools_missing + 1))
    fi

    log "Tools summary: $tools_available available, $tools_missing missing"
    return 0
}

# Main test execution
main() {
    log "Starting security-check.sh validation tests..."
    log "Project root: $PROJECT_ROOT"
    echo

    local passed_tests=0
    local failed_tests=0

    # Run all tests
    local tests=(
        "test_go_files"
        "test_k8s_manifests"
        "test_network_policies"
        "test_container_images"
        "test_service_accounts"
        "test_script_syntax"
        "test_basic_tools"
    )

    for test_func in "${tests[@]}"; do
        echo
        if $test_func; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done

    echo
    echo "==================== TEST SUMMARY ===================="
    echo "Total tests: $((passed_tests + failed_tests))"
    echo "Passed: $passed_tests"
    echo "Failed: $failed_tests"
    echo "======================================================="

    if [[ $failed_tests -eq 0 ]]; then
        success "All tests passed! Security check script should work correctly."
        return 0
    else
        error "$failed_tests test(s) failed. Please review the issues above."
        return 1
    fi
}

# Run tests
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi