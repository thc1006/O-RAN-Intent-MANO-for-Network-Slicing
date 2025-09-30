#!/bin/bash
set -e

echo "🔍 Validating O-RAN Intent MANO Fixes"
echo "======================================="
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

check() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        return 0
    else
        echo -e "${RED}✗${NC} $1"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

echo "1. Checking Nephio Renderer Implementation..."
echo "---------------------------------------------"

# Check if validatePackageStructure is implemented
if grep -q "func (r .PackageRenderer) validatePackageStructure" nephio-generator/pkg/renderer/package_renderer.go; then
    check "validatePackageStructure() implemented"
else
    echo -e "${RED}✗${NC} validatePackageStructure() not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check if readKptfile is implemented
if grep -q "func (r .PackageRenderer) readKptfile" nephio-generator/pkg/renderer/package_renderer.go; then
    check "readKptfile() implemented"
else
    echo -e "${RED}✗${NC} readKptfile() not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check if executeFunctionPipeline is implemented
if grep -q "func (r .PackageRenderer) executeFunctionPipeline" nephio-generator/pkg/renderer/package_renderer.go; then
    check "executeFunctionPipeline() implemented"
else
    echo -e "${RED}✗${NC} executeFunctionPipeline() not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check if readRenderedResources is implemented
if grep -q "func (r .PackageRenderer) readRenderedResources" nephio-generator/pkg/renderer/package_renderer.go; then
    check "readRenderedResources() implemented"
else
    echo -e "${RED}✗${NC} readRenderedResources() not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check if runKustomizeBuild is implemented
if grep -q "func (r .PackageRenderer) runKustomizeBuild" nephio-generator/pkg/renderer/package_renderer.go; then
    check "runKustomizeBuild() implemented"
else
    echo -e "${RED}✗${NC} runKustomizeBuild() not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

echo ""
echo "2. Checking Function Implementation Details..."
echo "---------------------------------------------"

# Check that validatePackageStructure has actual implementation (not just stub)
if grep -q "packagePath == \"\"" nephio-generator/pkg/renderer/package_renderer.go; then
    check "validatePackageStructure() has implementation"
else
    echo -e "${RED}✗${NC} validatePackageStructure() may be a stub"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check that readKptfile has implementation
if grep -q "os.ReadFile(kptfilePath)" nephio-generator/pkg/renderer/package_renderer.go; then
    check "readKptfile() has implementation"
else
    echo -e "${RED}✗${NC} readKptfile() may be a stub"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check that executeFunctionPipeline has implementation
if grep -q "functions = append(functions, kptfile.Pipeline.Mutators...)" nephio-generator/pkg/renderer/package_renderer.go; then
    check "executeFunctionPipeline() has implementation"
else
    echo -e "${RED}✗${NC} executeFunctionPipeline() may be a stub"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check that readRenderedResources has implementation
if grep -q "os.ReadDir(resourcesDir)" nephio-generator/pkg/renderer/package_renderer.go; then
    check "readRenderedResources() has implementation"
else
    echo -e "${RED}✗${NC} readRenderedResources() may be a stub"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

# Check that runKustomizeBuild has implementation
if grep -q "RenderPackageWithKustomize" nephio-generator/pkg/renderer/package_renderer.go; then
    check "runKustomizeBuild() has implementation"
else
    echo -e "${RED}✗${NC} runKustomizeBuild() may be a stub"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

echo ""
echo "3. Checking Test Module Configuration..."
echo "---------------------------------------------"

if [ -f "work_dir/tests/go.mod" ]; then
    check "work_dir/tests/go.mod exists"
else
    echo -e "${RED}✗${NC} work_dir/tests/go.mod missing"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

if [ -f "work_dir/tests/run-tests.sh" ]; then
    check "work_dir/tests/run-tests.sh exists"
else
    echo -e "${RED}✗${NC} work_dir/tests/run-tests.sh missing"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
fi

echo ""
echo "4. Checking GitOps Implementation..."
echo "---------------------------------------------"

if [ -f "adapters/vnf-operator/pkg/gitops/client.go" ]; then
    if grep -q "func (c \*PorchClient) CreatePackageRevision" adapters/vnf-operator/pkg/gitops/client.go; then
        if ! grep -q "return nil, nil" adapters/vnf-operator/pkg/gitops/client.go | head -1; then
            check "CreatePackageRevision() implemented"
        else
            echo -e "${YELLOW}⚠${NC} CreatePackageRevision() may still return nil, nil"
        fi
    fi
fi

echo ""
echo "5. Compilation Test..."
echo "---------------------------------------------"

# Try to build the main project
if cd nephio-generator && go build ./... 2>/dev/null; then
    check "Nephio generator compiles"
    cd ..
else
    echo -e "${RED}✗${NC} Nephio generator compilation failed"
    FAILED_CHECKS=$((FAILED_CHECKS + 1))
    cd ..
fi

echo ""
echo "======================================"
echo "📊 Validation Summary"
echo "======================================"
echo "Total Checks: $TOTAL_CHECKS"
echo -e "Passed: ${GREEN}$PASSED_CHECKS${NC}"
echo -e "Failed: ${RED}$FAILED_CHECKS${NC}"
echo ""

if [ $FAILED_CHECKS -eq 0 ]; then
    echo -e "${GREEN}✓ All fixes validated successfully!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some fixes are incomplete${NC}"
    exit 1
fi