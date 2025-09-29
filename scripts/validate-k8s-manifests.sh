#!/bin/bash
# Kubernetes v1.31 Manifest Validation Script

set -euo pipefail

echo "🔍 Kubernetes v1.31 Manifest Validation"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_FILES=0
VALID_FILES=0
ERROR_FILES=0

# Function to validate YAML syntax
validate_yaml_syntax() {
    local file="$1"
    echo -n "  Checking YAML syntax... "

    if python3 -c "import yaml, sys; [yaml.safe_load(doc) for doc in yaml.safe_load_all(open('$file').read())]" 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Function to check for deprecated APIs
check_deprecated_apis() {
    local file="$1"
    echo -n "  Checking for deprecated APIs... "

    if grep -q "apiVersion.*beta\|extensions/v1beta1\|policy/v1beta1" "$file" 2>/dev/null; then
        echo -e "${YELLOW}⚠${NC}"
        echo "    Warning: Found potentially deprecated API versions"
        grep "apiVersion.*beta\|extensions/v1beta1\|policy/v1beta1" "$file" | head -3
        return 1
    else
        echo -e "${GREEN}✓${NC}"
        return 0
    fi
}

# Function to check for deprecated annotations
check_deprecated_annotations() {
    local file="$1"
    echo -n "  Checking for deprecated annotations... "

    if grep -q "seccomp\.security\.alpha\|container\.apparmor\.security\.beta" "$file" 2>/dev/null; then
        if grep -q "# .*seccomp\.security\.alpha\|# .*container\.apparmor\.security\.beta" "$file" 2>/dev/null; then
            echo -e "${GREEN}✓${NC} (commented out)"
        else
            echo -e "${YELLOW}⚠${NC}"
            echo "    Warning: Found active deprecated security annotations"
        fi
        return 0
    else
        echo -e "${GREEN}✓${NC}"
        return 0
    fi
}

# Function to validate Kubernetes resources
validate_k8s_resources() {
    local file="$1"
    echo -n "  Validating Kubernetes resources... "

    if command -v kubectl >/dev/null 2>&1; then
        if kubectl --dry-run=client --validate=true apply -f "$file" >/dev/null 2>&1; then
            echo -e "${GREEN}✓${NC}"
            return 0
        else
            echo -e "${RED}✗${NC}"
            echo "    Error: kubectl validation failed"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠${NC} (kubectl not available)"
        return 0
    fi
}

# Function to validate a single file
validate_file() {
    local file="$1"
    local file_valid=true

    echo "📄 Validating: $file"

    if ! validate_yaml_syntax "$file"; then
        file_valid=false
    fi

    if ! check_deprecated_apis "$file"; then
        # Don't fail validation for this, just warn
        true
    fi

    if ! check_deprecated_annotations "$file"; then
        # Don't fail validation for this, just warn
        true
    fi

    if ! validate_k8s_resources "$file"; then
        file_valid=false
    fi

    if $file_valid; then
        echo -e "  ${GREEN}✅ File validation passed${NC}"
        ((VALID_FILES++))
    else
        echo -e "  ${RED}❌ File validation failed${NC}"
        ((ERROR_FILES++))
    fi

    echo ""
    ((TOTAL_FILES++))
}

# Main validation loop
echo "🔍 Searching for Kubernetes manifest files..."
echo ""

# Find all YAML files in key directories
for dir in "deploy/k8s" "security" "monitoring" "net" "observability"; do
    if [[ -d "$dir" ]]; then
        echo "📁 Checking directory: $dir"
        while IFS= read -r -d '' file; do
            validate_file "$file"
        done < <(find "$dir" -name "*.yaml" -o -name "*.yml" -print0 2>/dev/null)
    fi
done

# Summary
echo "📊 Validation Summary"
echo "===================="
echo "Total files processed: $TOTAL_FILES"
echo -e "Valid files: ${GREEN}$VALID_FILES${NC}"
echo -e "Files with errors: ${RED}$ERROR_FILES${NC}"

if [[ $ERROR_FILES -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 All manifests are valid for Kubernetes v1.31!${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some manifests have validation errors. Please review and fix.${NC}"
    exit 1
fi