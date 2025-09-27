#!/bin/bash

# Rollback Script for O-RAN MANO Monitoring Stack
# This script provides comprehensive rollback capabilities for monitoring deployments

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-monitoring}"
ROLLBACK_TYPE="${ROLLBACK_TYPE:-helm}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
TIMEOUT="${TIMEOUT:-600}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites for rollback..."

    local missing_tools=()

    command -v kubectl &> /dev/null || missing_tools+=("kubectl")
    command -v helm &> /dev/null || missing_tools+=("helm")

    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi

    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
        error "Namespace '$NAMESPACE' does not exist"
        exit 1
    fi

    success "Prerequisites check passed"
}

# Create backup before rollback
create_backup() {
    log "Creating backup before rollback..."

    local backup_timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_path="$BACKUP_DIR/rollback_backup_$backup_timestamp"

    mkdir -p "$backup_path"

    # Backup all monitoring resources
    log "Backing up monitoring resources..."

    # Backup Helm releases
    if command -v helm &> /dev/null; then
        helm list -n "$NAMESPACE" -o yaml > "$backup_path/helm_releases.yaml" 2>/dev/null || true

        # Backup specific helm release values
        helm get values prometheus-operator -n "$NAMESPACE" > "$backup_path/prometheus_values.yaml" 2>/dev/null || true
    fi

    # Backup Kubernetes resources
    local resource_types=(
        "prometheuses"
        "alertmanagers"
        "servicemonitors"
        "prometheusrules"
        "configmaps"
        "secrets"
        "persistentvolumeclaims"
        "services"
        "deployments"
        "statefulsets"
        "ingresses"
        "networkpolicies"
    )

    for resource in "${resource_types[@]}"; do
        kubectl get "$resource" -n "$NAMESPACE" -o yaml > "$backup_path/${resource}.yaml" 2>/dev/null || true
    done

    # Backup custom resources
    kubectl get crd | grep monitoring.coreos.com | while read -r crd _; do
        kubectl get "$crd" -n "$NAMESPACE" -o yaml > "$backup_path/crd_${crd}.yaml" 2>/dev/null || true
    done

    # Backup PV data (metadata only)
    kubectl get pv -o yaml | grep -A 100 "monitoring" > "$backup_path/persistent_volumes.yaml" 2>/dev/null || true

    success "Backup created at: $backup_path"
    echo "$backup_path" > /tmp/rollback_backup_path
}

# Rollback Helm releases
rollback_helm() {
    local revision="${1:-}"

    log "Rolling back Helm releases..."

    # Get Helm releases in monitoring namespace
    local releases
    releases=$(helm list -n "$NAMESPACE" -q 2>/dev/null || echo "")

    if [ -z "$releases" ]; then
        warning "No Helm releases found in namespace '$NAMESPACE'"
        return 0
    fi

    for release in $releases; do
        log "Processing Helm release: $release"

        # Get release history
        local history
        history=$(helm history "$release" -n "$NAMESPACE" --max 10 2>/dev/null || echo "")

        if [ -z "$history" ]; then
            warning "No history found for release '$release'"
            continue
        fi

        echo "Release history for $release:"
        helm history "$release" -n "$NAMESPACE" --max 5

        # Determine rollback target
        local target_revision="$revision"
        if [ -z "$target_revision" ]; then
            # Find the last successful deployment
            target_revision=$(helm history "$release" -n "$NAMESPACE" -o json | \
                jq -r '.[] | select(.status == "deployed") | .revision' | \
                sort -nr | head -2 | tail -1 2>/dev/null || echo "")

            if [ -z "$target_revision" ]; then
                warning "Could not determine target revision for $release"
                continue
            fi
        fi

        log "Rolling back $release to revision $target_revision"

        if helm rollback "$release" "$target_revision" -n "$NAMESPACE" --timeout "${TIMEOUT}s" --wait; then
            success "Successfully rolled back $release to revision $target_revision"
        else
            error "Failed to rollback $release"

            # Try to get more information about the failure
            log "Checking rollback status..."
            helm status "$release" -n "$NAMESPACE" || true
            kubectl get pods -n "$NAMESPACE" | grep "$release" || true
        fi
    done
}

# Rollback using kubectl (manifest-based)
rollback_kubectl() {
    local manifest_path="${1:-}"

    log "Rolling back using kubectl with manifests..."

    if [ -z "$manifest_path" ] || [ ! -f "$manifest_path" ]; then
        error "Manifest file not provided or does not exist: $manifest_path"
        return 1
    fi

    log "Applying manifests from: $manifest_path"

    # Apply the rollback manifests
    if kubectl apply -f "$manifest_path" -n "$NAMESPACE"; then
        success "Successfully applied rollback manifests"
    else
        error "Failed to apply rollback manifests"
        return 1
    fi

    # Wait for rollout to complete
    log "Waiting for rollout to complete..."

    # Wait for deployments
    kubectl get deployments -n "$NAMESPACE" -o name | while read -r deployment; do
        log "Waiting for $deployment to complete rollout..."
        kubectl rollout status "$deployment" -n "$NAMESPACE" --timeout="${TIMEOUT}s" || true
    done

    # Wait for statefulsets
    kubectl get statefulsets -n "$NAMESPACE" -o name | while read -r statefulset; do
        log "Waiting for $statefulset to complete rollout..."
        kubectl rollout status "$statefulset" -n "$NAMESPACE" --timeout="${TIMEOUT}s" || true
    done
}

# Rollback using Kustomize
rollback_kustomize() {
    local overlay_path="${1:-}"

    log "Rolling back using Kustomize..."

    if [ -z "$overlay_path" ] || [ ! -d "$overlay_path" ]; then
        error "Kustomize overlay path not provided or does not exist: $overlay_path"
        return 1
    fi

    log "Applying Kustomize overlay from: $overlay_path"

    # Apply the kustomize overlay
    if kubectl apply -k "$overlay_path"; then
        success "Successfully applied Kustomize rollback"
    else
        error "Failed to apply Kustomize rollback"
        return 1
    fi

    # Wait for rollout to complete (similar to kubectl method)
    log "Waiting for Kustomize rollout to complete..."

    local namespace
    namespace=$(grep "namespace:" "$overlay_path/kustomization.yaml" | awk '{print $2}' || echo "$NAMESPACE")

    # Wait for deployments
    kubectl get deployments -n "$namespace" -o name | while read -r deployment; do
        kubectl rollout status "$deployment" -n "$namespace" --timeout="${TIMEOUT}s" || true
    done
}

# Rollback to specific backup
rollback_from_backup() {
    local backup_path="${1:-}"

    log "Rolling back from backup..."

    if [ -z "$backup_path" ] || [ ! -d "$backup_path" ]; then
        error "Backup path not provided or does not exist: $backup_path"
        return 1
    fi

    log "Rolling back from backup: $backup_path"

    # Apply resources in specific order
    local resource_order=(
        "namespaces.yaml"
        "configmaps.yaml"
        "secrets.yaml"
        "persistentvolumeclaims.yaml"
        "services.yaml"
        "deployments.yaml"
        "statefulsets.yaml"
        "servicemonitors.yaml"
        "prometheusrules.yaml"
        "prometheuses.yaml"
        "alertmanagers.yaml"
    )

    for resource_file in "${resource_order[@]}"; do
        local file_path="$backup_path/$resource_file"

        if [ -f "$file_path" ]; then
            log "Applying $resource_file..."
            kubectl apply -f "$file_path" -n "$NAMESPACE" || warning "Failed to apply $resource_file"
        else
            warning "Backup file not found: $file_path"
        fi
    done

    success "Rollback from backup completed"
}

# Verify rollback success
verify_rollback() {
    log "Verifying rollback success..."

    local verification_failed=false

    # Check pod status
    log "Checking pod status..."
    if ! kubectl get pods -n "$NAMESPACE" | grep -q "Running"; then
        error "No running pods found in namespace $NAMESPACE"
        verification_failed=true
    else
        success "Found running pods in namespace $NAMESPACE"
    fi

    # Check service availability
    log "Checking service availability..."
    local services=("prometheus-operator-kube-p-prometheus" "prometheus-operator-grafana")

    for service in "${services[@]}"; do
        if kubectl get service "$service" -n "$NAMESPACE" &>/dev/null; then
            success "Service $service is available"
        else
            error "Service $service is not available"
            verification_failed=true
        fi
    done

    # Check if Prometheus is accessible
    log "Checking Prometheus accessibility..."
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus 9090:9090 &
    local pf_pid=$!
    sleep 5

    if curl -f http://localhost:9090/-/healthy &>/dev/null; then
        success "Prometheus is healthy"
    else
        warning "Prometheus health check failed"
    fi

    kill $pf_pid 2>/dev/null || true

    # Check if Grafana is accessible
    log "Checking Grafana accessibility..."
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-grafana 3000:80 &
    local gf_pid=$!
    sleep 5

    if curl -f http://localhost:3000/api/health &>/dev/null; then
        success "Grafana is healthy"
    else
        warning "Grafana health check failed"
    fi

    kill $gf_pid 2>/dev/null || true

    if [ "$verification_failed" = true ]; then
        error "Rollback verification failed"
        return 1
    else
        success "Rollback verification passed"
        return 0
    fi
}

# Clean up failed rollback
cleanup_failed_rollback() {
    log "Cleaning up failed rollback..."

    # Get the backup path from temporary file
    local backup_path
    if [ -f /tmp/rollback_backup_path ]; then
        backup_path=$(cat /tmp/rollback_backup_path)
        log "Found backup path: $backup_path"

        # Offer to restore from backup
        echo "Rollback failed. Would you like to restore from the pre-rollback backup? (y/n)"
        read -r response
        if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
            rollback_from_backup "$backup_path"
        fi
    fi

    # Clean up port forwards
    pkill -f "kubectl port-forward" 2>/dev/null || true

    rm -f /tmp/rollback_backup_path
}

# Generate rollback report
generate_rollback_report() {
    log "Generating rollback report..."

    local report_file="$BACKUP_DIR/rollback_report_$(date +%Y%m%d_%H%M%S).txt"

    cat > "$report_file" << EOF
O-RAN MANO Monitoring Stack Rollback Report
==========================================

Rollback Date: $(date)
Namespace: $NAMESPACE
Rollback Type: $ROLLBACK_TYPE

System Status After Rollback:
EOF

    # Add pod status
    echo "" >> "$report_file"
    echo "Pod Status:" >> "$report_file"
    kubectl get pods -n "$NAMESPACE" -o wide >> "$report_file" 2>/dev/null || echo "Could not get pod status" >> "$report_file"

    # Add service status
    echo "" >> "$report_file"
    echo "Service Status:" >> "$report_file"
    kubectl get services -n "$NAMESPACE" >> "$report_file" 2>/dev/null || echo "Could not get service status" >> "$report_file"

    # Add helm status
    echo "" >> "$report_file"
    echo "Helm Releases:" >> "$report_file"
    helm list -n "$NAMESPACE" >> "$report_file" 2>/dev/null || echo "Could not get Helm releases" >> "$report_file"

    success "Rollback report generated: $report_file"
}

# Main rollback function
main() {
    local revision="${1:-}"
    local manifest_path="${2:-}"

    log "Starting O-RAN MANO monitoring stack rollback"
    log "Namespace: $NAMESPACE, Type: $ROLLBACK_TYPE"

    # Set up error handling
    trap cleanup_failed_rollback ERR

    check_prerequisites
    create_backup

    case "$ROLLBACK_TYPE" in
        "helm")
            rollback_helm "$revision"
            ;;
        "kubectl")
            rollback_kubectl "$manifest_path"
            ;;
        "kustomize")
            rollback_kustomize "$manifest_path"
            ;;
        "backup")
            rollback_from_backup "$manifest_path"
            ;;
        *)
            error "Unknown rollback type: $ROLLBACK_TYPE"
            error "Supported types: helm, kubectl, kustomize, backup"
            exit 1
            ;;
    esac

    # Verify rollback
    if verify_rollback; then
        success "🎉 Rollback completed successfully!"
        generate_rollback_report
    else
        error "❌ Rollback verification failed"
        exit 1
    fi

    # Cleanup
    rm -f /tmp/rollback_backup_path
}

# Handle command line arguments
case "${1:-help}" in
    "helm")
        ROLLBACK_TYPE="helm"
        main "${2:-}" "${3:-}"
        ;;
    "kubectl")
        ROLLBACK_TYPE="kubectl"
        if [ -z "${2:-}" ]; then
            error "Manifest path required for kubectl rollback"
            exit 1
        fi
        main "" "$2"
        ;;
    "kustomize")
        ROLLBACK_TYPE="kustomize"
        if [ -z "${2:-}" ]; then
            error "Overlay path required for kustomize rollback"
            exit 1
        fi
        main "" "$2"
        ;;
    "backup")
        ROLLBACK_TYPE="backup"
        if [ -z "${2:-}" ]; then
            error "Backup path required for backup rollback"
            exit 1
        fi
        main "" "$2"
        ;;
    "verify")
        verify_rollback
        ;;
    "help"|*)
        echo "Usage: $0 [command] [options]"
        echo ""
        echo "Commands:"
        echo "  helm [revision]           - Rollback using Helm to specific revision (or last successful)"
        echo "  kubectl <manifest-path>   - Rollback using kubectl with manifest file"
        echo "  kustomize <overlay-path>  - Rollback using Kustomize overlay"
        echo "  backup <backup-path>      - Rollback from a specific backup"
        echo "  verify                    - Verify current deployment status"
        echo "  help                      - Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  NAMESPACE                 - Kubernetes namespace (default: monitoring)"
        echo "  BACKUP_DIR               - Backup directory (default: ./backups)"
        echo "  TIMEOUT                  - Operation timeout in seconds (default: 600)"
        echo ""
        echo "Examples:"
        echo "  $0 helm                                    # Rollback to last successful revision"
        echo "  $0 helm 3                                  # Rollback to revision 3"
        echo "  $0 kubectl /path/to/manifests.yaml        # Rollback using manifest"
        echo "  $0 kustomize ./overlays/staging            # Rollback using Kustomize"
        echo "  $0 backup ./backups/backup_20240101_120000 # Rollback from backup"
        ;;
esac