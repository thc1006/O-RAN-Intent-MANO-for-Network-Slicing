#!/bin/bash
# Production build script for O-RAN Intent-MANO Orchestrator
# 2025 Docker Best Practices - Build Script

set -euo pipefail

# Configuration
IMAGE_NAME="${IMAGE_NAME:-o-ran-orchestrator}"
TAG="${TAG:-latest}"
BUILD_CONTEXT="${BUILD_CONTEXT:-../../../.}"
DOCKERFILE="${DOCKERFILE:-./Dockerfile}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building O-RAN Intent-MANO Orchestrator Docker Image${NC}"
echo "Image: ${IMAGE_NAME}:${TAG}"
echo "Context: ${BUILD_CONTEXT}"
echo "Dockerfile: ${DOCKERFILE}"
echo

# Pre-build checks
echo -e "${YELLOW}Performing pre-build checks...${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed or not in PATH${NC}"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}Docker daemon is not running${NC}"
    exit 1
fi

if [ ! -f "${DOCKERFILE}" ]; then
    echo -e "${RED}Dockerfile not found at ${DOCKERFILE}${NC}"
    exit 1
fi

if [ ! -f "${BUILD_CONTEXT}/orchestrator/go.mod" ]; then
    echo -e "${RED}orchestrator/go.mod not found in build context${NC}"
    exit 1
fi

# Build with enhanced security scanning and optimization
echo -e "${YELLOW}Building Docker image...${NC}"

# Build arguments for reproducible builds
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

docker build \
    --file "${DOCKERFILE}" \
    --tag "${IMAGE_NAME}:${TAG}" \
    --tag "${IMAGE_NAME}:latest" \
    --label "org.opencontainers.image.created=${BUILD_DATE}" \
    --label "org.opencontainers.image.revision=${VCS_REF}" \
    --label "org.opencontainers.image.source=https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing" \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    --progress=plain \
    "${BUILD_CONTEXT}"

echo -e "${GREEN}Build completed successfully!${NC}"

# Security scanning (optional, if tools are available)
if command -v trivy &> /dev/null; then
    echo -e "${YELLOW}Running security scan with Trivy...${NC}"
    trivy image --severity HIGH,CRITICAL "${IMAGE_NAME}:${TAG}" || true
fi

# Image inspection
echo -e "${YELLOW}Image details:${NC}"
docker images "${IMAGE_NAME}:${TAG}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

# Layer analysis
echo -e "${YELLOW}Layer analysis:${NC}"
docker history "${IMAGE_NAME}:${TAG}" --human --no-trunc

echo -e "${GREEN}Docker image ${IMAGE_NAME}:${TAG} is ready for deployment!${NC}"

# Optional: Push to registry
if [ "${PUSH_TO_REGISTRY:-false}" == "true" ] && [ -n "${REGISTRY_URL:-}" ]; then
    echo -e "${YELLOW}Pushing to registry ${REGISTRY_URL}...${NC}"

    FULL_IMAGE_NAME="${REGISTRY_URL}/${IMAGE_NAME}:${TAG}"
    docker tag "${IMAGE_NAME}:${TAG}" "${FULL_IMAGE_NAME}"
    docker push "${FULL_IMAGE_NAME}"

    echo -e "${GREEN}Image pushed to registry: ${FULL_IMAGE_NAME}${NC}"
fi