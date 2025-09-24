#!/bin/bash
# Test compilation script for vnf-operator on Windows

echo "=== Testing VNF Operator Compilation ==="

# Test building all packages
echo "1. Building all packages..."
if go build ./...; then
    echo "✓ All packages compiled successfully"
else
    echo "✗ Package compilation failed"
    exit 1
fi

# Test building the main binary
echo "2. Building manager binary..."
if go build -o bin/manager.exe cmd/manager/main.go; then
    echo "✓ Manager binary compiled successfully"
else
    echo "✗ Manager binary compilation failed"
    exit 1
fi

# Test compilation of test files (without running them)
echo "3. Compiling test files..."
if go test -c ./controllers -o bin/controllers.test.exe; then
    echo "✓ Controller tests compiled successfully"
else
    echo "✗ Controller tests compilation failed"
    exit 1
fi

if go test -c ./tests/golden -o bin/golden.test.exe; then
    echo "✓ Golden tests compiled successfully"
else
    echo "✗ Golden tests compilation failed"
    exit 1
fi

# Check for any compilation errors in specific packages
echo "4. Checking individual packages..."
packages=("api/v1alpha1" "controllers" "pkg/dms" "pkg/gitops" "pkg/translator")

for pkg in "${packages[@]}"; do
    echo "   Checking $pkg..."
    if go build ./$pkg; then
        echo "   ✓ $pkg compiled"
    else
        echo "   ✗ $pkg failed"
        exit 1
    fi
done

echo ""
echo "=== All Compilation Tests Passed Successfully ==="
echo "The vnf-operator can be compiled without errors on this system."