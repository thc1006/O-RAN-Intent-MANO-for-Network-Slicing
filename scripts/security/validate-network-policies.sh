#!/bin/bash
# Pre-commit hook for Network Policy validation
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Validating Network Policies...${NC}"

EXIT_CODE=0

# Find all network policy files
NETPOL_FILES=$(find . -name "*network*policy*.yaml" -o -name "*network*policy*.yml" -not -path "./vendor/*" -not -path "./.git/*")

if [ -z "$NETPOL_FILES" ]; then
    echo -e "${YELLOW}No Network Policy files found to validate.${NC}"
    exit 0
fi

for netpol_file in $NETPOL_FILES; do
    echo -e "${GREEN}Validating: $netpol_file${NC}"

    ISSUES_FOUND=false

    # Check 1: Ensure proper kind
    if ! grep -q "kind.*NetworkPolicy" "$netpol_file"; then
        echo -e "${RED}❌ File $netpol_file does not contain NetworkPolicy kind${NC}"
        ISSUES_FOUND=true
    fi

    # Check 2: Ensure apiVersion is correct
    if ! grep -q "apiVersion.*networking.k8s.io/v1" "$netpol_file"; then
        echo -e "${RED}❌ Incorrect or missing apiVersion in $netpol_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 3: Ensure metadata.name exists
    if ! grep -q "name:" "$netpol_file"; then
        echo -e "${RED}❌ Missing metadata.name in $netpol_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 4: Ensure podSelector exists
    if ! grep -q "podSelector:" "$netpol_file"; then
        echo -e "${RED}❌ Missing spec.podSelector in $netpol_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 5: Ensure proper ingress/egress rules
    if ! grep -q -E "(ingress:|egress:)" "$netpol_file"; then
        echo -e "${YELLOW}⚠️  No ingress or egress rules found in $netpol_file${NC}"
    fi

    # Check 6: Validate no overly permissive rules
    if grep -q "podSelector: {}" "$netpol_file" && grep -q -E "(ingress:\s*-\s*{}|egress:\s*-\s*{})" "$netpol_file"; then
        echo -e "${RED}❌ Overly permissive network policy in $netpol_file (empty podSelector with empty rules)${NC}"
        ISSUES_FOUND=true
    fi

    # Check 7: Ensure proper YAML syntax
    if ! python3 -c "import yaml; yaml.safe_load(open('$netpol_file'))" 2>/dev/null; then
        echo -e "${RED}❌ Invalid YAML syntax in $netpol_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 8: Ensure namespace is specified for multi-tenant environments
    if ! grep -q "namespace:" "$netpol_file"; then
        echo -e "${YELLOW}⚠️  Consider specifying namespace in $netpol_file for better security isolation${NC}"
    fi

    if [ "$ISSUES_FOUND" = true ]; then
        EXIT_CODE=1
    else
        echo -e "${GREEN}✅ $netpol_file passed validation${NC}"
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All Network Policies are valid!${NC}"
else
    echo -e "${RED}Some Network Policies have issues. Please fix them before committing.${NC}"
fi

exit $EXIT_CODE