#!/bin/bash

# Build all Go components locally
# Based on .github/workflows/build.yml

set -e

echo "=== Building all Go components ==="

# Create bin directory
mkdir -p bin

# Build orchestrator
if [ -f "orchestrator/go.mod" ]; then
  echo "Building orchestrator..."
  cd orchestrator
  go build -v -ldflags="-s -w" -o ../bin/orchestrator ./cmd/orchestrator
  cd ..
fi

# Build VNF operator
if [ -f "adapters/vnf-operator/go.mod" ]; then
  echo "Building VNF operator..."
  cd adapters/vnf-operator
  go build -v -ldflags="-s -w" -o ../../bin/vnf-operator ./cmd/operator
  cd ../..
fi

# Build O2 client
if [ -f "o2-client/go.mod" ]; then
  echo "Building O2 client..."
  cd o2-client
  go build -v -ldflags="-s -w" -o ../bin/o2-client ./cmd/client
  cd ..
fi

# Build TN components
if [ -f "tn/go.mod" ]; then
  echo "Building TN components..."
  cd tn
  go build -v -ldflags="-s -w" -o ../bin/tn-manager ./cmd/manager
  go build -v -ldflags="-s -w" -o ../bin/tn-agent ./cmd/agent
  cd ..
fi

# Build CN-DMS
if [ -f "cn-dms/go.mod" ]; then
  echo "Building CN-DMS..."
  cd cn-dms
  go build -v -ldflags="-s -w" -o ../bin/cn-dms ./cmd/dms
  cd ..
fi

# Build RAN-DMS
if [ -f "ran-dms/go.mod" ]; then
  echo "Building RAN-DMS..."
  cd ran-dms
  go build -v -ldflags="-s -w" -o ../bin/ran-dms ./cmd/dms
  cd ..
fi

echo "=== Build complete! ==="
echo "Binaries created in ./bin/"
ls -la bin/