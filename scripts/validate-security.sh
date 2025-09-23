#!/bin/bash

# O-RAN MANO Security Validation Script
# This script validates the security configurations in the Kubernetes manifests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy/k8s/base"

echo "🔒 O-RAN MANO Security Validation"
echo "=================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0
WARNINGS=0

# Function to check if a security requirement is met
check_security() {
    local description="$1"
    local file="$2"
    local pattern="$3"
    local required="${4:-true}"

    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $description"
        ((PASSED++))
        return 0
    else
        if [ "$required" = "true" ]; then
            echo -e "${RED}✗${NC} $description"
            ((FAILED++))
        else
            echo -e "${YELLOW}⚠${NC} $description (optional)"
            ((WARNINGS++))
        fi
        return 1
    fi
}

echo "Validating Orchestrator Security..."
echo "-----------------------------------"

# Check Pod Security Standards
check_security "Pod Security Standards enforced" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "pod-security.kubernetes.io/enforce: restricted"

# Check SecurityContext
check_security "Non-root user configured" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "runAsNonRoot: true"

check_security "Specific user ID set" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "runAsUser: 65532"

check_security "Read-only root filesystem" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "readOnlyRootFilesystem: true"

check_security "Privilege escalation disabled" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "allowPrivilegeEscalation: false"

check_security "All capabilities dropped" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "drop:.*- ALL"

check_security "Seccomp profile set" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "seccompProfile:.*type: RuntimeDefault"

# Check Image Security
check_security "Image uses specific version with digest" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "image:.*@sha256:"

check_security "ImagePullPolicy set to Always" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "imagePullPolicy: Always"

# Check Service Account
check_security "Service account token mounting disabled" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "automountServiceAccountToken: false"

# Check Resource Limits
check_security "CPU limits defined" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "limits:.*cpu:"

check_security "Memory limits defined" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "limits:.*memory:"

check_security "Ephemeral storage limits defined" \
    "$DEPLOY_DIR/orchestrator.yaml" \
    "ephemeral-storage:"

echo
echo "Validating VNF Operator Security..."
echo "-----------------------------------"

# Check VNF Operator security configurations
check_security "VNF Operator Pod Security Standards" \
    "$DEPLOY_DIR/vnf-operator.yaml" \
    "pod-security.kubernetes.io/enforce: restricted"

check_security "VNF Operator non-root user" \
    "$DEPLOY_DIR/vnf-operator.yaml" \
    "runAsUser: 65532"

check_security "VNF Operator image with digest" \
    "$DEPLOY_DIR/vnf-operator.yaml" \
    "image:.*@sha256:"

echo
echo "Validating Network Policies..."
echo "------------------------------"

# Check NetworkPolicy existence and configuration
check_security "Orchestrator NetworkPolicy exists" \
    "$DEPLOY_DIR/network-policies.yaml" \
    "name: oran-orchestrator-netpol"

check_security "VNF Operator NetworkPolicy exists" \
    "$DEPLOY_DIR/network-policies.yaml" \
    "name: oran-vnf-operator-netpol"

check_security "Default deny-all policy exists" \
    "$DEPLOY_DIR/network-policies.yaml" \
    "name: default-deny-all"

check_security "DNS egress restrictions configured" \
    "$DEPLOY_DIR/network-policies.yaml" \
    "k8s-app: kube-dns"

echo
echo "Validating RBAC Configuration..."
echo "--------------------------------"

# Check RBAC configurations
check_security "Orchestrator ServiceAccount token disabled" \
    "$DEPLOY_DIR/rbac.yaml" \
    "name: oran-orchestrator" -A 5 | grep -q "automountServiceAccountToken: false"

check_security "VNF Operator ServiceAccount token disabled" \
    "$DEPLOY_DIR/rbac.yaml" \
    "name: oran-vnf-operator" -A 5 | grep -q "automountServiceAccountToken: false"

check_security "Orchestrator ClusterRole uses least privilege" \
    "$DEPLOY_DIR/rbac.yaml" \
    "# Read-only access to nodes"

check_security "VNF Operator ClusterRole documented" \
    "$DEPLOY_DIR/rbac.yaml" \
    "# Pod management for VNF lifecycle"

echo
echo "Validating Namespace Security..."
echo "--------------------------------"

# Check namespace security configurations
check_security "Namespace Pod Security Standards" \
    "$DEPLOY_DIR/namespace.yaml" \
    "pod-security.kubernetes.io/enforce: restricted"

check_security "Namespace seccomp annotations" \
    "$DEPLOY_DIR/namespace.yaml" \
    "seccomp.security.alpha.kubernetes.io/defaultProfileName"

echo
echo "Validating Security Policies..."
echo "-------------------------------"

if [[ -f "$DEPLOY_DIR/security-policies.yaml" ]]; then
    check_security "Pod Security Policy exists" \
        "$DEPLOY_DIR/security-policies.yaml" \
        "kind: PodSecurityPolicy"

    check_security "Resource quotas defined" \
        "$DEPLOY_DIR/security-policies.yaml" \
        "kind: ResourceQuota"

    check_security "Limit ranges configured" \
        "$DEPLOY_DIR/security-policies.yaml" \
        "kind: LimitRange"

    check_security "Admission webhook configured" \
        "$DEPLOY_DIR/security-policies.yaml" \
        "kind: ValidatingAdmissionWebhook"

    check_security "Gatekeeper constraint template" \
        "$DEPLOY_DIR/security-policies.yaml" \
        "kind: ConstraintTemplate" false
else
    echo -e "${YELLOW}⚠${NC} Security policies file not found (optional)"
    ((WARNINGS++))
fi

echo
echo "Security Validation Summary"
echo "==========================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"
echo

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All critical security checks passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED critical security checks failed!${NC}"
    echo "Please review and fix the failed security configurations."
    exit 1
fi