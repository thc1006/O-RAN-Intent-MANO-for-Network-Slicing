#!/bin/bash
set -e

cd "$(dirname "$0")"

# Initialize module if needed
if [ ! -f "go.sum" ]; then
    echo "Initializing Go module..."
    go mod tidy
fi

# Run tests
echo "Running tests..."
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
if [ -f "coverage.out" ]; then
    echo ""
    echo "Coverage Summary:"
    go tool cover -func=coverage.out | tail -5

    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo "HTML coverage report: coverage.html"
fi

echo ""
echo "All tests completed!"