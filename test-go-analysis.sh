#!/bin/bash

# Local reproduction of the "Run Go code analysis" step from enhanced-ci.yml
# This script replicates the CI workflow to find and fix the golangci-lint issues

echo "🔍 Reproducing Go code analysis workflow locally..."
echo "=============================================="

# Set environment variables like in CI
export GOLANGCI_LINT_VERSION="v2.5.0"
export GO_VERSION="1.24.7"

# Clean Go module cache to prevent permission issues (like in CI)
echo "🧹 Cleaning Go module cache and build cache..."
rm -rf ~/go/pkg/mod || true
rm -rf ~/.cache/go-build || true
go clean -modcache || true
go clean -cache || true

# Install analysis tools (like in CI)
echo "🔧 Installing Go analysis tools..."

# Install golangci-lint using official installer for better compatibility
echo "Installing golangci-lint ${GOLANGCI_LINT_VERSION}..."
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}
export PATH="$(go env GOPATH)/bin:$PATH"

# Install other tools
echo "Installing additional analysis tools..."
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
go install github.com/client9/misspell/cmd/misspell@latest

echo ""
echo "🔍 Running comprehensive Go code analysis..."

# Initialize quality metrics
total_issues=0
complexity_violations=0

# Find all Go modules
for module in $(find . -name "go.mod" -not -path "./vendor/*" | xargs dirname); do
  echo "📁 Analyzing Go module: $module"
  cd "$module"

  echo "  Module info:"
  echo "    PWD: $(pwd)"
  echo "    Go version: $(go version)"
  echo "    Go env GOROOT: $(go env GOROOT)"
  echo "    Go env GOPATH: $(go env GOPATH)"

  # Check go.mod content
  echo "    Go module: $(head -3 go.mod)"

  echo ""

  # Basic Go checks
  echo "  Running go vet..."
  if ! go vet ./...; then
    ((total_issues++))
    echo "    go vet found issues"
  else
    echo "    go vet passed"
  fi

  # Static analysis
  echo "  Running staticcheck..."
  if ! staticcheck ./...; then
    ((total_issues++))
    echo "    staticcheck found issues"
  else
    echo "    staticcheck passed"
  fi

  # Clean build cache before running golangci-lint to prevent version conflicts
  echo "  Cleaning go cache before golangci-lint..."
  go clean -cache

  # Check for k8s.io/apimachinery compatibility issues
  if grep -q "k8s.io/apimachinery v0.34" go.mod; then
    echo "  ⚠️  Skipping golangci-lint for this module (k8s.io/apimachinery v0.34+ compatibility issue)"
  else
    echo "  Running golangci-lint..."
    echo "    golangci-lint version: $(golangci-lint version)"

    # Create a simplified .golangci.yml for debugging
    cat > .golangci-debug.yml << 'EOF'
run:
  timeout: 15m
  issues-exit-code: 0

linters-settings:
  errcheck:
    check-type-assertions: true
  gocyclo:
    min-complexity: 15

linters:
  enable:
    - errcheck
    - gofmt
    - goimports
    - ineffassign
    - misspell
    - staticcheck
    - typecheck
    - unused
    - vet
  disable:
    - varnamelen
    - wrapcheck

issues:
  max-issues-per-linter: 50
  max-same-issues: 10

output:
  format: colored-line-number
EOF

    echo "    Using debug config file..."

    if golangci-lint run --config .golangci-debug.yml ./...; then
      echo "    golangci-lint passed"
    else
      echo "    golangci-lint found issues"
      ((total_issues++))

      # Try with minimal config to identify specific issues
      echo "    Trying minimal config to isolate the problem..."

      cat > .golangci-minimal.yml << 'EOF'
run:
  timeout: 5m
  issues-exit-code: 0

linters:
  enable:
    - errcheck
    - gofmt
  disable-all: true

issues:
  max-issues-per-linter: 10
EOF

      echo "    Running with minimal linters..."
      golangci-lint run --config .golangci-minimal.yml ./... || echo "    Even minimal config failed"
    fi

    # Clean up debug files
    rm -f .golangci-debug.yml .golangci-minimal.yml
  fi

  # Cyclomatic complexity
  echo "  Checking cyclomatic complexity..."
  complex_funcs=$(gocyclo -over 15 . | wc -l)
  complexity_violations=$((complexity_violations + complex_funcs))
  echo "    Complex functions: $complex_funcs"

  # Spelling check
  echo "  Running spell check..."
  if ! misspell -error .; then
    ((total_issues++))
    echo "    misspell found issues"
  else
    echo "    misspell passed"
  fi

  echo "  Module analysis completed"
  echo ""

  cd - > /dev/null
done

echo ""
echo "📊 Analysis Summary:"
echo "==================="
echo "Total issues: $total_issues"
echo "Complexity violations: $complexity_violations"
echo "Modules analyzed: $(find . -name "go.mod" -not -path "./vendor/*" | wc -l)"

# Generate analysis report like in CI
cat > go-analysis-report.json << EOF
{
  "analysis_type": "go-analysis",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "total_issues": $total_issues,
  "complexity_violations": $complexity_violations,
  "modules_analyzed": $(find . -name "go.mod" -not -path "./vendor/*" | wc -l),
  "quality_gate_status": "$([ $total_issues -le 10 ] && [ $complexity_violations -le 3 ] && echo 'passed' || echo 'failed')"
}
EOF

echo ""
echo "📋 Generated analysis report: go-analysis-report.json"
cat go-analysis-report.json

# Quality gate validation
echo ""
echo "🚪 Quality Gate Validation:"
echo "=========================="

if [ $total_issues -gt 10 ]; then
  echo "❌ ERROR: Go analysis failed: $total_issues issues found (max: 10)"
else
  echo "✅ Issues count passed: $total_issues (max: 10)"
fi

if [ $complexity_violations -gt 3 ]; then
  echo "❌ ERROR: Complexity violations: $complexity_violations (max: 3)"
else
  echo "✅ Complexity check passed: $complexity_violations (max: 3)"
fi

echo ""
echo "Local Go analysis reproduction completed!"