#!/bin/bash
# Docker Build Validation Script for O-RAN Intent-MANO
# Validates all Dockerfiles build successfully with correct Go 1.24.7 version

set -euo pipefail

# Script configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly DOCKER_DIR="${PROJECT_ROOT}/deploy/docker"
readonly LOG_FILE="${PROJECT_ROOT}/docker-build-validation.log"
readonly GO_VERSION="1.24.7"
readonly EXPECTED_GOTOOLCHAIN="go1.24.7"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_FILE}"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "${LOG_FILE}"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" | tee -a "${LOG_FILE}"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" | tee -a "${LOG_FILE}"
}

# Initialize log file
echo "Docker Build Validation Log - $(date)" > "${LOG_FILE}"
echo "=========================================" >> "${LOG_FILE}"

# Services to build and validate
declare -a SERVICES=(
    "orchestrator"
    "vnf-operator"
    "o2-client"
    "tn-manager"
    "tn-agent"
    "cn-dms"
    "ran-dms"
    "test-framework"
)

# Function to validate Dockerfile content
validate_dockerfile_content() {
    local service="$1"
    local dockerfile="${DOCKER_DIR}/${service}/Dockerfile"

    log "Validating Dockerfile content for ${service}..."

    if [[ ! -f "${dockerfile}" ]]; then
        error "Dockerfile not found: ${dockerfile}"
        return 1
    fi

    # Check for correct Go version in FROM statement
    if ! grep -q "FROM golang:${GO_VERSION}-alpine" "${dockerfile}"; then
        error "${service}: Dockerfile doesn't use golang:${GO_VERSION}-alpine base image"
        return 1
    fi

    # Check for correct GOTOOLCHAIN
    if ! grep -q "ENV GOTOOLCHAIN=${EXPECTED_GOTOOLCHAIN}" "${dockerfile}"; then
        error "${service}: Dockerfile doesn't set GOTOOLCHAIN=${EXPECTED_GOTOOLCHAIN}"
        return 1
    fi

    # Check for security best practices
    if ! grep -q "USER.*[0-9]" "${dockerfile}"; then
        warning "${service}: Dockerfile may not set non-root user properly"
    fi

    success "${service}: Dockerfile content validation passed"
    return 0
}

# Function to build Docker image
build_docker_image() {
    local service="$1"
    local dockerfile="${DOCKER_DIR}/${service}/Dockerfile"
    local image_name="oran-${service}:${GO_VERSION}"

    log "Building Docker image for ${service}..."

    cd "${PROJECT_ROOT}"

    # Build with explicit Go version tag
    if docker build \
        --no-cache \
        --tag "${image_name}" \
        --file "${dockerfile}" \
        --build-arg GOTOOLCHAIN="${EXPECTED_GOTOOLCHAIN}" \
        --label "build.timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --label "build.go.version=${GO_VERSION}" \
        --label "build.service=${service}" \
        . > "/tmp/docker-build-${service}.log" 2>&1; then
        success "${service}: Docker build completed successfully"

        # Verify the built image
        if docker inspect "${image_name}" > /dev/null 2>&1; then
            success "${service}: Docker image ${image_name} created successfully"

            # Get image size
            local image_size
            image_size=$(docker images "${image_name}" --format "table {{.Size}}" | tail -n1)
            log "${service}: Image size: ${image_size}"

            return 0
        else
            error "${service}: Docker image inspection failed"
            return 1
        fi
    else
        error "${service}: Docker build failed. Check /tmp/docker-build-${service}.log"
        cat "/tmp/docker-build-${service}.log" | tail -20 >> "${LOG_FILE}"
        return 1
    fi
}

# Function to test Docker image
test_docker_image() {
    local service="$1"
    local image_name="oran-${service}:${GO_VERSION}"

    log "Testing Docker image for ${service}..."

    # Test that image starts without immediate crash
    local container_id
    if container_id=$(docker run -d "${image_name}" sleep 5 2>/dev/null); then
        sleep 2

        # Check if container is running
        if docker ps -q --filter id="${container_id}" | grep -q "${container_id}"; then
            success "${service}: Docker container starts successfully"
            docker stop "${container_id}" > /dev/null 2>&1 || true
            docker rm "${container_id}" > /dev/null 2>&1 || true
            return 0
        else
            error "${service}: Docker container failed to start properly"
            docker logs "${container_id}" 2>&1 | tail -10 >> "${LOG_FILE}"
            docker rm "${container_id}" > /dev/null 2>&1 || true
            return 1
        fi
    else
        error "${service}: Failed to start Docker container"
        return 1
    fi
}

# Function to clean up old images
cleanup_old_images() {
    log "Cleaning up old Docker images..."

    # Remove old images with different tags
    docker images --format "table {{.Repository}}" | grep "^oran-" | while read -r repo; do
        if [[ -n "${repo}" ]]; then
            # Keep only the latest Go version, remove others
            docker images "${repo}" --format "table {{.Tag}}" | grep -v "^TAG$" | grep -v "${GO_VERSION}" | while read -r tag; do
                if [[ -n "${tag}" && "${tag}" != "latest" ]]; then
                    warning "Removing old image: ${repo}:${tag}"
                    docker rmi "${repo}:${tag}" 2>/dev/null || true
                fi
            done
        fi
    done

    # Remove dangling images
    docker image prune -f > /dev/null 2>&1 || true

    success "Docker cleanup completed"
}

# Main validation function
main() {
    local exit_code=0
    local failed_services=()

    log "Starting Docker build validation for O-RAN Intent-MANO"
    log "Expected Go version: ${GO_VERSION}"
    log "Expected GOTOOLCHAIN: ${EXPECTED_GOTOOLCHAIN}"

    # Clean up old images first
    cleanup_old_images

    # Validate each service
    for service in "${SERVICES[@]}"; do
        log "Processing service: ${service}"

        # Step 1: Validate Dockerfile content
        if ! validate_dockerfile_content "${service}"; then
            failed_services+=("${service}")
            exit_code=1
            continue
        fi

        # Step 2: Build Docker image
        if ! build_docker_image "${service}"; then
            failed_services+=("${service}")
            exit_code=1
            continue
        fi

        # Step 3: Test Docker image
        if ! test_docker_image "${service}"; then
            failed_services+=("${service}")
            exit_code=1
            continue
        fi

        success "${service}: All validation steps passed"
        echo "---"
    done

    # Final report
    echo ""
    log "========================================="
    log "Docker Build Validation Summary"
    log "========================================="

    if [[ ${#failed_services[@]} -eq 0 ]]; then
        success "All ${#SERVICES[@]} services built and validated successfully!"
        log "All Docker images are ready for deployment with Go ${GO_VERSION}"
    else
        error "Failed services: ${failed_services[*]}"
        error "${#failed_services[@]} out of ${#SERVICES[@]} services failed validation"
    fi

    # Show disk usage
    log "Docker disk usage:"
    docker system df | tee -a "${LOG_FILE}"

    log "Validation completed. Full log available at: ${LOG_FILE}"
    exit ${exit_code}
}

# Trap to ensure cleanup on exit
trap cleanup_old_images EXIT

# Run main function
main "$@"