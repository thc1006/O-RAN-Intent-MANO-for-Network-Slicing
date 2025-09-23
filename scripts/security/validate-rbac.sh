#!/bin/bash
# Pre-commit hook for RBAC validation
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Validating RBAC policies...${NC}"

EXIT_CODE=0

# Find all RBAC files
RBAC_FILES=$(find . -name "*rbac*.yaml" -o -name "*rbac*.yml" -not -path "./vendor/*" -not -path "./.git/*")

if [ -z "$RBAC_FILES" ]; then
    echo -e "${YELLOW}No RBAC files found to validate.${NC}"
    exit 0
fi

for rbac_file in $RBAC_FILES; do
    echo -e "${GREEN}Validating: $rbac_file${NC}"

    ISSUES_FOUND=false

    # Check 1: Ensure proper RBAC kinds
    if ! grep -qE "kind.*(Role|ClusterRole|RoleBinding|ClusterRoleBinding|ServiceAccount)" "$rbac_file"; then
        echo -e "${RED}❌ File $rbac_file does not contain valid RBAC kinds${NC}"
        ISSUES_FOUND=true
    fi

    # Check 2: Ensure proper apiVersion
    if ! grep -q "apiVersion.*rbac.authorization.k8s.io/v1" "$rbac_file" && ! grep -q "apiVersion.*v1" "$rbac_file"; then
        echo -e "${RED}❌ Incorrect or missing apiVersion in $rbac_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 3: Check for overly permissive permissions
    if grep -q "resources:.*\\[\"\\*\"\\]" "$rbac_file" || grep -q "verbs:.*\\[\"\\*\"\\]" "$rbac_file"; then
        echo -e "${RED}❌ Overly permissive RBAC rules found in $rbac_file (wildcard permissions)${NC}"
        ISSUES_FOUND=true
    fi

    # Check 4: Ensure specific resource permissions
    if grep -q "kind.*Role" "$rbac_file" && ! grep -q "rules:" "$rbac_file"; then
        echo -e "${RED}❌ Role without rules found in $rbac_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 5: Check for dangerous verbs
    if grep -qE "verbs:.*\"(create|delete|update)\".*\"secrets\"" "$rbac_file"; then
        echo -e "${YELLOW}⚠️  Potentially dangerous secret permissions in $rbac_file${NC}"
    fi

    if grep -qE "verbs:.*\"(escalate|bind)\"" "$rbac_file"; then
        echo -e "${YELLOW}⚠️  Privilege escalation verbs found in $rbac_file - ensure this is intentional${NC}"
    fi

    # Check 6: Ensure proper binding subjects
    if grep -q "kind.*Binding" "$rbac_file" && ! grep -q "subjects:" "$rbac_file"; then
        echo -e "${RED}❌ Binding without subjects found in $rbac_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 7: Validate YAML syntax
    if ! python3 -c "import yaml; yaml.safe_load(open('$rbac_file'))" 2>/dev/null; then
        echo -e "${RED}❌ Invalid YAML syntax in $rbac_file${NC}"
        ISSUES_FOUND=true
    fi

    # Check 8: Ensure least privilege principle
    if grep -q "apiGroups:.*\\[\"\\*\"\\]" "$rbac_file"; then
        echo -e "${YELLOW}⚠️  Wildcard API group permissions in $rbac_file - consider being more specific${NC}"
    fi

    # Check 9: Check for default service account usage
    if grep -q "name:.*default" "$rbac_file" && grep -q "kind.*ServiceAccount" "$rbac_file"; then
        echo -e "${YELLOW}⚠️  Using default service account in $rbac_file - consider using dedicated service accounts${NC}"
    fi

    if [ "$ISSUES_FOUND" = true ]; then
        EXIT_CODE=1
    else
        echo -e "${GREEN}✅ $rbac_file passed validation${NC}"
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All RBAC policies are valid!${NC}"
else
    echo -e "${RED}Some RBAC policies have issues. Please fix them before committing.${NC}"
fi

exit $EXIT_CODE