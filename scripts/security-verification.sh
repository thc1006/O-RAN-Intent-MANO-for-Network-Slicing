#!/bin/bash

# Security Verification Script for O-RAN Intent MANO
# Verifies all security fixes are properly implemented

set -e

echo "🔒 O-RAN Intent MANO Security Verification"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Function to check and report
check_security_fix() {
    local description="$1"
    local check_command="$2"
    local expected_result="$3"

    echo -n "Checking: $description... "

    if eval "$check_command"; then
        echo -e "${GREEN}✅ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "  Expected: $expected_result"
        ((FAILED++))
    fi
}

echo -e "\n🔍 1. Weak Random Number Generator Fixes"
echo "----------------------------------------"

# Check crypto/rand usage in nephio-generator
check_security_fix \
    "nephio-generator uses crypto/rand" \
    "grep -q 'crypto/rand' nephio-generator/pkg/errors/error_handling.go && ! grep -q 'math/rand' nephio-generator/pkg/errors/error_handling.go" \
    "crypto/rand import present, math/rand not used"

# Check crypto/rand usage in orchestrator
check_security_fix \
    "orchestrator uses crypto/rand" \
    "grep -q 'crypto/rand' orchestrator/pkg/placement/metrics_mock.go && ! grep -q 'math/rand' orchestrator/pkg/placement/metrics_mock.go" \
    "crypto/rand import present, math/rand not used"

echo -e "\n🛡️ 2. HTTP Server Timeout Fixes (Slowloris Protection)"
echo "----------------------------------------------------"

# Check CN DMS timeouts
check_security_fix \
    "CN DMS has ReadHeaderTimeout" \
    "grep -q 'ReadHeaderTimeout.*time.Second' cn-dms/cmd/main.go" \
    "ReadHeaderTimeout configured"

check_security_fix \
    "CN DMS has complete timeout configuration" \
    "grep -A5 -B5 'ReadHeaderTimeout' cn-dms/cmd/main.go | grep -q 'ReadTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' cn-dms/cmd/main.go | grep -q 'WriteTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' cn-dms/cmd/main.go | grep -q 'IdleTimeout.*time.Second'" \
    "All HTTP timeouts configured"

# Check RAN DMS timeouts
check_security_fix \
    "RAN DMS has ReadHeaderTimeout" \
    "grep -q 'ReadHeaderTimeout.*time.Second' ran-dms/cmd/main.go" \
    "ReadHeaderTimeout configured"

check_security_fix \
    "RAN DMS has complete timeout configuration" \
    "grep -A5 -B5 'ReadHeaderTimeout' ran-dms/cmd/main.go | grep -q 'ReadTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' ran-dms/cmd/main.go | grep -q 'WriteTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' ran-dms/cmd/main.go | grep -q 'IdleTimeout.*time.Second'" \
    "All HTTP timeouts configured"

# Check Dashboard timeouts
check_security_fix \
    "Dashboard has ReadHeaderTimeout" \
    "grep -q 'ReadHeaderTimeout.*time.Second' tests/framework/dashboard/dashboard.go" \
    "ReadHeaderTimeout configured"

check_security_fix \
    "Dashboard has complete timeout configuration" \
    "grep -A5 -B5 'ReadHeaderTimeout' tests/framework/dashboard/dashboard.go | grep -q 'ReadTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' tests/framework/dashboard/dashboard.go | grep -q 'WriteTimeout.*time.Second' && grep -A5 -B5 'ReadHeaderTimeout' tests/framework/dashboard/dashboard.go | grep -q 'IdleTimeout.*time.Second'" \
    "All HTTP timeouts configured"

echo -e "\n🚢 3. Kubernetes Security Fixes"
echo "-------------------------------"

# Check orchestrator security
check_security_fix \
    "Orchestrator has seccomp profile" \
    "grep -q 'type: RuntimeDefault' deploy/k8s/base/orchestrator.yaml" \
    "seccomp profile set to RuntimeDefault"

check_security_fix \
    "Orchestrator uses pinned image" \
    "grep -q 'sha256:' deploy/k8s/base/orchestrator.yaml" \
    "Image tag includes SHA256 hash"

check_security_fix \
    "Orchestrator has ImagePullPolicy Always" \
    "grep -q 'imagePullPolicy: Always' deploy/k8s/base/orchestrator.yaml" \
    "ImagePullPolicy set to Always"

check_security_fix \
    "Orchestrator disables service account token mounting" \
    "grep -q 'automountServiceAccountToken: false' deploy/k8s/base/orchestrator.yaml" \
    "Service account token mounting disabled"

# Check VNF operator security
check_security_fix \
    "VNF Operator has seccomp profile" \
    "grep -q 'type: RuntimeDefault' deploy/k8s/base/vnf-operator.yaml" \
    "seccomp profile set to RuntimeDefault"

check_security_fix \
    "VNF Operator uses pinned image" \
    "grep -q 'sha256:' deploy/k8s/base/vnf-operator.yaml" \
    "Image tag includes SHA256 hash"

check_security_fix \
    "VNF Operator has ImagePullPolicy Always" \
    "grep -q 'imagePullPolicy: Always' deploy/k8s/base/vnf-operator.yaml" \
    "ImagePullPolicy set to Always"

check_security_fix \
    "VNF Operator disables service account token mounting" \
    "grep -q 'automountServiceAccountToken: false' deploy/k8s/base/vnf-operator.yaml" \
    "Service account token mounting disabled"

echo -e "\n📊 Security Verification Summary"
echo "================================"
echo -e "Total Checks: $((PASSED + FAILED))"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All security fixes verified successfully!${NC}"
    echo -e "${GREEN}The O-RAN Intent MANO codebase meets security requirements.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some security checks failed. Please review and fix the issues above.${NC}"
    exit 1
fi