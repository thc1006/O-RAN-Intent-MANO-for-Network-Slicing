#!/bin/bash
# Test runner script to fix CI issues

set -e

echo "Running VNF Operator Tests..."

# Set environment variables
export KUBEBUILDER_ASSETS="${KUBEBUILDER_ASSETS:-$(setup-envtest use 1.25 --bin-dir /tmp -p path)}"
export CGO_ENABLED=1
export GO111MODULE=on

# Function to run tests
run_tests() {
    echo "Testing package: $1"
    go test -v -race -coverprofile=/tmp/coverage.out -covermode=atomic "$1" || {
        echo "Tests failed for $1"
        return 1
    }
}

# Build the operator first to catch compilation errors
echo "Building VNF Operator..."
go build -o /tmp/vnf-operator ./cmd/manager || {
    echo "Build failed"
    exit 1
}

# Run controller tests
echo "Running controller tests..."
run_tests "./controllers"

# Run other package tests
echo "Running API tests..."
run_tests "./api/..."

# Run integration tests if available
if [ -d "./tests/integration" ]; then
    echo "Running integration tests..."
    run_tests "./tests/integration"
fi

echo "All tests passed successfully!"