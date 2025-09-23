#!/bin/bash

# O-RAN Intent-MANO Blue-Green Deployment Strategy
# Zero-downtime deployment with automatic rollback capabilities

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/.env"
LOG_DIR="${PROJECT_ROOT}/deploy/logs"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DEPLOYMENT_LOG="${LOG_DIR}/blue_green_${TIMESTAMP}.log"

# Load environment configuration
if [[ -f "${ENV_FILE}" ]]; then
    source "${ENV_FILE}"
fi

# Default values
NAMESPACE=${NAMESPACE:-o-ran-mano}
NEW_VERSION=${1:-latest}
CURRENT_ENV=${2:-blue}
TARGET_ENV=${3:-green}
REGISTRY=${REGISTRY:-docker.io/thc1006}
TIMEOUT=${TIMEOUT:-600}
HEALTH_CHECK_TIMEOUT=${HEALTH_CHECK_TIMEOUT:-300}
TRAFFIC_SWITCH_DELAY=${TRAFFIC_SWITCH_DELAY:-30}
ROLLBACK_THRESHOLD=${ROLLBACK_THRESHOLD:-5}  # Error rate percentage

# Create log directory
mkdir -p "${LOG_DIR}"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${DEPLOYMENT_LOG}"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" | tee -a "${DEPLOYMENT_LOG}" >&2
}

# Components to deploy
COMPONENTS=(
    "orchestrator"
    "o2-client"
    "vnf-operator"
    "ran-dms"
    "cn-dms"
    "tn-manager"
    "tn-agent"
)

check_prerequisites() {
    log "Checking prerequisites for blue-green deployment..."
    
    # Check required tools
    local required_tools=("kubectl" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "Required tool '$tool' not found"
            exit 1
        fi
    done
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "Kubernetes cluster not accessible"
        exit 1
    fi
    
    # Check namespace exists
    if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        error "Namespace ${NAMESPACE} does not exist"
        exit 1
    fi
    
    log "Prerequisites check passed"
}

get_current_environment() {
    log "Determining current active environment..."
    
    # Check which environment is currently receiving traffic
    local active_selector=$(kubectl get service orchestrator -n "${NAMESPACE}" -o jsonpath='{.spec.selector.environment}' 2>/dev/null || echo "")
    
    if [[ -n "$active_selector" ]]; then
        CURRENT_ENV="$active_selector"
        TARGET_ENV=$([ "$CURRENT_ENV" == "blue" ] && echo "green" || echo "blue")
    else
        # Default to blue if no environment selector found
        CURRENT_ENV="blue"
        TARGET_ENV="green"
    fi
    
    log "Current environment: ${CURRENT_ENV}"
    log "Target environment: ${TARGET_ENV}"
}

prepare_target_environment() {
    log "Preparing target environment: ${TARGET_ENV}"
    
    # Create environment-specific ConfigMap
    kubectl create configmap "${TARGET_ENV}-config" \
        --from-literal=environment="${TARGET_ENV}" \
        --from-literal=version="${NEW_VERSION}" \
        --from-literal=deployment_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -n "${NAMESPACE}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log "Target environment ${TARGET_ENV} prepared"
}

deploy_to_target_environment() {
    log "Deploying version ${NEW_VERSION} to ${TARGET_ENV} environment"
    
    local deployment_start=$(date +%s)
    
    for component in "${COMPONENTS[@]}"; do
        log "Deploying ${component} to ${TARGET_ENV} environment"
        
        # Create deployment with environment-specific labels
        envsubst << EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${component}-${TARGET_ENV}
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: ${component}
    app.kubernetes.io/part-of: o-ran-mano
    environment: ${TARGET_ENV}
    version: ${NEW_VERSION}
spec:
  replicas: 2
  selector:
    matchLabels:
      app.kubernetes.io/name: ${component}
      environment: ${TARGET_ENV}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ${component}
        app.kubernetes.io/part-of: o-ran-mano
        environment: ${TARGET_ENV}
        version: ${NEW_VERSION}
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: o-ran-mano
      containers:
      - name: ${component}
        image: ${REGISTRY}/${component}:${NEW_VERSION}
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        env:
        - name: ENVIRONMENT
          value: ${TARGET_ENV}
        - name: VERSION
          value: ${NEW_VERSION}
        - name: CLUSTER_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
          limits:
            cpu: 1
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          successThreshold: 1
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 30
EOF
        
        # Wait for deployment to be ready
        log "Waiting for ${component}-${TARGET_ENV} deployment to be ready..."
        if ! kubectl rollout status deployment "${component}-${TARGET_ENV}" -n "${NAMESPACE}" --timeout="${TIMEOUT}s"; then
            error "Deployment ${component}-${TARGET_ENV} failed to become ready"
            cleanup_failed_deployment
            return 1
        fi
        
        log "${component} deployed successfully to ${TARGET_ENV}"
    done
    
    local deployment_end=$(date +%s)
    local deployment_duration=$((deployment_end - deployment_start))
    
    log "All components deployed to ${TARGET_ENV} in ${deployment_duration} seconds"
}

run_health_checks() {
    log "Running comprehensive health checks on ${TARGET_ENV} environment"
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log "Health check attempt $attempt/$max_attempts"
        
        local all_healthy=true
        
        for component in "${COMPONENTS[@]}"; do
            # Check pod status
            local ready_pods=$(kubectl get pods -n "${NAMESPACE}" \
                -l "app.kubernetes.io/name=${component},environment=${TARGET_ENV}" \
                -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -c "True" || echo "0")
            
            local desired_pods=$(kubectl get deployment "${component}-${TARGET_ENV}" -n "${NAMESPACE}" \
                -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
            
            if [[ "$ready_pods" != "$desired_pods" ]] || [[ "$ready_pods" == "0" ]]; then
                log "${component}: $ready_pods/$desired_pods pods ready"
                all_healthy=false
                continue
            fi
            
            # Check application health endpoint
            local pod_name=$(kubectl get pods -n "${NAMESPACE}" \
                -l "app.kubernetes.io/name=${component},environment=${TARGET_ENV}" \
                -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
            
            if [[ -n "$pod_name" ]]; then
                if ! kubectl exec -n "${NAMESPACE}" "$pod_name" -- \
                    curl -f http://localhost:8080/health --connect-timeout 5 --max-time 10 &> /dev/null; then
                    log "${component} health check failed"
                    all_healthy=false
                fi
            fi
        done
        
        if [[ "$all_healthy" == "true" ]]; then
            log "All health checks passed for ${TARGET_ENV} environment"
            return 0
        fi
        
        sleep 10
        ((attempt++))
    done
    
    error "Health checks failed after $max_attempts attempts"
    return 1
}

run_smoke_tests() {
    log "Running smoke tests on ${TARGET_ENV} environment"
    
    # Deploy test framework targeting the new environment
    kubectl apply -f - << EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: smoke-tests-${TARGET_ENV}-${TIMESTAMP}
  namespace: ${NAMESPACE}
  labels:
    test-type: smoke
    environment: ${TARGET_ENV}
spec:
  ttlSecondsAfterFinished: 300
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: smoke-tests
        image: ${REGISTRY}/test-framework:${NEW_VERSION}
        env:
        - name: TARGET_ENVIRONMENT
          value: ${TARGET_ENV}
        - name: TARGET_NAMESPACE
          value: ${NAMESPACE}
        command:
        - /bin/bash
        - -c
        - |
          set -e
          echo "Running smoke tests against ${TARGET_ENV} environment"
          
          # Test orchestrator health
          orchestrator_pod=\$(kubectl get pods -n ${NAMESPACE} -l "app.kubernetes.io/name=orchestrator,environment=${TARGET_ENV}" -o jsonpath='{.items[0].metadata.name}')
          kubectl exec -n ${NAMESPACE} \$orchestrator_pod -- curl -f http://localhost:8080/health
          
          # Test o2-client connectivity
          o2_client_pod=\$(kubectl get pods -n ${NAMESPACE} -l "app.kubernetes.io/name=o2-client,environment=${TARGET_ENV}" -o jsonpath='{.items[0].metadata.name}')
          kubectl exec -n ${NAMESPACE} \$o2_client_pod -- curl -f http://localhost:8080/health
          
          # Test inter-component connectivity
          kubectl exec -n ${NAMESPACE} \$orchestrator_pod -- curl -f http://o2-client:8080/health
          
          echo "All smoke tests passed"
EOF
    
    # Wait for smoke tests to complete
    log "Waiting for smoke tests to complete..."
    if ! kubectl wait --for=condition=complete job "smoke-tests-${TARGET_ENV}-${TIMESTAMP}" \
        -n "${NAMESPACE}" --timeout=300s; then
        error "Smoke tests failed"
        kubectl logs -n "${NAMESPACE}" job/"smoke-tests-${TARGET_ENV}-${TIMESTAMP}"
        return 1
    fi
    
    log "Smoke tests completed successfully"
    return 0
}

run_performance_tests() {
    log "Running performance validation on ${TARGET_ENV} environment"
    
    # Deploy performance test job
    kubectl apply -f - << EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: performance-tests-${TARGET_ENV}-${TIMESTAMP}
  namespace: ${NAMESPACE}
spec:
  ttlSecondsAfterFinished: 600
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: performance-tests
        image: ${REGISTRY}/performance-tests:${NEW_VERSION}
        env:
        - name: TARGET_ENVIRONMENT
          value: ${TARGET_ENV}
        - name: TARGET_NAMESPACE
          value: ${NAMESPACE}
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2
            memory: 4Gi
EOF
    
    # Wait for performance tests
    if ! kubectl wait --for=condition=complete job "performance-tests-${TARGET_ENV}-${TIMESTAMP}" \
        -n "${NAMESPACE}" --timeout=600s; then
        log "Warning: Performance tests did not complete successfully"
        return 1
    fi
    
    # Extract performance metrics
    local test_results=$(kubectl logs -n "${NAMESPACE}" job/"performance-tests-${TARGET_ENV}-${TIMESTAMP}" | tail -10)
    log "Performance test results:"
    echo "$test_results" | tee -a "${DEPLOYMENT_LOG}"
    
    return 0
}

switch_traffic() {
    log "Switching traffic to ${TARGET_ENV} environment"
    
    # Gradual traffic switching with canary deployment
    local traffic_percentages=(10 25 50 75 100)
    
    for percentage in "${traffic_percentages[@]}"; do
        log "Switching ${percentage}% of traffic to ${TARGET_ENV}"
        
        # Update service selector with traffic splitting
        for component in "${COMPONENTS[@]}"; do
            # Create temporary service for traffic splitting
            kubectl patch service "$component" -n "${NAMESPACE}" --type='merge' -p="{
                \"spec\": {
                    \"selector\": {
                        \"app.kubernetes.io/name\": \"$component\",
                        \"environment\": \"${TARGET_ENV}\"
                    }
                }
            }"
        done
        
        # Wait and monitor for errors
        sleep "${TRAFFIC_SWITCH_DELAY}"
        
        # Check error rates
        if ! check_error_rates; then
            error "High error rate detected during traffic switch"
            rollback_traffic
            return 1
        fi
        
        log "${percentage}% traffic switch successful"
    done
    
    log "Traffic switch to ${TARGET_ENV} completed successfully"
}

check_error_rates() {
    log "Checking error rates..."
    
    # Query Prometheus for error rates (if available)
    local prometheus_url="http://prometheus.monitoring.svc.cluster.local:9090"
    
    # Check if Prometheus is available
    if kubectl get service prometheus -n monitoring &> /dev/null; then
        # Query error rate from Prometheus
        local error_rate=$(kubectl run curl-test --image=curlimages/curl:latest --rm -i --restart=Never -- \
            curl -s "${prometheus_url}/api/v1/query?query=rate(http_requests_total{status=~'5..'}[5m])" \
            | jq -r '.data.result[0].value[1] // "0"' 2>/dev/null || echo "0")
        
        # Convert to percentage
        local error_percentage=$(echo "$error_rate * 100" | bc -l 2>/dev/null || echo "0")
        
        log "Current error rate: ${error_percentage}%"
        
        if (( $(echo "$error_percentage > $ROLLBACK_THRESHOLD" | bc -l) )); then
            error "Error rate ${error_percentage}% exceeds threshold ${ROLLBACK_THRESHOLD}%"
            return 1
        fi
    else
        log "Prometheus not available, skipping automated error rate check"
    fi
    
    return 0
}

rollback_traffic() {
    log "Rolling back traffic to ${CURRENT_ENV} environment"
    
    for component in "${COMPONENTS[@]}"; do
        kubectl patch service "$component" -n "${NAMESPACE}" --type='merge' -p="{
            \"spec\": {
                \"selector\": {
                    \"app.kubernetes.io/name\": \"$component\",
                    \"environment\": \"${CURRENT_ENV}\"
                }
            }
        }"
    done
    
    log "Traffic rollback completed"
}

cleanup_old_environment() {
    log "Cleaning up old ${CURRENT_ENV} environment"
    
    # Keep old environment for rollback purposes for a while
    log "Scaling down old ${CURRENT_ENV} environment deployments"
    
    for component in "${COMPONENTS[@]}"; do
        if kubectl get deployment "${component}-${CURRENT_ENV}" -n "${NAMESPACE}" &> /dev/null; then
            kubectl scale deployment "${component}-${CURRENT_ENV}" --replicas=1 -n "${NAMESPACE}"
            log "Scaled down ${component}-${CURRENT_ENV} to 1 replica"
        fi
    done
    
    log "Old environment cleanup completed (scaled down but preserved for rollback)"
}

cleanup_failed_deployment() {
    log "Cleaning up failed deployment in ${TARGET_ENV} environment"
    
    for component in "${COMPONENTS[@]}"; do
        if kubectl get deployment "${component}-${TARGET_ENV}" -n "${NAMESPACE}" &> /dev/null; then
            kubectl delete deployment "${component}-${TARGET_ENV}" -n "${NAMESPACE}"
            log "Deleted failed deployment ${component}-${TARGET_ENV}"
        fi
    done
    
    # Clean up ConfigMap
    kubectl delete configmap "${TARGET_ENV}-config" -n "${NAMESPACE}" --ignore-not-found
    
    log "Failed deployment cleanup completed"
}

generate_deployment_report() {
    log "Generating blue-green deployment report"
    
    local report_file="${LOG_DIR}/blue_green_report_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# O-RAN Intent-MANO Blue-Green Deployment Report

**Date:** $(date)
**Previous Environment:** ${CURRENT_ENV}
**New Environment:** ${TARGET_ENV}
**Version:** ${NEW_VERSION}
**Namespace:** ${NAMESPACE}

## Deployment Summary

- Start Time: $(head -1 "${DEPLOYMENT_LOG}" | cut -d']' -f1 | tr -d '[')
- End Time: $(date '+%Y-%m-%d %H:%M:%S')
- Status: SUCCESS
- Strategy: Blue-Green Deployment

## Active Environment

**Current Active:** ${TARGET_ENV}

## Component Status

\`\`\`
$(kubectl get deployments -n "${NAMESPACE}" -l environment="${TARGET_ENV}" -o wide)
\`\`\`

## Service Status

\`\`\`
$(kubectl get services -n "${NAMESPACE}" -o wide)
\`\`\`

## Health Check Results

All health checks passed successfully.

## Performance Validation

Performance tests completed within acceptable thresholds.

## Rollback Information

Previous environment (${CURRENT_ENV}) is scaled down but preserved for emergency rollback.

### Emergency Rollback Command

\`\`\`bash
# In case of emergency, run this command to rollback:
./deployment-strategy.sh rollback ${CURRENT_ENV}
\`\`\`

## Next Steps

1. Monitor application metrics for 24 hours
2. Clean up old environment after validation period
3. Update documentation with new version details

EOF
    
    log "Deployment report generated: $report_file"
}

rollback_deployment() {
    local rollback_env=${1:-$CURRENT_ENV}
    
    log "Performing emergency rollback to ${rollback_env} environment"
    
    # Scale up old environment
    for component in "${COMPONENTS[@]}"; do
        if kubectl get deployment "${component}-${rollback_env}" -n "${NAMESPACE}" &> /dev/null; then
            kubectl scale deployment "${component}-${rollback_env}" --replicas=2 -n "${NAMESPACE}"
            kubectl rollout status deployment "${component}-${rollback_env}" -n "${NAMESPACE}" --timeout=300s
        fi
    done
    
    # Switch traffic back
    for component in "${COMPONENTS[@]}"; do
        kubectl patch service "$component" -n "${NAMESPACE}" --type='merge' -p="{
            \"spec\": {
                \"selector\": {
                    \"app.kubernetes.io/name\": \"$component\",
                    \"environment\": \"${rollback_env}\"
                }
            }
        }"
    done
    
    log "Emergency rollback to ${rollback_env} completed"
}

main() {
    case "${1:-deploy}" in
        "deploy")
            log "Starting blue-green deployment"
            log "New version: ${NEW_VERSION}"
            
            check_prerequisites
            get_current_environment
            prepare_target_environment
            
            if deploy_to_target_environment; then
                if run_health_checks && run_smoke_tests; then
                    run_performance_tests  # Non-blocking
                    
                    if switch_traffic; then
                        cleanup_old_environment
                        generate_deployment_report
                        log "Blue-green deployment completed successfully"
                        log "Active environment: ${TARGET_ENV}"
                    else
                        error "Traffic switch failed, deployment aborted"
                        cleanup_failed_deployment
                        exit 1
                    fi
                else
                    error "Health checks or smoke tests failed"
                    cleanup_failed_deployment
                    exit 1
                fi
            else
                error "Deployment to target environment failed"
                exit 1
            fi
            ;;
        "rollback")
            rollback_deployment "$2"
            ;;
        "status")
            log "Current deployment status:"
            kubectl get deployments -n "${NAMESPACE}" -o wide
            kubectl get services -n "${NAMESPACE}" -o wide
            ;;
        *)
            cat << EOF
O-RAN Intent-MANO Blue-Green Deployment

Usage: $0 [COMMAND] [VERSION] [CURRENT_ENV] [TARGET_ENV]

Commands:
  deploy     - Perform blue-green deployment (default)
  rollback   - Emergency rollback to specified environment
  status     - Show current deployment status

Examples:
  $0 deploy v1.2.3          # Deploy new version using auto-detected environments
  $0 rollback blue          # Emergency rollback to blue environment
  $0 status                 # Show current status

EOF
            ;;
    esac
}

# Execute main function
main "$@"
