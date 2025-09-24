#!/bin/bash

# Simple license compliance check script that mirrors CI behavior

echo "📄 Running license compliance check..."

license_issues=0

# Install go-licenses if not available
if ! command -v go-licenses &> /dev/null; then
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@v1.6.0
fi

# Check Go module licenses
echo "  Checking Go module licenses..."

for module in $(find . -name "go.mod" -not -path "./vendor/*" | xargs dirname); do
    echo "  Checking licenses in $module..."
    cd "$module"

    # Skip if no Go source files exist (empty modules)
    if ! find . -name "*.go" -not -path "./vendor/*" | grep -q .; then
        echo "    Skipping $module: no Go source files found"
        cd - > /dev/null
        continue
    fi

    # Ensure go.mod has dependencies before running go-licenses
    if ! go list -m all | grep -v -E "^(github\.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing|[^/]+)$" | grep -q .; then
        echo "    Skipping $module: no external dependencies found"
        cd - > /dev/null
        continue
    fi

    # Use go-licenses to check dependencies with updated flags
    # We ignore errors from our own packages and only check external dependencies
    go-licenses check ./... --disallowed_types=forbidden,unknown --ignore=github.com/thc1006 2>&1 | \
        grep -v "Failed to find license for github.com/thc1006" | \
        grep -v "does not have module info" | \
        grep -v "contains non-Go code" | \
        grep -v "some errors occurred when loading" | \
        grep -E "(forbidden|unknown|UNKNOWN)" && ((license_issues++)) || true

    cd - > /dev/null
done

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
echo "License compliance check completed"

if [ $license_issues -gt 0 ]; then
    echo "⚠️  Found $license_issues license issue(s)"
    exit 1
else
    echo "✅ No license issues found"
fi

exit 0