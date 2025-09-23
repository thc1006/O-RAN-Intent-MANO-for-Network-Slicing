#!/bin/bash

# O-RAN Intent-MANO Production Deployment Automation
# One-click production deployment with comprehensive validation

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/.env"
LOG_DIR="${PROJECT_ROOT}/deploy/logs"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEPLOYMENT_LOG="${LOG_DIR}/deployment_${TIMESTAMP}.log"

# Load environment configuration
if [[ -f "${ENV_FILE}" ]]; then
    source "${ENV_FILE}"
fi

# Default values
ENVIRONMENT=${1:-production}
NAMESPACE=${NAMESPACE:-o-ran-mano}
REGISTRY=${REGISTRY:-docker.io/thc1006}
VERSION=${VERSION:-latest}
TIMEOUT=${TIMEOUT:-600}
HEALTH_CHECK_INTERVAL=${HEALTH_CHECK_INTERVAL:-30}

# Performance targets from thesis
TARGET_DEPLOY_TIME=600  # 10 minutes
TARGET_THROUGHPUT_HIGH=4.57  # Mbps
TARGET_THROUGHPUT_MID=2.77   # Mbps
TARGET_THROUGHPUT_LOW=0.93   # Mbps
TARGET_RTT_HIGH=16.1         # ms
TARGET_RTT_MID=15.7          # ms
TARGET_RTT_LOW=6.3           # ms

# Create log directory
mkdir -p "${LOG_DIR}"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${DEPLOYMENT_LOG}"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "${DEPLOYMENT_LOG}" >&2
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "helm" "docker" "git" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "Required tool '$tool' not found"
            exit 1
        fi
    done
    
    # Check Kubernetes context
    if ! kubectl cluster-info &> /dev/null; then
        error "Kubernetes cluster not accessible"
        exit 1
    fi
    
    # Verify cluster meets minimum requirements
    local nodes=$(kubectl get nodes --no-headers | wc -l)
    if [[ $nodes -lt 3 ]]; then
        error "Minimum 3 nodes required for production deployment"
        exit 1
    fi
    
    log "Prerequisites check passed"
}

setup_environment() {
    log "Setting up environment: ${ENVIRONMENT}"
    
    # Create namespace if it doesn't exist
    if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        kubectl create namespace "${NAMESPACE}"
        log "Created namespace: ${NAMESPACE}"
    fi
    
    # Apply RBAC configurations
    kubectl apply -f "${PROJECT_ROOT}/deploy/k8s/base/rbac.yaml" -n "${NAMESPACE}"
    
    # Set up secrets from environment variables
    setup_secrets
    
    log "Environment setup completed"
}

setup_secrets() {
    log "Setting up secrets..."
    
    # Create secrets from environment variables
    local secrets=(
        "o2-ims-credentials:O2_IMS_USERNAME:O2_IMS_PASSWORD"
        "o2-dms-credentials:O2_DMS_USERNAME:O2_DMS_PASSWORD"
        "nephio-credentials:NEPHIO_USERNAME:NEPHIO_PASSWORD"
        "database-credentials:DB_USERNAME:DB_PASSWORD"
    )
    
    for secret_config in "${secrets[@]}"; do
        IFS=':' read -r secret_name username_var password_var <<< "$secret_config"
        
        if [[ -n "${!username_var:-}" && -n "${!password_var:-}" ]]; then
            kubectl create secret generic "$secret_name" \
                --from-literal=username="${!username_var}" \
                --from-literal=password="${!password_var}" \
                -n "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
            log "Created/updated secret: $secret_name"
        else
            log "Warning: Credentials for $secret_name not found in environment"
        fi
    done
}

deploy_infrastructure() {
    log "Deploying infrastructure components..."
    
    # Deploy storage classes and persistent volumes
    kubectl apply -f "${PROJECT_ROOT}/deploy/infrastructure/storage/"
    
    # Deploy network policies
    kubectl apply -f "${PROJECT_ROOT}/deploy/infrastructure/network/"
    
    # Deploy service mesh (if configured)
    if [[ "${ENABLE_SERVICE_MESH:-false}" == "true" ]]; then
        deploy_service_mesh
    fi
    
    # Deploy monitoring stack
    deploy_monitoring_stack
    
    log "Infrastructure deployment completed"
}

deploy_service_mesh() {
    log "Deploying service mesh..."
    
    # Install Istio or Linkerd based on configuration
    local mesh_type=${SERVICE_MESH_TYPE:-istio}
    
    case "$mesh_type" in
        "istio")
            helm repo add istio https://istio-release.storage.googleapis.com/charts
            helm repo update
            helm upgrade --install istio-base istio/base -n istio-system --create-namespace
            helm upgrade --install istiod istio/istiod -n istio-system
            ;;
        "linkerd")
            curl -sL https://run.linkerd.io/install | sh
            linkerd install | kubectl apply -f -
            ;;
        *)
            log "Unknown service mesh type: $mesh_type"
            ;;
    esac
}

deploy_monitoring_stack() {
    log "Deploying monitoring stack..."
    
    # Deploy Prometheus Operator
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo update
    
    helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --create-namespace \
        --values "${PROJECT_ROOT}/deploy/monitoring/prometheus-values.yaml" \
        --timeout "${TIMEOUT}s"
    
    # Deploy custom monitoring components
    kubectl apply -f "${PROJECT_ROOT}/deploy/monitoring/"
    
    log "Monitoring stack deployed"
}

deploy_o_ran_components() {
    log "Deploying O-RAN Intent-MANO components..."
    
    local start_time=$(date +%s)
    
    # Deploy in dependency order
    local components=(
        "o2-client"
        "orchestrator"
        "vnf-operator"
        "ran-dms"
        "cn-dms"
        "tn-manager"
        "tn-agent"
    )
    
    for component in "${components[@]}"; do
        log "Deploying component: $component"
        
        # Use Helm if chart exists, otherwise use Kubernetes manifests
        if [[ -d "${PROJECT_ROOT}/deploy/helm/charts/${component}" ]]; then
            helm upgrade --install "${component}" \
                "${PROJECT_ROOT}/deploy/helm/charts/${component}" \
                --namespace "${NAMESPACE}" \
                --set image.repository="${REGISTRY}/${component}" \
                --set image.tag="${VERSION}" \
                --set environment="${ENVIRONMENT}" \
                --timeout "${TIMEOUT}s" \
                --wait
        else
            # Apply Kubernetes manifests with environment substitution
            envsubst < "${PROJECT_ROOT}/deploy/k8s/base/${component}.yaml" | kubectl apply -f - -n "${NAMESPACE}"
        fi
        
        # Wait for deployment to be ready
        kubectl rollout status deployment "${component}" -n "${NAMESPACE}" --timeout="${TIMEOUT}s"
        
        log "Component $component deployed successfully"
    done
    
    local end_time=$(date +%s)
    local deploy_duration=$((end_time - start_time))
    
    log "All components deployed in ${deploy_duration} seconds"
    
    if [[ $deploy_duration -gt $TARGET_DEPLOY_TIME ]]; then
        log "Warning: Deployment time ${deploy_duration}s exceeds target ${TARGET_DEPLOY_TIME}s"
    fi
}

run_health_checks() {
    log "Running health checks..."
    
    local max_attempts=20
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log "Health check attempt $attempt/$max_attempts"
        
        local all_healthy=true
        
        # Check deployment status
        while IFS= read -r deployment; do
            if ! kubectl rollout status deployment "$deployment" -n "${NAMESPACE}" --timeout=30s &> /dev/null; then
                log "Deployment $deployment not ready"
                all_healthy=false
            fi
        done < <(kubectl get deployments -n "${NAMESPACE}" -o jsonpath='{.items[*].metadata.name}')
        
        # Check pod health
        local unhealthy_pods=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase!=Running -o name | wc -l)
        if [[ $unhealthy_pods -gt 0 ]]; then
            log "$unhealthy_pods pods not in Running state"
            all_healthy=false
        fi
        
        # Check service endpoints
        check_service_endpoints || all_healthy=false
        
        if [[ "$all_healthy" == "true" ]]; then
            log "All health checks passed"
            return 0
        fi
        
        sleep "${HEALTH_CHECK_INTERVAL}"
        ((attempt++))
    done
    
    error "Health checks failed after $max_attempts attempts"
    return 1
}

check_service_endpoints() {
    log "Checking service endpoints..."
    
    local services=("orchestrator" "o2-client" "tn-manager")
    
    for service in "${services[@]}"; do
        local service_ip=$(kubectl get service "$service" -n "${NAMESPACE}" -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")
        
        if [[ -z "$service_ip" ]]; then
            log "Service $service not found"
            return 1
        fi
        
        local port=$(kubectl get service "$service" -n "${NAMESPACE}" -o jsonpath='{.spec.ports[0].port}')
        
        if ! kubectl run curl-test --image=curlimages/curl:latest --rm -i --restart=Never -- \
            curl -f "http://${service_ip}:${port}/health" --connect-timeout 10 &> /dev/null; then
            log "Health endpoint for $service not responding"
            return 1
        fi
    done
    
    return 0
}

run_performance_validation() {
    log "Running performance validation..."
    
    # Deploy test framework
    kubectl apply -f "${PROJECT_ROOT}/deploy/testing/performance-test-framework.yaml" -n "${NAMESPACE}"
    
    # Wait for test framework to be ready
    kubectl wait --for=condition=ready pod -l app=performance-test -n "${NAMESPACE}" --timeout=300s
    
    # Run performance tests
    local test_results=$(kubectl exec -n "${NAMESPACE}" deployment/performance-test -- \
        python3 /tests/run_performance_suite.py --format json)
    
    # Parse and validate results
    validate_performance_results "$test_results"
}

validate_performance_results() {
    local results="$1"
    log "Validating performance results..."
    
    # Extract metrics from JSON results
    local deploy_time=$(echo "$results" | jq -r '.deployment_time // 0')
    local throughput_high=$(echo "$results" | jq -r '.throughput.high // 0')
    local throughput_mid=$(echo "$results" | jq -r '.throughput.mid // 0')
    local throughput_low=$(echo "$results" | jq -r '.throughput.low // 0')
    local rtt_high=$(echo "$results" | jq -r '.rtt.high // 0')
    local rtt_mid=$(echo "$results" | jq -r '.rtt.mid // 0')
    local rtt_low=$(echo "$results" | jq -r '.rtt.low // 0')
    
    local validation_passed=true
    
    # Validate against thesis targets
    if (( $(echo "$throughput_high < $TARGET_THROUGHPUT_HIGH" | bc -l) )); then
        log "Warning: High priority throughput ${throughput_high} Mbps below target ${TARGET_THROUGHPUT_HIGH} Mbps"
        validation_passed=false
    fi
    
    if (( $(echo "$rtt_high > $TARGET_RTT_HIGH" | bc -l) )); then
        log "Warning: High priority RTT ${rtt_high} ms above target ${TARGET_RTT_HIGH} ms"
        validation_passed=false
    fi
    
    # Save results
    echo "$results" > "${LOG_DIR}/performance_results_${TIMESTAMP}.json"
    
    if [[ "$validation_passed" == "true" ]]; then
        log "Performance validation passed"
    else
        log "Performance validation failed - check results for details"
    fi
}

setup_gitops_integration() {
    log "Setting up GitOps integration..."
    
    # Deploy ArgoCD or Flux based on configuration
    local gitops_tool=${GITOPS_TOOL:-argocd}
    
    case "$gitops_tool" in
        "argocd")
            kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
            kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
            
            # Wait for ArgoCD to be ready
            kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=600s
            
            # Apply ArgoCD applications
            kubectl apply -f "${PROJECT_ROOT}/deploy/gitops/argocd/"
            ;;
        "flux")
            # Install Flux CLI and bootstrap
            curl -s https://fluxcd.io/install.sh | sudo bash
            flux bootstrap git \
                --url="${GIT_REPOSITORY_URL}" \
                --branch="${GIT_BRANCH:-main}" \
                --path="clusters/${ENVIRONMENT}"
            ;;
    esac
    
    log "GitOps integration configured"
}

cleanup_on_failure() {
    local exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        error "Deployment failed with exit code $exit_code"
        
        # Collect diagnostic information
        log "Collecting diagnostic information..."
        
        kubectl get all -n "${NAMESPACE}" > "${LOG_DIR}/resources_${TIMESTAMP}.txt" 2>&1
        kubectl describe pods -n "${NAMESPACE}" > "${LOG_DIR}/pod_descriptions_${TIMESTAMP}.txt" 2>&1
        kubectl logs -n "${NAMESPACE}" --all-containers=true --previous=false > "${LOG_DIR}/pod_logs_${TIMESTAMP}.txt" 2>&1
        
        # Trigger rollback if requested
        if [[ "${AUTO_ROLLBACK:-false}" == "true" ]]; then
            log "Triggering automatic rollback..."
            rollback_deployment
        fi
    fi
    
    exit $exit_code
}

rollback_deployment() {
    log "Rolling back deployment..."
    
    # Get list of deployed components
    local components=$(kubectl get deployments -n "${NAMESPACE}" -o jsonpath='{.items[*].metadata.name}')
    
    for component in $components; do
        log "Rolling back $component"
        kubectl rollout undo deployment "$component" -n "${NAMESPACE}"
        kubectl rollout status deployment "$component" -n "${NAMESPACE}" --timeout=300s
    done
    
    log "Rollback completed"
}

generate_deployment_report() {
    log "Generating deployment report..."
    
    local report_file="${LOG_DIR}/deployment_report_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# O-RAN Intent-MANO Deployment Report

**Date:** $(date)
**Environment:** ${ENVIRONMENT}
**Version:** ${VERSION}
**Namespace:** ${NAMESPACE}

## Deployment Summary

- Start Time: $(head -1 "${DEPLOYMENT_LOG}" | cut -d']' -f1 | tr -d '[')
- End Time: $(date '+%Y-%m-%d %H:%M:%S')
- Status: SUCCESS

## Component Status

\`\`\`
$(kubectl get deployments -n "${NAMESPACE}" -o wide)
\`\`\`

## Service Status

\`\`\`
$(kubectl get services -n "${NAMESPACE}" -o wide)
\`\`\`

## Performance Metrics

See: [Performance Results](./performance_results_${TIMESTAMP}.json)

## Next Steps

1. Monitor application metrics in Grafana
2. Review logs for any warnings
3. Schedule next maintenance window

EOF
    
    log "Deployment report generated: $report_file"
}

main() {
    log "Starting O-RAN Intent-MANO production deployment"
    log "Environment: ${ENVIRONMENT}"
    log "Version: ${VERSION}"
    log "Namespace: ${NAMESPACE}"
    
    # Set up error handling
    trap cleanup_on_failure ERR EXIT
    
    # Run deployment phases
    check_prerequisites
    setup_environment
    deploy_infrastructure
    deploy_o_ran_components
    run_health_checks
    run_performance_validation
    setup_gitops_integration
    
    # Generate final report
    generate_deployment_report
    
    log "O-RAN Intent-MANO production deployment completed successfully"
    
    # Remove error trap on successful completion
    trap - ERR EXIT
}

# Print usage information
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat << EOF
O-RAN Intent-MANO Production Deployment Automation

Usage: $0 [ENVIRONMENT]

Environments:
  production  - Full production deployment with all validations
  staging     - Staging environment deployment
  development - Development environment deployment

Environment Variables:
  NAMESPACE              - Kubernetes namespace (default: o-ran-mano)
  REGISTRY              - Container registry (default: docker.io/thc1006)
  VERSION               - Image version (default: latest)
  TIMEOUT               - Deployment timeout in seconds (default: 600)
  HEALTH_CHECK_INTERVAL - Health check interval in seconds (default: 30)
  AUTO_ROLLBACK         - Enable automatic rollback on failure (default: false)
  GITOPS_TOOL           - GitOps tool (argocd|flux, default: argocd)
  SERVICE_MESH_TYPE     - Service mesh type (istio|linkerd, default: istio)

Example:
  $0 production
  NAMESPACE=test-env VERSION=v1.2.3 $0 staging

EOF
    exit 0
fi

# Run main function
main "$@"
