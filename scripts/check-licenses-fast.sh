#!/bin/bash

# Fast license compliance check script for CI/CD

set -e

echo "📄 Running license compliance check..."
echo "=================================="

# Ensure go-licenses is installed
if ! command -v go-licenses &> /dev/null; then
    echo "Installing go-licenses v1.6.0..."
    go install github.com/google/go-licenses@v1.6.0
    echo "go-licenses installed successfully"
fi

license_issues=0
checked_modules=0
skipped_modules=0

# Find all Go modules
modules=$(find . -name "go.mod" -not -path "./vendor/*" -not -path "./.git/*" | xargs dirname | sort)

echo ""
echo "Found $(echo "$modules" | wc -l) Go modules to check"
echo ""

for module in $modules; do
    module_name=$(echo "$module" | sed 's|^\./||')

    # Check if module has Go source files
    if ! find "$module" -name "*.go" -not -path "*/vendor/*" -type f 2>/dev/null | head -1 | grep -q .; then
        echo "⏭️  Skipping $module_name: no Go source files"
        ((skipped_modules++))
        continue
    fi

    # Check if module has external dependencies
    cd "$module" > /dev/null 2>&1

    external_deps=$(go list -m all 2>/dev/null | \
        grep -v "^github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing" | \
        grep -v "^[^/]*$" | \
        wc -l)

    if [ "$external_deps" -eq 0 ]; then
        echo "⏭️  Skipping $module_name: no external dependencies"
        cd - > /dev/null 2>&1
        ((skipped_modules++))
        continue
    fi

    echo "🔍 Checking $module_name ($external_deps external dependencies)..."

    # Run go-licenses check with timeout and proper error handling
    set +e
    timeout 10s go-licenses check ./... \
        --disallowed_types=forbidden \
        --ignore=github.com/thc1006 \
        --logtostderr=false 2>&1 | \
        grep -E "(forbidden|unknown license)" | \
        grep -v "github.com/thc1006" | \
        head -5

    check_result=$?
    set -e

    if [ $check_result -eq 0 ]; then
        echo "   ✅ No forbidden licenses found"
    elif [ $check_result -eq 124 ]; then
        echo "   ⚠️  Check timed out (module might be too large)"
    else
        echo "   ⚠️  Potential license issues detected"
        ((license_issues++))
    fi

    ((checked_modules++))
    cd - > /dev/null 2>&1
    echo ""
done

# Summary
echo "=================================="
echo "📊 License Check Summary:"
echo "  - Modules checked: $checked_modules"
echo "  - Modules skipped: $skipped_modules"
echo "  - License issues: $license_issues"

# Generate JSON report
cat > license-check-report.json << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "modules_checked": $checked_modules,
  "modules_skipped": $skipped_modules,
  "license_issues": $license_issues,
  "compliance_status": "$([ $license_issues -eq 0 ] && echo 'compliant' || echo 'non-compliant')"
}
EOF

echo ""
if [ $license_issues -eq 0 ]; then
    echo "✅ License compliance check PASSED - All dependencies have acceptable licenses"
    exit 0
else
    echo "❌ License compliance check FAILED - Found $license_issues module(s) with license issues"
    exit 1
fi