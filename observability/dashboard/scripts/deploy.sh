#!/bin/bash

# O-RAN Intent-MANO Dashboard Deployment Script

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="${IMAGE_NAME:-oran-dashboard}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
NAMESPACE="${NAMESPACE:-oran-system}"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if required commands exist
    local required_commands=("docker" "kubectl" "npm")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "$cmd is required but not installed"
            exit 1
        fi
    done

    # Check Node.js version
    local node_version
    node_version=$(node --version | sed 's/v//')
    local required_version="18.0.0"

    if ! npx semver -r ">=$required_version" "$node_version" &> /dev/null; then
        log_error "Node.js version $required_version or higher is required (current: $node_version)"
        exit 1
    fi

    # Check if kubectl is configured
    if ! kubectl cluster-info &> /dev/null; then
        log_error "kubectl is not configured or cluster is not accessible"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

build_application() {
    log_info "Building React application..."

    cd "$PROJECT_DIR"

    # Install dependencies
    log_info "Installing dependencies..."
    npm ci

    # Run type checking
    log_info "Running type checks..."
    npm run type-check

    # Run linting
    log_info "Running linting..."
    npm run lint

    # Run tests
    log_info "Running tests..."
    npm run test -- --run

    # Build application
    log_info "Building application..."
    npm run build

    log_success "Application build completed"
}

build_docker_image() {
    log_info "Building Docker image..."

    cd "$PROJECT_DIR"

    # Build Docker image
    docker build \
        --tag "$IMAGE_NAME:$IMAGE_TAG" \
        --tag "$IMAGE_NAME:latest" \
        --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
        --build-arg VCS_REF="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
        --build-arg VERSION="$IMAGE_TAG" \
        .

    log_success "Docker image built: $IMAGE_NAME:$IMAGE_TAG"
}

create_namespace() {
    log_info "Creating namespace if it doesn't exist..."

    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        kubectl create namespace "$NAMESPACE"
        log_success "Namespace $NAMESPACE created"
    else
        log_info "Namespace $NAMESPACE already exists"
    fi
}

deploy_to_kubernetes() {
    log_info "Deploying to Kubernetes..."

    cd "$PROJECT_DIR"

    # Create namespace
    create_namespace

    # Apply Kubernetes manifests
    log_info "Applying Kubernetes manifests..."
    kubectl apply -f kubernetes/ -n "$NAMESPACE"

    # Wait for deployment to be ready
    log_info "Waiting for deployment to be ready..."
    kubectl rollout status deployment/oran-dashboard -n "$NAMESPACE" --timeout=300s

    # Get service information
    log_info "Getting service information..."
    kubectl get services -n "$NAMESPACE" -l app=oran-dashboard

    log_success "Deployment completed successfully"
}

run_health_check() {
    log_info "Running health checks..."

    # Get pod status
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app=oran-dashboard -o jsonpath='{.items[*].metadata.name}')

    for pod in $pods; do
        log_info "Checking health of pod: $pod"

        # Wait for pod to be ready
        kubectl wait --for=condition=ready pod/"$pod" -n "$NAMESPACE" --timeout=120s

        # Check health endpoint
        if kubectl exec -n "$NAMESPACE" "$pod" -- curl -f http://localhost:3000/health &> /dev/null; then
            log_success "Pod $pod is healthy"
        else
            log_warning "Pod $pod health check failed"
        fi
    done

    log_success "Health checks completed"
}

cleanup() {
    log_info "Cleaning up temporary files..."

    cd "$PROJECT_DIR"

    # Remove build artifacts if in development
    if [[ "$ENVIRONMENT" == "development" ]]; then
        rm -rf dist/
        log_info "Build artifacts cleaned up"
    fi
}

show_access_info() {
    log_info "Deployment Information:"
    echo ""
    echo "Namespace: $NAMESPACE"
    echo "Image: $IMAGE_NAME:$IMAGE_TAG"
    echo ""

    # Get ingress information
    local ingress_host
    if ingress_host=$(kubectl get ingress oran-dashboard-ingress -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].host}' 2>/dev/null); then
        echo "Dashboard URL: https://$ingress_host"
    else
        echo "No ingress configured. Use port-forward to access:"
        echo "kubectl port-forward -n $NAMESPACE service/oran-dashboard-service 3000:80"
        echo "Then access: http://localhost:3000"
    fi

    echo ""
    echo "Useful commands:"
    echo "  View logs: kubectl logs -f deployment/oran-dashboard -n $NAMESPACE"
    echo "  Scale deployment: kubectl scale deployment oran-dashboard --replicas=3 -n $NAMESPACE"
    echo "  Delete deployment: kubectl delete -f kubernetes/ -n $NAMESPACE"
}

# Main execution
main() {
    local action="${1:-deploy}"

    case "$action" in
        "build")
            check_prerequisites
            build_application
            build_docker_image
            ;;
        "deploy")
            check_prerequisites
            build_application
            build_docker_image
            deploy_to_kubernetes
            run_health_check
            show_access_info
            ;;
        "health-check")
            run_health_check
            ;;
        "cleanup")
            cleanup
            ;;
        *)
            echo "Usage: $0 {build|deploy|health-check|cleanup}"
            echo ""
            echo "Commands:"
            echo "  build       - Build application and Docker image"
            echo "  deploy      - Full deployment (build + deploy to Kubernetes)"
            echo "  health-check - Run health checks on deployed pods"
            echo "  cleanup     - Clean up temporary files"
            exit 1
            ;;
    esac
}

# Trap cleanup on exit
trap cleanup EXIT

# Execute main function
main "$@"