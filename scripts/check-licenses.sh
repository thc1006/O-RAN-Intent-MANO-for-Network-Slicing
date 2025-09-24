#!/bin/bash

# License compliance check script for O-RAN Intent MANO project

echo "📄 Running license compliance check..."

license_issues=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Install go-licenses if not available
if ! command -v go-licenses &> /dev/null; then
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@v1.6.0
fi

# Check Go module licenses
echo "  Checking Go module licenses..."

# Find all go.mod files (excluding vendor)
for module_path in $(find "$PROJECT_ROOT" -name "go.mod" -not -path "*/vendor/*" | xargs dirname); do
    module_name=$(basename "$module_path")
    echo "  Checking licenses in $module_name..."

    cd "$module_path"

    # Skip if no Go source files exist (empty modules)
    if ! find . -name "*.go" -not -path "./vendor/*" 2>/dev/null | grep -q .; then
        echo "    Skipping $module_name: no Go source files found"
        continue
    fi

    # Check if module has external dependencies (not just internal packages)
    external_deps=$(go list -m all 2>/dev/null | grep -v "^github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing" | grep -v "^[^/]*$" | wc -l)

    if [ "$external_deps" -eq 0 ]; then
        echo "    Skipping $module_name: no external dependencies found"
        continue
    fi

    echo "    Found $external_deps external dependencies to check"

    # Run go-licenses check only on external dependencies
    # We'll use go list to get external packages and check them
    go list -json ./... 2>/dev/null | \
        jq -r '.Imports[]' 2>/dev/null | \
        grep -v "^github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing" | \
        grep -v "^C$" | \
        sort -u | while read -r pkg; do

        # Skip standard library packages
        if ! echo "$pkg" | grep -q "/"; then
            continue
        fi

        # Check the package license
        go-licenses check "$pkg" --disallowed_types=forbidden,unknown 2>/dev/null || {
            echo "    ⚠️  License issue found in dependency: $pkg"
            ((license_issues++))
        }
    done
done

cd "$PROJECT_ROOT"

# Generate license report
cat > license-check-report.json << EOF
{
  "analysis_type": "license-check",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "license_issues": $license_issues,
  "compliance_status": "$([ $license_issues -eq 0 ] && echo 'compliant' || echo 'non-compliant')"
}
EOF

echo ""
if [ $license_issues -eq 0 ]; then
    echo "✅ License compliance check completed successfully - No issues found"
else
    echo "⚠️  License compliance check completed - Found $license_issues issue(s)"
    exit 1
fi

exit 0