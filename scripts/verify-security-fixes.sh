#!/bin/bash
# Verify security fixes for Go dependencies

echo "Security Fix Verification Report"
echo "================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check for vulnerable package versions
check_version() {
    local module_path=$1
    local package=$2
    local min_version=$3
    local module_name=$(basename $(dirname $module_path))

    echo -n "Checking $module_name ($package): "

    if [ -f "$module_path" ]; then
        version=$(grep "$package " "$module_path" | head -1 | sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+).*/\1/')
        if [ ! -z "$version" ]; then
            # Compare versions (simple comparison, works for most cases)
            if [ "$(printf '%s\n' "$min_version" "$version" | sort -V | head -n1)" = "$min_version" ]; then
                echo -e "${GREEN}✓ v$version (secure)${NC}"
                return 0
            else
                echo -e "${RED}✗ v$version (vulnerable - needs >= v$min_version)${NC}"
                return 1
            fi
        else
            echo "Not found in module"
        fi
    else
        echo "Module file not found"
    fi
}

# List of modules to check
modules=(
    "tn/manager/go.mod"
    "tests/go.mod"
    "adapters/vnf-operator/go.mod"
    "clusters/validation-framework/go.mod"
    "tests/framework/dashboard/go.mod"
)

# Track overall status
all_fixed=true

echo "1. Checking golang.org/x/oauth2 (CVE: JWT vulnerability)"
echo "   Required: >= v0.24.0"
for module in "${modules[@]}"; do
    check_version "$module" "golang.org/x/oauth2" "0.24.0" || all_fixed=false
done
echo ""

echo "2. Checking google.golang.org/protobuf (CVE: Infinite loop in Unmarshal)"
echo "   Required: >= v1.33.0"
for module in "${modules[@]}"; do
    check_version "$module" "google.golang.org/protobuf" "1.33.0" || all_fixed=false
done
echo ""

echo "3. Checking golang.org/x/net (Multiple CVEs)"
echo "   Required: >= v0.23.0"
for module in "${modules[@]}"; do
    check_version "$module" "golang.org/x/net" "0.23.0" || all_fixed=false
done
echo ""

echo "4. Code vulnerability checks:"
echo -n "   Slice allocation fix in metrics_aggregator.go: "
if grep -q "const maxLimit = 10000" tests/framework/dashboard/metrics_aggregator.go 2>/dev/null; then
    echo -e "${GREEN}✓ Fixed${NC}"
else
    echo -e "${RED}✗ Not fixed${NC}"
    all_fixed=false
fi
echo ""

# Summary
echo "================================="
if [ "$all_fixed" = true ]; then
    echo -e "${GREEN}✓ All security vulnerabilities have been fixed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some vulnerabilities still need attention${NC}"
    exit 1
fi