#!/bin/bash

# Kubernetes Security Validation Script
# Validates security configurations in O-RAN MANO deployments
# Author: Generated for O-RAN Intent-Based MANO project
# Date: 2025-09-24

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
K8S_BASE_DIR="$PROJECT_ROOT/deploy/k8s/base"

# Colors for output (disabled for compatibility)
RED=''
GREEN=''
YELLOW=''
BLUE=''
NC='' # No Color

# Counters
PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASS_COUNT++))
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARN_COUNT++))
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAIL_COUNT++))
}

# Check if file exists
check_file_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        fail "File not found: $file"
        return 1
    fi
    return 0
}

# Validate image uses SHA256 digest
validate_image_digest() {
    local file="$1"
    local component="$2"

    log "Validating image digest for $component in $file"

    if ! check_file_exists "$file"; then
        return 1
    fi

    # Check for SHA256 digest in image reference
    if grep -q "image:.*@sha256:" "$file"; then
        pass "$component: Image uses SHA256 digest"
    else
        fail "$component: Image does not use SHA256 digest"
    fi

    # Check that no 'latest' tags are used
    if grep -q "image:.*:latest" "$file"; then
        fail "$component: Image uses 'latest' tag (security risk)"
    else
        pass "$component: No 'latest' tags found"
    fi
}

# Validate seccomp profile configuration
validate_seccomp_profile() {
    local file="$1"
    local component="$2"

    log "Validating seccomp profile for $component in $file"

    if ! check_file_exists "$file"; then
        return 1
    fi

    # Check for modern seccompProfile in securityContext
    if grep -A 2 "seccompProfile:" "$file" | grep -q "type: RuntimeDefault"; then
        pass "$component: Uses modern seccompProfile configuration"
    else
        fail "$component: Missing or incorrect seccompProfile configuration"
    fi

    # Check for deprecated seccomp annotations
    if grep -q "seccomp.security.alpha.kubernetes.io/pod:" "$file"; then
        if grep -q "# Legacy seccomp annotation (deprecated)" "$file"; then
            warn "$component: Contains legacy seccomp annotation but marked as deprecated"
        else
            fail "$component: Uses deprecated seccomp annotation without deprecation notice"
        fi
    fi
}

# Validate service account token configuration
validate_service_account_token() {
    local file="$1"
    local component="$2"

    log "Validating service account token for $component in $file"

    if ! check_file_exists "$file"; then
        return 1
    fi

    # Check automountServiceAccountToken setting
    if grep -q "automountServiceAccountToken: false" "$file"; then
        pass "$component: Service account token auto-mounting is disabled"
    elif grep -q "automountServiceAccountToken: true" "$file"; then
        # Check if there's proper justification
        if grep -A 10 "automountServiceAccountToken: true" "$file" | grep -q "SECURITY EXCEPTION JUSTIFICATION"; then
            pass "$component: Service account token auto-mounting enabled with proper justification"
        else
            fail "$component: Service account token auto-mounting enabled without justification"
        fi
    else
        warn "$component: automountServiceAccountToken not explicitly set (defaults to true)"
    fi
}

# Validate security context
validate_security_context() {
    local file="$1"
    local component="$2"

    log "Validating security context for $component in $file"

    if ! check_file_exists "$file"; then
        return 1
    fi

    # Check runAsNonRoot
    if grep -q "runAsNonRoot: true" "$file"; then
        pass "$component: Runs as non-root user"
    else
        fail "$component: Missing runAsNonRoot: true"
    fi

    # Check readOnlyRootFilesystem
    if grep -q "readOnlyRootFilesystem: true" "$file"; then
        pass "$component: Uses read-only root filesystem"
    else
        fail "$component: Missing readOnlyRootFilesystem: true"
    fi

    # Check allowPrivilegeEscalation
    if grep -q "allowPrivilegeEscalation: false" "$file"; then
        pass "$component: Privilege escalation is disabled"
    else
        fail "$component: Missing allowPrivilegeEscalation: false"
    fi

    # Check capabilities are dropped
    if grep -A 3 "capabilities:" "$file" | grep -q "drop:" && grep -A 5 "drop:" "$file" | grep -q "ALL"; then
        pass "$component: All capabilities are dropped"
    else
        fail "$component: Capabilities not properly dropped"
    fi
}

# Validate NetworkPolicy exists and is referenced
validate_network_policy() {
    local deployment_file="$1"
    local policy_file="$2"
    local component="$3"

    log "Validating NetworkPolicy for $component"

    if ! check_file_exists "$deployment_file" || ! check_file_exists "$policy_file"; then
        return 1
    fi

    # Check NetworkPolicy exists
    if grep -q "kind: NetworkPolicy" "$policy_file"; then
        pass "$component: NetworkPolicy exists"
    else
        fail "$component: NetworkPolicy file does not contain NetworkPolicy"
    fi

    # Check deployment references NetworkPolicy
    if grep -q "security.kubernetes.io/network-policy:" "$deployment_file"; then
        pass "$component: Deployment references NetworkPolicy in annotations"
    else
        warn "$component: Deployment does not reference NetworkPolicy in annotations"
    fi

    # Check NetworkPolicy has both Ingress and Egress rules
    if grep -A 5 "policyTypes:" "$policy_file" | grep -q "Ingress" && grep -A 5 "policyTypes:" "$policy_file" | grep -q "Egress"; then
        pass "$component: NetworkPolicy includes both Ingress and Egress rules"
    else
        fail "$component: NetworkPolicy missing Ingress or Egress rules"
    fi
}

# Validate resource limits and requests
validate_resources() {
    local file="$1"
    local component="$2"

    log "Validating resource limits for $component in $file"

    if ! check_file_exists "$file"; then
        return 1
    fi

    # Check CPU limits
    if grep -A 5 "limits:" "$file" | grep -q "cpu:"; then
        pass "$component: CPU limits are set"
    else
        fail "$component: Missing CPU limits"
    fi

    # Check memory limits
    if grep -A 5 "limits:" "$file" | grep -q "memory:"; then
        pass "$component: Memory limits are set"
    else
        fail "$component: Missing memory limits"
    fi

    # Check ephemeral-storage limits
    if grep -A 5 "limits:" "$file" | grep -q "ephemeral-storage:"; then
        pass "$component: Ephemeral storage limits are set"
    else
        warn "$component: Ephemeral storage limits not set"
    fi

    # Check requests are set
    if grep -A 5 "requests:" "$file" | grep -q "cpu:" && grep -A 5 "requests:" "$file" | grep -q "memory:"; then
        pass "$component: Resource requests are set"
    else
        fail "$component: Missing resource requests"
    fi
}

# Main validation function
validate_component() {
    local deployment_file="$1"
    local policy_file="$2"
    local component="$3"

    echo
    log "=== Validating $component ==="

    validate_image_digest "$deployment_file" "$component"
    validate_seccomp_profile "$deployment_file" "$component"
    validate_service_account_token "$deployment_file" "$component"
    validate_security_context "$deployment_file" "$component"
    validate_network_policy "$deployment_file" "$policy_file" "$component"
    validate_resources "$deployment_file" "$component"
}

# Print summary
print_summary() {
    echo
    echo "=== SECURITY VALIDATION SUMMARY ==="
    echo -e "${GREEN}PASSED: $PASS_COUNT${NC}"
    echo -e "${YELLOW}WARNINGS: $WARN_COUNT${NC}"
    echo -e "${RED}FAILED: $FAIL_COUNT${NC}"
    echo

    if [[ $FAIL_COUNT -eq 0 ]]; then
        echo -e "${GREEN}✓ All critical security checks passed!${NC}"
        if [[ $WARN_COUNT -gt 0 ]]; then
            echo -e "${YELLOW}⚠ Please review warnings for best practices${NC}"
        fi
        return 0
    else
        echo -e "${RED}✗ Security validation failed with $FAIL_COUNT critical issues${NC}"
        echo "Please fix the failed checks before deploying to production."
        return 1
    fi
}

# Main execution
main() {
    log "Starting Kubernetes Security Validation"
    log "Project Root: $PROJECT_ROOT"
    log "K8S Base Directory: $K8S_BASE_DIR"

    # Validate orchestrator
    validate_component \
        "$K8S_BASE_DIR/orchestrator.yaml" \
        "$K8S_BASE_DIR/networkpolicy-orchestrator.yaml" \
        "Orchestrator"

    # Validate VNF operator
    validate_component \
        "$K8S_BASE_DIR/vnf-operator.yaml" \
        "$K8S_BASE_DIR/networkpolicy-vnf-operator.yaml" \
        "VNF-Operator"

    print_summary
}

# Run main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi