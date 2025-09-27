#!/bin/bash

# Emergency Rollback Script for O-RAN MANO Monitoring Stack
# This script provides rapid rollback capabilities for critical failures

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-monitoring}"
EMERGENCY_BACKUP_DIR="${EMERGENCY_BACKUP_DIR:-/tmp/emergency-backup}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[EMERGENCY] [$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[EMERGENCY] [$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}[EMERGENCY] [$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

error() {
    echo -e "${RED}[EMERGENCY] [$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
}

# Emergency backup function
emergency_backup() {
    log "Creating emergency backup..."

    mkdir -p "$EMERGENCY_BACKUP_DIR"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$EMERGENCY_BACKUP_DIR/emergency_backup_$timestamp.yaml"

    # Quick backup of critical resources
    {
        echo "# Emergency backup created at $(date)"
        echo "---"
        kubectl get all -n "$NAMESPACE" -o yaml 2>/dev/null || true
        echo "---"
        kubectl get configmaps,secrets,pvc -n "$NAMESPACE" -o yaml 2>/dev/null || true
        echo "---"
        kubectl get servicemonitors,prometheusrules -n "$NAMESPACE" -o yaml 2>/dev/null || true
    } > "$backup_file"

    success "Emergency backup created: $backup_file"
    echo "$backup_file" > /tmp/emergency_backup_file
}

# Stop all monitoring components
stop_monitoring() {
    log "Stopping all monitoring components..."

    # Scale down deployments
    local deployments
    deployments=$(kubectl get deployments -n "$NAMESPACE" -o name 2>/dev/null || echo "")

    for deployment in $deployments; do
        log "Scaling down $deployment..."
        kubectl scale "$deployment" --replicas=0 -n "$NAMESPACE" 2>/dev/null || warning "Failed to scale $deployment"
    done

    # Scale down statefulsets
    local statefulsets
    statefulsets=$(kubectl get statefulsets -n "$NAMESPACE" -o name 2>/dev/null || echo "")

    for statefulset in $statefulsets; do
        log "Scaling down $statefulset..."
        kubectl scale "$statefulset" --replicas=0 -n "$NAMESPACE" 2>/dev/null || warning "Failed to scale $statefulset"
    done

    # Delete problematic pods
    log "Deleting all pods in monitoring namespace..."
    kubectl delete pods --all -n "$NAMESPACE" --grace-period=0 --force 2>/dev/null || warning "Failed to delete some pods"

    success "Monitoring components stopped"
}

# Delete problematic resources
delete_problematic_resources() {
    log "Deleting problematic resources..."

    # Delete custom resources that might be stuck
    local crd_resources=(
        "prometheuses.monitoring.coreos.com"
        "alertmanagers.monitoring.coreos.com"
        "servicemonitors.monitoring.coreos.com"
        "prometheusrules.monitoring.coreos.com"
    )

    for resource in "${crd_resources[@]}"; do
        log "Deleting all $resource..."
        kubectl delete "$resource" --all -n "$NAMESPACE" --grace-period=0 --force 2>/dev/null || warning "Failed to delete $resource"
    done

    # Delete stuck finalizers
    log "Removing finalizers from stuck resources..."
    kubectl get all -n "$NAMESPACE" -o name | while read -r resource; do
        kubectl patch "$resource" -n "$NAMESPACE" -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null || true
    done

    success "Problematic resources deleted"
}

# Uninstall Helm releases
uninstall_helm_releases() {
    log "Uninstalling Helm releases..."

    local releases
    releases=$(helm list -n "$NAMESPACE" -q 2>/dev/null || echo "")

    if [ -n "$releases" ]; then
        for release in $releases; do
            log "Uninstalling Helm release: $release"
            helm uninstall "$release" -n "$NAMESPACE" --wait --timeout=60s 2>/dev/null || warning "Failed to uninstall $release"
        done
    else
        warning "No Helm releases found"
    fi

    success "Helm releases uninstalled"
}

# Clean up namespace
cleanup_namespace() {
    log "Cleaning up namespace..."

    # Remove all resources from namespace
    kubectl delete all --all -n "$NAMESPACE" --grace-period=0 --force 2>/dev/null || warning "Failed to delete all resources"

    # Clean up configmaps and secrets
    kubectl delete configmaps,secrets --all -n "$NAMESPACE" --grace-period=0 --force 2>/dev/null || warning "Failed to delete configmaps/secrets"

    # Clean up PVCs (careful - this deletes data!)
    warning "This will delete all Persistent Volume Claims and DATA in namespace $NAMESPACE"
    read -p "Are you sure you want to continue? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
        kubectl delete pvc --all -n "$NAMESPACE" --grace-period=0 --force 2>/dev/null || warning "Failed to delete PVCs"
    else
        log "Skipping PVC deletion"
    fi

    success "Namespace cleaned up"
}

# Restore minimal monitoring
restore_minimal_monitoring() {
    log "Restoring minimal monitoring setup..."

    # Create basic Prometheus deployment
    cat > /tmp/minimal-prometheus.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minimal-prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: minimal-prometheus
  template:
    metadata:
      labels:
        app: minimal-prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus/'
        - '--web.console.libraries=/etc/prometheus/console_libraries'
        - '--web.console.templates=/etc/prometheus/consoles'
        - '--storage.tsdb.retention.time=24h'
        - '--web.enable-lifecycle'
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
        - name: storage
          mountPath: /prometheus
      volumes:
      - name: config
        configMap:
          name: minimal-prometheus-config
      - name: storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: minimal-prometheus
  namespace: monitoring
spec:
  selector:
    app: minimal-prometheus
  ports:
  - port: 9090
    targetPort: 9090
  type: ClusterIP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: minimal-prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 30s
    scrape_configs:
    - job_name: 'prometheus'
      static_configs:
      - targets: ['localhost:9090']
    - job_name: 'kubernetes-nodes'
      kubernetes_sd_configs:
      - role: node
      relabel_configs:
      - source_labels: [__address__]
        regex: '(.*):10250'
        target_label: __address__
        replacement: '${1}:9100'
EOF

    kubectl apply -f /tmp/minimal-prometheus.yaml
    rm -f /tmp/minimal-prometheus.yaml

    success "Minimal monitoring restored"
}

# Verify emergency rollback
verify_emergency_rollback() {
    log "Verifying emergency rollback..."

    # Check if namespace exists and is accessible
    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        success "Namespace $NAMESPACE is accessible"
    else
        error "Namespace $NAMESPACE is not accessible"
        return 1
    fi

    # Check for running pods
    local pod_count
    pod_count=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$pod_count" -gt 0 ]; then
        success "Found $pod_count running pods"
        kubectl get pods -n "$NAMESPACE"
    else
        warning "No pods found in namespace"
    fi

    # Check if minimal monitoring is accessible
    if kubectl get service minimal-prometheus -n "$NAMESPACE" &>/dev/null; then
        log "Testing minimal Prometheus accessibility..."
        kubectl port-forward -n "$NAMESPACE" svc/minimal-prometheus 9090:9090 &
        local pf_pid=$!
        sleep 5

        if curl -f http://localhost:9090/-/healthy &>/dev/null; then
            success "Minimal Prometheus is accessible"
        else
            warning "Minimal Prometheus is not accessible"
        fi

        kill $pf_pid 2>/dev/null || true
    fi

    success "Emergency rollback verification completed"
}

# Generate emergency report
generate_emergency_report() {
    log "Generating emergency rollback report..."

    local report_file="/tmp/emergency_rollback_report_$(date +%Y%m%d_%H%M%S).txt"

    cat > "$report_file" << EOF
O-RAN MANO Emergency Rollback Report
===================================

Emergency Rollback Date: $(date)
Namespace: $NAMESPACE
Actions Performed: $ACTIONS_PERFORMED

Current System Status:
EOF

    # Add current pod status
    echo "" >> "$report_file"
    echo "Current Pods:" >> "$report_file"
    kubectl get pods -n "$NAMESPACE" -o wide >> "$report_file" 2>/dev/null || echo "No pods found" >> "$report_file"

    # Add current services
    echo "" >> "$report_file"
    echo "Current Services:" >> "$report_file"
    kubectl get services -n "$NAMESPACE" >> "$report_file" 2>/dev/null || echo "No services found" >> "$report_file"

    # Add emergency backup location
    if [ -f /tmp/emergency_backup_file ]; then
        local backup_file
        backup_file=$(cat /tmp/emergency_backup_file)
        echo "" >> "$report_file"
        echo "Emergency Backup Location: $backup_file" >> "$report_file"
    fi

    success "Emergency report generated: $report_file"
}

# Main emergency rollback function
main() {
    local action="${1:-full}"

    log "🚨 STARTING EMERGENCY ROLLBACK FOR O-RAN MANO MONITORING STACK 🚨"
    log "Action: $action"
    warning "THIS IS AN EMERGENCY PROCEDURE - IT WILL CAUSE DOWNTIME"

    # Confirm emergency action
    if [ "$action" != "verify" ]; then
        echo ""
        warning "This emergency rollback will:"
        warning "1. Stop all monitoring components"
        warning "2. Delete problematic resources"
        warning "3. Uninstall Helm releases"
        warning "4. Clean up the namespace"
        warning "5. Restore minimal monitoring"
        echo ""
        read -p "Are you sure you want to proceed? Type 'EMERGENCY' to continue: " confirm
        if [ "$confirm" != "EMERGENCY" ]; then
            log "Emergency rollback cancelled"
            exit 0
        fi
    fi

    ACTIONS_PERFORMED=""

    case "$action" in
        "full")
            emergency_backup
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED backup"

            stop_monitoring
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED stop"

            delete_problematic_resources
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED delete-resources"

            uninstall_helm_releases
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED uninstall-helm"

            cleanup_namespace
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED cleanup"

            restore_minimal_monitoring
            ACTIONS_PERFORMED="$ACTIONS_PERFORMED restore-minimal"
            ;;
        "stop")
            emergency_backup
            stop_monitoring
            ACTIONS_PERFORMED="backup stop"
            ;;
        "clean")
            emergency_backup
            delete_problematic_resources
            cleanup_namespace
            ACTIONS_PERFORMED="backup delete-resources cleanup"
            ;;
        "restore")
            restore_minimal_monitoring
            ACTIONS_PERFORMED="restore-minimal"
            ;;
        "verify")
            verify_emergency_rollback
            return $?
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 [full|stop|clean|restore|verify]"
            exit 1
            ;;
    esac

    verify_emergency_rollback
    generate_emergency_report

    success "🎉 Emergency rollback completed!"
    warning "Next steps:"
    warning "1. Investigate the root cause of the emergency"
    warning "2. Plan proper recovery from backup or fresh deployment"
    warning "3. Restore full monitoring stack when ready"

    # Cleanup
    rm -f /tmp/emergency_backup_file
}

# Handle signals
trap 'error "Emergency rollback interrupted"; exit 1' INT TERM

# Execute main function
main "${1:-full}"