#!/bin/bash
# VNF Operator Docker Build Script
# Usage: ./build.sh [tag]

set -e

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Default tag
TAG="${1:-vnf-operator:latest}"

echo "Building VNF Operator Docker image..."
echo "Project root: ${PROJECT_ROOT}"
echo "Tag: ${TAG}"
echo

# Build from project root with proper context
cd "${PROJECT_ROOT}"

docker build \
    -f deploy/docker/vnf-operator/Dockerfile \
    -t "${TAG}" \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    .

echo
echo "✅ Build completed successfully!"
echo "Image: ${TAG}"
echo
echo "To run the image:"
echo "docker run --rm -p 8080:8080 -p 8081:8081 ${TAG}"
echo
echo "To push to registry:"
echo "docker tag ${TAG} your-registry/${TAG}"
echo "docker push your-registry/${TAG}"