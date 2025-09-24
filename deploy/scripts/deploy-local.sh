#!/bin/bash
# O-RAN Intent-MANO Local Deployment Script
# Comprehensive deployment with validation and health checks

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DOCKER_COMPOSE_DIR="${PROJECT_ROOT}/deploy/docker"
COMPOSE_FILES=(
    "${DOCKER_COMPOSE_DIR}/docker-compose.local.yml"
    "${DOCKER_COMPOSE_DIR}/docker-compose.test.yml"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validation functions
check_dependencies() {
    log_info "Checking dependencies..."

    local deps=("docker" "docker-compose" "curl" "wget" "jq")
    local missing=()

    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        log_info "Please install: sudo apt-get update && sudo apt-get install -y ${missing[*]}"
        exit 1
    fi

    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    log_success "All dependencies are available"
}

create_directories() {
    log_info "Creating required directories..."

    local dirs=(
        "${DOCKER_COMPOSE_DIR}/data/orchestrator"
        "${DOCKER_COMPOSE_DIR}/data/vnf-operator"
        "${DOCKER_COMPOSE_DIR}/data/o2-client"
        "${DOCKER_COMPOSE_DIR}/data/tn-manager"
        "${DOCKER_COMPOSE_DIR}/data/tn-agent"
        "${DOCKER_COMPOSE_DIR}/data/ran-dms"
        "${DOCKER_COMPOSE_DIR}/data/cn-dms"
        "${DOCKER_COMPOSE_DIR}/test-results"
        "${DOCKER_COMPOSE_DIR}/logs"
        "${DOCKER_COMPOSE_DIR}/certs"
        "${DOCKER_COMPOSE_DIR}/certs/webhook"
    )

    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log_info "Created directory: $dir"
    done

    # Set appropriate permissions
    chmod 755 "${DOCKER_COMPOSE_DIR}/data"/*
    chmod 755 "${DOCKER_COMPOSE_DIR}/test-results"
    chmod 755 "${DOCKER_COMPOSE_DIR}/logs"

    log_success "Directory structure created"
}

generate_certificates() {
    log_info "Generating self-signed certificates..."

    local cert_dir="${DOCKER_COMPOSE_DIR}/certs"
    local webhook_cert_dir="${DOCKER_COMPOSE_DIR}/certs/webhook"

    # Generate CA key and certificate
    if [[ ! -f "${cert_dir}/ca.key" ]] || [[ ! -f "${cert_dir}/ca.crt" ]]; then
        openssl genpkey -algorithm RSA -out "${cert_dir}/ca.key" -pkcs8 -aes256 -pass pass:oran-mano
        openssl req -new -x509 -key "${cert_dir}/ca.key" -out "${cert_dir}/ca.crt" \
            -days 365 -passin pass:oran-mano \
            -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN MANO/OU=Development/CN=oran-mano-ca"
        log_info "Generated CA certificate"
    fi

    # Generate server certificates
    local services=("orchestrator" "ran-dms" "cn-dms" "o2-client")
    for service in "${services[@]}"; do
        if [[ ! -f "${cert_dir}/${service}.key" ]] || [[ ! -f "${cert_dir}/${service}.crt" ]]; then
            openssl genpkey -algorithm RSA -out "${cert_dir}/${service}.key"
            openssl req -new -key "${cert_dir}/${service}.key" -out "${cert_dir}/${service}.csr" \
                -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN MANO/OU=Development/CN=${service}"
            openssl x509 -req -in "${cert_dir}/${service}.csr" -CA "${cert_dir}/ca.crt" \
                -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${cert_dir}/${service}.crt" \
                -days 365 -passin pass:oran-mano
            rm "${cert_dir}/${service}.csr"
            log_info "Generated certificate for $service"
        fi
    done

    # Generate webhook certificates
    if [[ ! -f "${webhook_cert_dir}/server.key" ]] || [[ ! -f "${webhook_cert_dir}/server.crt" ]]; then
        openssl genpkey -algorithm RSA -out "${webhook_cert_dir}/server.key"
        openssl req -new -key "${webhook_cert_dir}/server.key" -out "${webhook_cert_dir}/server.csr" \
            -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN MANO/OU=Development/CN=vnf-operator-webhook"
        openssl x509 -req -in "${webhook_cert_dir}/server.csr" -CA "${cert_dir}/ca.crt" \
            -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${webhook_cert_dir}/server.crt" \
            -days 365 -passin pass:oran-mano
        rm "${webhook_cert_dir}/server.csr"
        log_info "Generated webhook certificates"
    fi

    # Use default server certificates
    if [[ ! -f "${cert_dir}/server.key" ]] || [[ ! -f "${cert_dir}/server.crt" ]]; then
        cp "${cert_dir}/orchestrator.key" "${cert_dir}/server.key"
        cp "${cert_dir}/orchestrator.crt" "${cert_dir}/server.crt"
        log_info "Created default server certificates"
    fi

    log_success "Certificate generation completed"
}

build_images() {
    log_info "Building Docker images..."

    cd "${DOCKER_COMPOSE_DIR}"

    # Set build arguments
    export BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    export VERSION="v1.0.0-local"
    export COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    export LOG_LEVEL="debug"

    # Build with parallel execution
    docker-compose -f docker-compose.local.yml build --parallel --pull

    log_success "Docker images built successfully"
}

start_infrastructure() {
    log_info "Starting infrastructure services..."

    cd "${DOCKER_COMPOSE_DIR}"

    # Start core services first
    docker-compose -f docker-compose.local.yml up -d \
        ran-dms cn-dms prometheus grafana

    log_info "Waiting for infrastructure services to be healthy..."
    sleep 15

    # Check infrastructure health
    wait_for_service "ran-dms" "8087" "/health"
    wait_for_service "cn-dms" "8088" "/health"
    wait_for_service "prometheus" "9090" "/-/healthy"
    wait_for_service "grafana" "3000" "/api/health"

    log_success "Infrastructure services started successfully"
}

start_core_services() {
    log_info "Starting core MANO services..."

    cd "${DOCKER_COMPOSE_DIR}"

    # Start core services
    docker-compose -f docker-compose.local.yml up -d \
        orchestrator vnf-operator o2-client tn-manager

    log_info "Waiting for core services to be healthy..."
    sleep 20

    # Check core service health
    wait_for_service "orchestrator" "8080" "/health"
    wait_for_service "vnf-operator" "8081" "/healthz"
    wait_for_service "o2-client" "8083" "/health"
    wait_for_service "tn-manager" "8084" "/health"

    log_success "Core MANO services started successfully"
}

start_edge_services() {
    log_info "Starting edge services..."

    cd "${DOCKER_COMPOSE_DIR}"

    # Start TN agents
    docker-compose -f docker-compose.local.yml up -d \
        tn-agent-edge01 tn-agent-edge02

    log_info "Waiting for edge services to be healthy..."
    sleep 15

    # Check edge service health
    wait_for_service "tn-agent-edge01" "8085" "/health"
    wait_for_service "tn-agent-edge02" "8086" "/health"

    log_success "Edge services started successfully"
}

wait_for_service() {
    local service_name="$1"
    local port="$2"
    local path="$3"
    local max_attempts=30
    local attempt=1

    log_info "Waiting for $service_name to be healthy..."

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "http://localhost:${port}${path}" > /dev/null 2>&1; then
            log_success "$service_name is healthy"
            return 0
        fi

        log_info "Attempt $attempt/$max_attempts: $service_name not ready yet..."
        sleep 5
        ((attempt++))
    done

    log_error "$service_name failed to become healthy after $max_attempts attempts"
    return 1
}

run_health_checks() {
    log_info "Running comprehensive health checks..."

    local services=(
        "ran-dms:8087:/health"
        "cn-dms:8088:/health"
        "orchestrator:8080:/health"
        "vnf-operator:8081:/healthz"
        "o2-client:8083:/health"
        "tn-manager:8084:/health"
        "tn-agent-edge01:8085:/health"
        "tn-agent-edge02:8086:/health"
        "prometheus:9090:/-/healthy"
        "grafana:3000:/api/health"
    )

    local failed_services=()

    for service_info in "${services[@]}"; do
        IFS=':' read -r service port path <<< "$service_info"

        if ! curl -f -s "http://localhost:${port}${path}" > /dev/null 2>&1; then
            failed_services+=("$service")
        fi
    done

    if [ ${#failed_services[@]} -eq 0 ]; then
        log_success "All services are healthy"
        return 0
    else
        log_error "Failed services: ${failed_services[*]}"
        return 1
    fi
}

run_connectivity_tests() {
    log_info "Running connectivity tests..."

    # Test inter-service connectivity
    local test_results="${DOCKER_COMPOSE_DIR}/test-results/connectivity-$(date +%s).json"

    cat > "$test_results" << 'EOF'
{
  "timestamp": "'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'",
  "tests": [
EOF

    # Test orchestrator -> RAN DMS
    if docker exec oran-orchestrator wget -qO- --timeout=5 http://ran-dms:8080/health > /dev/null 2>&1; then
        echo '    {"test": "orchestrator->ran-dms", "status": "PASS"},' >> "$test_results"
    else
        echo '    {"test": "orchestrator->ran-dms", "status": "FAIL"},' >> "$test_results"
    fi

    # Test orchestrator -> CN DMS
    if docker exec oran-orchestrator wget -qO- --timeout=5 http://cn-dms:8080/health > /dev/null 2>&1; then
        echo '    {"test": "orchestrator->cn-dms", "status": "PASS"},' >> "$test_results"
    else
        echo '    {"test": "orchestrator->cn-dms", "status": "FAIL"},' >> "$test_results"
    fi

    # Test VNF operator -> RAN DMS
    if docker exec oran-vnf-operator wget -qO- --timeout=5 http://ran-dms:8080/health > /dev/null 2>&1; then
        echo '    {"test": "vnf-operator->ran-dms", "status": "PASS"},' >> "$test_results"
    else
        echo '    {"test": "vnf-operator->ran-dms", "status": "FAIL"},' >> "$test_results"
    fi

    # Test TN agents -> TN manager
    if docker exec oran-tn-agent-edge01 wget -qO- --timeout=5 http://tn-manager:8080/health > /dev/null 2>&1; then
        echo '    {"test": "tn-agent-edge01->tn-manager", "status": "PASS"},' >> "$test_results"
    else
        echo '    {"test": "tn-agent-edge01->tn-manager", "status": "FAIL"},' >> "$test_results"
    fi

    if docker exec oran-tn-agent-edge02 wget -qO- --timeout=5 http://tn-manager:8080/health > /dev/null 2>&1; then
        echo '    {"test": "tn-agent-edge02->tn-manager", "status": "PASS"}' >> "$test_results"
    else
        echo '    {"test": "tn-agent-edge02->tn-manager", "status": "FAIL"}' >> "$test_results"
    fi

    cat >> "$test_results" << 'EOF'
  ],
  "summary": {
    "total": 5,
    "passed": 0,
    "failed": 0
  }
}
EOF

    # Count results
    local passed=$(grep '"status": "PASS"' "$test_results" | wc -l)
    local failed=$(grep '"status": "FAIL"' "$test_results" | wc -l)

    # Update summary
    sed -i "s/\"passed\": 0/\"passed\": $passed/" "$test_results"
    sed -i "s/\"failed\": 0/\"failed\": $failed/" "$test_results"

    log_info "Connectivity test results saved to: $test_results"
    log_success "Connectivity tests completed: $passed passed, $failed failed"

    return $failed
}

show_service_urls() {
    log_info "Service URLs:"
    echo "┌─────────────────────────────────────────────────────────────────┐"
    echo "│                        O-RAN MANO Services                      │"
    echo "├─────────────────────────────────────────────────────────────────┤"
    echo "│ Orchestrator:     http://localhost:8080                        │"
    echo "│   - Metrics:      http://localhost:9090                        │"
    echo "│   - Debug:        http://localhost:8180                        │"
    echo "│                                                                 │"
    echo "│ VNF Operator:     http://localhost:8081 (metrics)              │"
    echo "│   - Health:       http://localhost:8082                        │"
    echo "│                                                                 │"
    echo "│ O2 Client:        http://localhost:8083                        │"
    echo "│   - Metrics:      http://localhost:9093                        │"
    echo "│                                                                 │"
    echo "│ TN Manager:       http://localhost:8084                        │"
    echo "│   - Metrics:      http://localhost:9091                        │"
    echo "│   - Debug:        http://localhost:8184                        │"
    echo "│                                                                 │"
    echo "│ TN Agent Edge01:  http://localhost:8085                        │"
    echo "│   - iPerf3:       localhost:5201                               │"
    echo "│                                                                 │"
    echo "│ TN Agent Edge02:  http://localhost:8086                        │"
    echo "│   - iPerf3:       localhost:5202                               │"
    echo "│                                                                 │"
    echo "│ RAN DMS:          http://localhost:8087                        │"
    echo "│   - HTTPS:        https://localhost:8443                       │"
    echo "│   - Metrics:      http://localhost:9087                        │"
    echo "│                                                                 │"
    echo "│ CN DMS:           http://localhost:8088                        │"
    echo "│   - HTTPS:        https://localhost:8444                       │"
    echo "│   - Metrics:      http://localhost:9088                        │"
    echo "├─────────────────────────────────────────────────────────────────┤"
    echo "│                       Monitoring                                │"
    echo "├─────────────────────────────────────────────────────────────────┤"
    echo "│ Prometheus:       http://localhost:9090                        │"
    echo "│ Grafana:          http://localhost:3000                        │"
    echo "│   - User:         admin                                        │"
    echo "│   - Password:     admin123                                     │"
    echo "└─────────────────────────────────────────────────────────────────┘"
    echo ""
}

show_usage() {
    cat << 'EOF'
O-RAN Intent-MANO Local Deployment Script

Usage: ./deploy-local.sh [COMMAND] [OPTIONS]

Commands:
  start          Start all services (default)
  stop           Stop all services
  restart        Restart all services
  build          Build Docker images only
  clean          Clean up containers and volumes
  logs           Show service logs
  health         Run health checks
  test           Run connectivity tests
  status         Show service status

Options:
  -h, --help     Show this help message
  -v, --verbose  Enable verbose output
  --no-build     Skip building images
  --profile      Start with specific profile (monitoring, testing, development)

Examples:
  ./deploy-local.sh start
  ./deploy-local.sh start --profile testing
  ./deploy-local.sh stop
  ./deploy-local.sh health
  ./deploy-local.sh logs orchestrator
EOF
}

# Main execution functions
start_deployment() {
    log_info "Starting O-RAN Intent-MANO local deployment..."

    check_dependencies
    create_directories
    generate_certificates

    if [[ "${NO_BUILD:-}" != "true" ]]; then
        build_images
    fi

    start_infrastructure
    start_core_services
    start_edge_services

    if run_health_checks && run_connectivity_tests; then
        show_service_urls
        log_success "🎉 O-RAN Intent-MANO deployment completed successfully!"
        log_info "Run './deploy-local.sh status' to check service status"
        log_info "Run './deploy-local.sh logs <service>' to view logs"
        return 0
    else
        log_error "Deployment completed with some services failing health checks"
        return 1
    fi
}

stop_deployment() {
    log_info "Stopping O-RAN Intent-MANO services..."

    cd "${DOCKER_COMPOSE_DIR}"
    docker-compose -f docker-compose.local.yml down

    log_success "All services stopped"
}

restart_deployment() {
    log_info "Restarting O-RAN Intent-MANO services..."

    stop_deployment
    sleep 5
    start_deployment
}

clean_deployment() {
    log_warn "This will remove all containers, volumes, and data. Continue? (y/N)"
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        log_info "Cleaning up deployment..."

        cd "${DOCKER_COMPOSE_DIR}"
        docker-compose -f docker-compose.local.yml down -v --remove-orphans
        docker system prune -f

        # Remove data directories
        rm -rf "${DOCKER_COMPOSE_DIR}/data"/*
        rm -rf "${DOCKER_COMPOSE_DIR}/test-results"/*
        rm -rf "${DOCKER_COMPOSE_DIR}/logs"/*

        log_success "Cleanup completed"
    else
        log_info "Cleanup cancelled"
    fi
}

show_logs() {
    local service="${1:-}"

    cd "${DOCKER_COMPOSE_DIR}"

    if [[ -n "$service" ]]; then
        docker-compose -f docker-compose.local.yml logs -f "$service"
    else
        docker-compose -f docker-compose.local.yml logs -f
    fi
}

show_status() {
    log_info "Service status:"

    cd "${DOCKER_COMPOSE_DIR}"
    docker-compose -f docker-compose.local.yml ps

    echo ""
    run_health_checks
}

# Parse command line arguments
COMMAND="${1:-start}"
shift || true

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            set -x
            shift
            ;;
        --no-build)
            NO_BUILD=true
            shift
            ;;
        --profile)
            PROFILE="$2"
            shift 2
            ;;
        *)
            SERVICE="$1"
            shift
            ;;
    esac
done

# Execute command
case "$COMMAND" in
    start)
        start_deployment
        ;;
    stop)
        stop_deployment
        ;;
    restart)
        restart_deployment
        ;;
    build)
        check_dependencies
        create_directories
        generate_certificates
        build_images
        ;;
    clean)
        clean_deployment
        ;;
    logs)
        show_logs "${SERVICE:-}"
        ;;
    health)
        run_health_checks
        ;;
    test)
        run_connectivity_tests
        ;;
    status)
        show_status
        ;;
    *)
        log_error "Unknown command: $COMMAND"
        show_usage
        exit 1
        ;;
esac