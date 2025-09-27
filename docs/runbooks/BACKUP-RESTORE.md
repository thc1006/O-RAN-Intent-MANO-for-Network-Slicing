# O-RAN Monitoring Stack Backup and Restore Guide

This guide provides comprehensive backup and restore procedures for the O-RAN monitoring stack, ensuring data protection and disaster recovery capabilities.

## Overview

The backup strategy covers all critical components:
- Prometheus time-series data and configuration
- Grafana dashboards, data sources, and database
- AlertManager configuration and state
- Kubernetes manifests and secrets

## Backup Strategy

### Backup Types

| Type | Frequency | Retention | Purpose |
|------|-----------|-----------|---------|
| Full Backup | Daily | 30 days | Complete system restore |
| Incremental | Hourly | 7 days | Quick recovery |
| Configuration | On change | 90 days | Config rollback |
| Snapshot | Weekly | 12 weeks | Long-term archive |

### Storage Locations

- **Primary**: Local cluster storage (fast recovery)
- **Secondary**: S3/Object storage (offsite backup)
- **Archive**: Glacier/Cold storage (long-term retention)

## Backup Components

### Prometheus Backup

#### Data Backup Script

```bash
#!/bin/bash
# prometheus-backup.sh

set -euo pipefail

# Configuration
NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_BASE_DIR="${BACKUP_DIR:-/backups/prometheus}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BUCKET:-oran-monitoring-backups}"
COMPRESSION_LEVEL="${COMPRESSION_LEVEL:-6}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Create backup directory
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_BASE_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_DIR"

log_info "Starting Prometheus backup to $BACKUP_DIR"

# Function to backup Prometheus data
backup_prometheus_data() {
    log_info "Backing up Prometheus TSDB data..."

    # Get Prometheus pod
    local prometheus_pod
    prometheus_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')

    if [[ -z "$prometheus_pod" ]]; then
        log_error "No Prometheus pod found"
        return 1
    fi

    # Create snapshot via API
    log_info "Creating Prometheus snapshot..."
    kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
        curl -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot

    # Get snapshot name
    local snapshot_name
    snapshot_name=$(kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
        ls -t /prometheus/snapshots | head -1)

    if [[ -z "$snapshot_name" ]]; then
        log_error "Failed to create snapshot"
        return 1
    fi

    log_info "Created snapshot: $snapshot_name"

    # Backup snapshot data
    log_info "Compressing and backing up snapshot data..."
    kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
        tar czf - -C /prometheus/snapshots "$snapshot_name" | \
        pv -p -s 1G > "$BACKUP_DIR/prometheus-data-${BACKUP_DATE}.tar.gz"

    # Cleanup snapshot
    kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
        rm -rf "/prometheus/snapshots/$snapshot_name"

    log_success "Prometheus data backup completed"
}

# Function to backup Prometheus configuration
backup_prometheus_config() {
    log_info "Backing up Prometheus configuration..."

    # Backup ConfigMaps
    kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o yaml > \
        "$BACKUP_DIR/prometheus-configmaps-${BACKUP_DATE}.yaml"

    # Backup Secrets
    kubectl get secret -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o yaml > \
        "$BACKUP_DIR/prometheus-secrets-${BACKUP_DATE}.yaml"

    # Backup ServiceMonitors (if using Prometheus Operator)
    if kubectl get crd servicemonitors.monitoring.coreos.com &>/dev/null; then
        kubectl get servicemonitor -n "$NAMESPACE" -o yaml > \
            "$BACKUP_DIR/prometheus-servicemonitors-${BACKUP_DATE}.yaml"
    fi

    # Backup PrometheusRules
    if kubectl get crd prometheusrules.monitoring.coreos.com &>/dev/null; then
        kubectl get prometheusrule -n "$NAMESPACE" -o yaml > \
            "$BACKUP_DIR/prometheus-rules-${BACKUP_DATE}.yaml"
    fi

    # Backup Deployment/StatefulSet
    kubectl get deployment,statefulset -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o yaml > \
        "$BACKUP_DIR/prometheus-workloads-${BACKUP_DATE}.yaml"

    log_success "Prometheus configuration backup completed"
}

# Function to backup WAL and current data
backup_prometheus_wal() {
    log_info "Backing up Prometheus WAL..."

    local prometheus_pod
    prometheus_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')

    # Backup WAL files
    kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
        tar czf - /prometheus/wal | \
        pv -p > "$BACKUP_DIR/prometheus-wal-${BACKUP_DATE}.tar.gz"

    log_success "Prometheus WAL backup completed"
}

# Execute Prometheus backup
backup_prometheus_data
backup_prometheus_config
backup_prometheus_wal

log_success "Prometheus backup completed successfully"
```

#### Incremental Backup Script

```bash
#!/bin/bash
# prometheus-incremental-backup.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_BASE_DIR="${BACKUP_DIR:-/backups/prometheus/incremental}"
LAST_BACKUP_FILE="$BACKUP_BASE_DIR/.last_backup"

# Create incremental backup directory
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_BASE_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_DIR"

# Get last backup timestamp
if [[ -f "$LAST_BACKUP_FILE" ]]; then
    LAST_BACKUP=$(cat "$LAST_BACKUP_FILE")
else
    LAST_BACKUP=$(date -d "1 hour ago" +%s)
fi

CURRENT_TIME=$(date +%s)

log_info "Creating incremental backup from $(date -d @$LAST_BACKUP) to $(date -d @$CURRENT_TIME)"

# Backup only changed data
prometheus_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')

# Find files modified since last backup
kubectl exec -n "$NAMESPACE" "$prometheus_pod" -- \
    find /prometheus -type f -newermt "@$LAST_BACKUP" -print0 | \
    kubectl exec -i -n "$NAMESPACE" "$prometheus_pod" -- \
    tar czf - --null -T - > "$BACKUP_DIR/prometheus-incremental-${BACKUP_DATE}.tar.gz"

# Update last backup timestamp
echo "$CURRENT_TIME" > "$LAST_BACKUP_FILE"

log_success "Incremental backup completed"
```

### Grafana Backup

#### Grafana Backup Script

```bash
#!/bin/bash
# grafana-backup.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_BASE_DIR="${BACKUP_DIR:-/backups/grafana}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-}"

# Create backup directory
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_BASE_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_DIR"

log_info "Starting Grafana backup to $BACKUP_DIR"

# Function to get Grafana admin password
get_grafana_password() {
    if [[ -z "$GRAFANA_PASSWORD" ]]; then
        GRAFANA_PASSWORD=$(kubectl get secret -n "$NAMESPACE" grafana-admin-credentials \
            -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || echo "admin")
    fi
}

# Function to setup port-forward
setup_grafana_port_forward() {
    local local_port=3000

    # Find available port
    while lsof -i :$local_port &>/dev/null; do
        ((local_port++))
    done

    # Start port-forward
    kubectl port-forward -n "$NAMESPACE" service/grafana $local_port:3000 &>/dev/null &
    local port_forward_pid=$!

    # Wait for port-forward to be ready
    sleep 3

    echo "$port_forward_pid:$local_port"
}

# Function to cleanup port-forward
cleanup_port_forward() {
    local port_forward_pid="$1"
    kill "$port_forward_pid" 2>/dev/null || true
}

# Function to backup Grafana database
backup_grafana_database() {
    log_info "Backing up Grafana database..."

    local grafana_pod
    grafana_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}')

    # Check database type
    local db_type
    db_type=$(kubectl exec -n "$NAMESPACE" "$grafana_pod" -- \
        grep -o 'type = [^[:space:]]*' /etc/grafana/grafana.ini | cut -d' ' -f3 || echo "sqlite3")

    case "$db_type" in
        "sqlite3")
            log_info "Backing up SQLite database..."
            kubectl exec -n "$NAMESPACE" "$grafana_pod" -- \
                sqlite3 /var/lib/grafana/grafana.db .dump > \
                "$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql"
            ;;
        "postgres")
            log_info "Backing up PostgreSQL database..."
            kubectl exec -n "$NAMESPACE" "$grafana_pod" -- \
                pg_dump grafana | gzip > \
                "$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql.gz"
            ;;
        "mysql")
            log_info "Backing up MySQL database..."
            kubectl exec -n "$NAMESPACE" "$grafana_pod" -- \
                mysqldump grafana | gzip > \
                "$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql.gz"
            ;;
        *)
            log_warning "Unknown database type: $db_type"
            ;;
    esac

    log_success "Grafana database backup completed"
}

# Function to backup Grafana API data
backup_grafana_api_data() {
    log_info "Backing up Grafana API data..."

    get_grafana_password
    local port_info
    port_info=$(setup_grafana_port_forward)
    local port_forward_pid="${port_info%:*}"
    local local_port="${port_info#*:}"

    # Ensure cleanup on exit
    trap "cleanup_port_forward $port_forward_pid" EXIT

    # Backup dashboards
    log_info "Backing up dashboards..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/search?type=dash-db" | \
        jq -r '.[].uid' | while read -r uid; do
            if [[ -n "$uid" ]]; then
                curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
                    "http://localhost:$local_port/api/dashboards/uid/$uid" > \
                    "$BACKUP_DIR/dashboard-${uid}.json"
            fi
        done

    # Backup data sources
    log_info "Backing up data sources..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/datasources" > \
        "$BACKUP_DIR/datasources-${BACKUP_DATE}.json"

    # Backup folders
    log_info "Backing up folders..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/folders" > \
        "$BACKUP_DIR/folders-${BACKUP_DATE}.json"

    # Backup users
    log_info "Backing up users..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/users" > \
        "$BACKUP_DIR/users-${BACKUP_DATE}.json"

    # Backup organization
    log_info "Backing up organization..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/org" > \
        "$BACKUP_DIR/organization-${BACKUP_DATE}.json"

    # Backup teams
    log_info "Backing up teams..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/teams/search" > \
        "$BACKUP_DIR/teams-${BACKUP_DATE}.json"

    # Backup alerting rules
    log_info "Backing up alerting rules..."
    curl -s -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
        "http://localhost:$local_port/api/ruler/grafana/api/v1/rules" > \
        "$BACKUP_DIR/alert-rules-${BACKUP_DATE}.json"

    cleanup_port_forward "$port_forward_pid"
    log_success "Grafana API data backup completed"
}

# Function to backup Grafana configuration
backup_grafana_config() {
    log_info "Backing up Grafana configuration..."

    # Backup ConfigMaps
    kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o yaml > \
        "$BACKUP_DIR/grafana-configmaps-${BACKUP_DATE}.yaml"

    # Backup Secrets
    kubectl get secret -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o yaml > \
        "$BACKUP_DIR/grafana-secrets-${BACKUP_DATE}.yaml"

    # Backup PVCs
    kubectl get pvc -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o yaml > \
        "$BACKUP_DIR/grafana-pvcs-${BACKUP_DATE}.yaml"

    # Backup Deployment
    kubectl get deployment -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o yaml > \
        "$BACKUP_DIR/grafana-deployment-${BACKUP_DATE}.yaml"

    log_success "Grafana configuration backup completed"
}

# Execute Grafana backup
backup_grafana_database
backup_grafana_api_data
backup_grafana_config

log_success "Grafana backup completed successfully"
```

### AlertManager Backup

```bash
#!/bin/bash
# alertmanager-backup.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_BASE_DIR="${BACKUP_DIR:-/backups/alertmanager}"

# Create backup directory
BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_BASE_DIR/$BACKUP_DATE"
mkdir -p "$BACKUP_DIR"

log_info "Starting AlertManager backup to $BACKUP_DIR"

# Function to backup AlertManager data
backup_alertmanager_data() {
    log_info "Backing up AlertManager data..."

    local alertmanager_pods
    alertmanager_pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager -o jsonpath='{.items[*].metadata.name}')

    for pod in $alertmanager_pods; do
        log_info "Backing up data from pod: $pod"
        kubectl exec -n "$NAMESPACE" "$pod" -- \
            tar czf - /alertmanager | \
            pv -p > "$BACKUP_DIR/alertmanager-data-${pod}-${BACKUP_DATE}.tar.gz"
    done

    log_success "AlertManager data backup completed"
}

# Function to backup AlertManager configuration
backup_alertmanager_config() {
    log_info "Backing up AlertManager configuration..."

    # Backup ConfigMaps
    kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager -o yaml > \
        "$BACKUP_DIR/alertmanager-configmaps-${BACKUP_DATE}.yaml"

    # Backup Secrets
    kubectl get secret -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager -o yaml > \
        "$BACKUP_DIR/alertmanager-secrets-${BACKUP_DATE}.yaml"

    # Backup StatefulSet
    kubectl get statefulset -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager -o yaml > \
        "$BACKUP_DIR/alertmanager-statefulset-${BACKUP_DATE}.yaml"

    log_success "AlertManager configuration backup completed"
}

# Execute AlertManager backup
backup_alertmanager_data
backup_alertmanager_config

log_success "AlertManager backup completed successfully"
```

## Backup Automation

### Backup CronJob

```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: monitoring-backup
  namespace: oran-monitoring
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: monitoring-backup
          securityContext:
            runAsNonRoot: true
            runAsUser: 65534
            fsGroup: 65534
          containers:
          - name: backup
            image: alpine:3.18
            command:
            - /bin/sh
            - -c
            - |
              # Install required tools
              apk add --no-cache curl jq kubectl

              # Run backup scripts
              /scripts/prometheus-backup.sh
              /scripts/grafana-backup.sh
              /scripts/alertmanager-backup.sh
              /scripts/upload-to-s3.sh

              # Cleanup old backups
              find /backups -type d -mtime +30 -exec rm -rf {} +
            env:
            - name: MONITORING_NAMESPACE
              value: "oran-monitoring"
            - name: BACKUP_DIR
              value: "/backups"
            - name: S3_BUCKET
              value: "oran-monitoring-backups"
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: backup-credentials
                  key: aws-access-key-id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-credentials
                  key: aws-secret-access-key
            volumeMounts:
            - name: backup-scripts
              mountPath: /scripts
            - name: backup-storage
              mountPath: /backups
            resources:
              limits:
                cpu: 500m
                memory: 1Gi
              requests:
                cpu: 100m
                memory: 256Mi
          volumes:
          - name: backup-scripts
            configMap:
              name: backup-scripts
              defaultMode: 0755
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-storage
          restartPolicy: OnFailure
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
```

### Backup RBAC

```yaml
# backup-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: monitoring-backup
  namespace: oran-monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: oran-monitoring
  name: monitoring-backup
rules:
- apiGroups: [""]
  resources: ["pods", "pods/exec", "configmaps", "secrets", "persistentvolumeclaims"]
  verbs: ["get", "list", "create"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list"]
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors", "prometheusrules"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: monitoring-backup
  namespace: oran-monitoring
subjects:
- kind: ServiceAccount
  name: monitoring-backup
  namespace: oran-monitoring
roleRef:
  kind: Role
  name: monitoring-backup
  apiGroup: rbac.authorization.k8s.io
```

## Restore Procedures

### Prometheus Restore

```bash
#!/bin/bash
# prometheus-restore.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_DATE="${1:-$(ls /backups/prometheus | tail -1)}"
BACKUP_DIR="/backups/prometheus/$BACKUP_DATE"

log_info "Starting Prometheus restore from $BACKUP_DIR"

# Function to restore Prometheus data
restore_prometheus_data() {
    log_info "Restoring Prometheus data..."

    # Scale down Prometheus
    kubectl scale deployment prometheus --replicas=0 -n "$NAMESPACE"

    # Wait for pod termination
    kubectl wait --for=delete pod -l app.kubernetes.io/name=prometheus -n "$NAMESPACE" --timeout=300s

    # Get Prometheus storage PVC
    local pvc_name
    pvc_name=$(kubectl get pvc -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')

    # Create temporary pod to restore data
    kubectl run prometheus-restore --rm -i --tty \
        --namespace="$NAMESPACE" \
        --image=alpine:3.18 \
        --overrides='{
          "spec": {
            "containers": [{
              "name": "prometheus-restore",
              "image": "alpine:3.18",
              "command": ["/bin/sh"],
              "stdin": true,
              "tty": true,
              "volumeMounts": [{
                "name": "prometheus-storage",
                "mountPath": "/prometheus"
              }]
            }],
            "volumes": [{
              "name": "prometheus-storage",
              "persistentVolumeClaim": {
                "claimName": "'$pvc_name'"
              }
            }]
          }
        }' << EOF
# Clear existing data
rm -rf /prometheus/*

# Install required tools
apk add --no-cache pv

# Restore data from backup
cat > /prometheus/restore.tar.gz
tar xzf /prometheus/restore.tar.gz -C /prometheus --strip-components=1
rm /prometheus/restore.tar.gz

exit
EOF

    # Pipe backup data to restore pod
    cat "$BACKUP_DIR/prometheus-data-${BACKUP_DATE}.tar.gz" | \
        kubectl exec -i prometheus-restore -n "$NAMESPACE" -- sh -c 'cat > /prometheus/restore.tar.gz && tar xzf /prometheus/restore.tar.gz -C /prometheus --strip-components=1 && rm /prometheus/restore.tar.gz'

    # Scale up Prometheus
    kubectl scale deployment prometheus --replicas=1 -n "$NAMESPACE"

    # Wait for readiness
    kubectl wait --for=condition=available deployment/prometheus -n "$NAMESPACE" --timeout=300s

    log_success "Prometheus data restore completed"
}

# Function to restore Prometheus configuration
restore_prometheus_config() {
    log_info "Restoring Prometheus configuration..."

    # Restore ConfigMaps
    kubectl apply -f "$BACKUP_DIR/prometheus-configmaps-${BACKUP_DATE}.yaml"

    # Restore Secrets (excluding auto-generated ones)
    kubectl apply -f "$BACKUP_DIR/prometheus-secrets-${BACKUP_DATE}.yaml"

    # Restore ServiceMonitors if they exist
    if [[ -f "$BACKUP_DIR/prometheus-servicemonitors-${BACKUP_DATE}.yaml" ]]; then
        kubectl apply -f "$BACKUP_DIR/prometheus-servicemonitors-${BACKUP_DATE}.yaml"
    fi

    # Restore PrometheusRules if they exist
    if [[ -f "$BACKUP_DIR/prometheus-rules-${BACKUP_DATE}.yaml" ]]; then
        kubectl apply -f "$BACKUP_DIR/prometheus-rules-${BACKUP_DATE}.yaml"
    fi

    log_success "Prometheus configuration restore completed"
}

# Execute restore
if [[ ! -d "$BACKUP_DIR" ]]; then
    log_error "Backup directory not found: $BACKUP_DIR"
    exit 1
fi

restore_prometheus_config
restore_prometheus_data

log_success "Prometheus restore completed successfully"
```

### Grafana Restore

```bash
#!/bin/bash
# grafana-restore.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_DATE="${1:-$(ls /backups/grafana | tail -1)}"
BACKUP_DIR="/backups/grafana/$BACKUP_DATE"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-}"

log_info "Starting Grafana restore from $BACKUP_DIR"

# Function to restore Grafana database
restore_grafana_database() {
    log_info "Restoring Grafana database..."

    # Scale down Grafana
    kubectl scale deployment grafana --replicas=0 -n "$NAMESPACE"
    kubectl wait --for=delete pod -l app.kubernetes.io/name=grafana -n "$NAMESPACE" --timeout=300s

    # Find database backup file
    local db_backup_file
    if [[ -f "$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql" ]]; then
        db_backup_file="$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql"
    elif [[ -f "$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql.gz" ]]; then
        db_backup_file="$BACKUP_DIR/grafana-database-${BACKUP_DATE}.sql.gz"
    else
        log_error "Database backup file not found"
        return 1
    fi

    # Scale up Grafana
    kubectl scale deployment grafana --replicas=1 -n "$NAMESPACE"
    kubectl wait --for=condition=available deployment/grafana -n "$NAMESPACE" --timeout=300s

    # Get Grafana pod
    local grafana_pod
    grafana_pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}')

    # Restore database based on type
    if [[ "$db_backup_file" == *.sql.gz ]]; then
        zcat "$db_backup_file" | kubectl exec -i -n "$NAMESPACE" "$grafana_pod" -- \
            psql grafana -U grafana
    else
        kubectl exec -i -n "$NAMESPACE" "$grafana_pod" -- \
            sqlite3 /var/lib/grafana/grafana.db < "$db_backup_file"
    fi

    log_success "Grafana database restore completed"
}

# Function to restore Grafana via API
restore_grafana_api_data() {
    log_info "Restoring Grafana via API..."

    # Get Grafana password
    if [[ -z "$GRAFANA_PASSWORD" ]]; then
        GRAFANA_PASSWORD=$(kubectl get secret -n "$NAMESPACE" grafana-admin-credentials \
            -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || echo "admin")
    fi

    # Setup port-forward
    kubectl port-forward -n "$NAMESPACE" service/grafana 3000:3000 &>/dev/null &
    local port_forward_pid=$!
    sleep 3

    # Ensure cleanup
    trap "kill $port_forward_pid 2>/dev/null || true" EXIT

    # Restore data sources
    if [[ -f "$BACKUP_DIR/datasources-${BACKUP_DATE}.json" ]]; then
        log_info "Restoring data sources..."
        jq -c '.[]' "$BACKUP_DIR/datasources-${BACKUP_DATE}.json" | while read -r datasource; do
            curl -X POST -H "Content-Type: application/json" \
                -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
                -d "$datasource" \
                "http://localhost:3000/api/datasources" || true
        done
    fi

    # Restore folders
    if [[ -f "$BACKUP_DIR/folders-${BACKUP_DATE}.json" ]]; then
        log_info "Restoring folders..."
        jq -c '.[]' "$BACKUP_DIR/folders-${BACKUP_DATE}.json" | while read -r folder; do
            curl -X POST -H "Content-Type: application/json" \
                -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
                -d "$folder" \
                "http://localhost:3000/api/folders" || true
        done
    fi

    # Restore dashboards
    log_info "Restoring dashboards..."
    for dashboard_file in "$BACKUP_DIR"/dashboard-*.json; do
        if [[ -f "$dashboard_file" ]]; then
            local dashboard_data
            dashboard_data=$(jq '.dashboard' "$dashboard_file")
            local restore_payload
            restore_payload=$(jq -n --argjson dashboard "$dashboard_data" '{dashboard: $dashboard, overwrite: true}')

            curl -X POST -H "Content-Type: application/json" \
                -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
                -d "$restore_payload" \
                "http://localhost:3000/api/dashboards/db" || true
        fi
    done

    # Restore alert rules
    if [[ -f "$BACKUP_DIR/alert-rules-${BACKUP_DATE}.json" ]]; then
        log_info "Restoring alert rules..."
        curl -X PUT -H "Content-Type: application/json" \
            -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
            -d "@$BACKUP_DIR/alert-rules-${BACKUP_DATE}.json" \
            "http://localhost:3000/api/ruler/grafana/api/v1/rules/default" || true
    fi

    kill $port_forward_pid 2>/dev/null || true
    log_success "Grafana API data restore completed"
}

# Function to restore Grafana configuration
restore_grafana_config() {
    log_info "Restoring Grafana configuration..."

    # Restore ConfigMaps
    kubectl apply -f "$BACKUP_DIR/grafana-configmaps-${BACKUP_DATE}.yaml"

    # Restore Secrets
    kubectl apply -f "$BACKUP_DIR/grafana-secrets-${BACKUP_DATE}.yaml"

    log_success "Grafana configuration restore completed"
}

# Execute restore
if [[ ! -d "$BACKUP_DIR" ]]; then
    log_error "Backup directory not found: $BACKUP_DIR"
    exit 1
fi

restore_grafana_config
restore_grafana_database
restore_grafana_api_data

log_success "Grafana restore completed successfully"
```

### AlertManager Restore

```bash
#!/bin/bash
# alertmanager-restore.sh

set -euo pipefail

NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
BACKUP_DATE="${1:-$(ls /backups/alertmanager | tail -1)}"
BACKUP_DIR="/backups/alertmanager/$BACKUP_DATE"

log_info "Starting AlertManager restore from $BACKUP_DIR"

# Function to restore AlertManager data
restore_alertmanager_data() {
    log_info "Restoring AlertManager data..."

    # Scale down AlertManager
    kubectl scale statefulset alertmanager --replicas=0 -n "$NAMESPACE"
    kubectl wait --for=delete pod -l app.kubernetes.io/name=alertmanager -n "$NAMESPACE" --timeout=300s

    # Get PVCs
    local pvcs
    pvcs=$(kubectl get pvc -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager -o jsonpath='{.items[*].metadata.name}')

    # Restore data for each instance
    for pvc in $pvcs; do
        local instance_num
        instance_num=$(echo "$pvc" | grep -o '[0-9]*$')

        log_info "Restoring data for AlertManager instance $instance_num..."

        # Create temporary pod for restore
        kubectl run alertmanager-restore-$instance_num --rm -i \
            --namespace="$NAMESPACE" \
            --image=alpine:3.18 \
            --overrides='{
              "spec": {
                "containers": [{
                  "name": "alertmanager-restore",
                  "image": "alpine:3.18",
                  "command": ["/bin/sh", "-c", "sleep 3600"],
                  "volumeMounts": [{
                    "name": "alertmanager-storage",
                    "mountPath": "/alertmanager"
                  }]
                }],
                "volumes": [{
                  "name": "alertmanager-storage",
                  "persistentVolumeClaim": {
                    "claimName": "'$pvc'"
                  }
                }]
              }
            }' &

        sleep 10

        # Restore data
        local backup_file="$BACKUP_DIR/alertmanager-data-alertmanager-${instance_num}-${BACKUP_DATE}.tar.gz"
        if [[ -f "$backup_file" ]]; then
            kubectl exec -i alertmanager-restore-$instance_num -n "$NAMESPACE" -- \
                sh -c 'rm -rf /alertmanager/* && tar xzf - -C /' < "$backup_file"
        fi

        # Cleanup restore pod
        kubectl delete pod alertmanager-restore-$instance_num -n "$NAMESPACE"
    done

    # Scale up AlertManager
    kubectl scale statefulset alertmanager --replicas=3 -n "$NAMESPACE"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=alertmanager -n "$NAMESPACE" --timeout=300s

    log_success "AlertManager data restore completed"
}

# Function to restore AlertManager configuration
restore_alertmanager_config() {
    log_info "Restoring AlertManager configuration..."

    # Restore ConfigMaps
    kubectl apply -f "$BACKUP_DIR/alertmanager-configmaps-${BACKUP_DATE}.yaml"

    # Restore Secrets
    kubectl apply -f "$BACKUP_DIR/alertmanager-secrets-${BACKUP_DATE}.yaml"

    log_success "AlertManager configuration restore completed"
}

# Execute restore
if [[ ! -d "$BACKUP_DIR" ]]; then
    log_error "Backup directory not found: $BACKUP_DIR"
    exit 1
fi

restore_alertmanager_config
restore_alertmanager_data

log_success "AlertManager restore completed successfully"
```

## Disaster Recovery

### Complete Stack Restore

```bash
#!/bin/bash
# complete-restore.sh

set -euo pipefail

BACKUP_BASE_DATE="${1:-$(ls /backups/prometheus | tail -1)}"

log_info "=== Starting Complete O-RAN Monitoring Stack Restore ==="
log_info "Using backup date: $BACKUP_BASE_DATE"

# Restore in order
log_info "Step 1: Restoring Prometheus..."
./prometheus-restore.sh "$BACKUP_BASE_DATE"

log_info "Step 2: Restoring AlertManager..."
./alertmanager-restore.sh "$BACKUP_BASE_DATE"

log_info "Step 3: Restoring Grafana..."
./grafana-restore.sh "$BACKUP_BASE_DATE"

# Verify restoration
log_info "Step 4: Verifying restoration..."
sleep 30

# Health checks
./deployment/kubernetes/health-checks/check-prometheus-targets.sh
./deployment/kubernetes/health-checks/check-grafana-dashboards.sh
./deployment/kubernetes/health-checks/check-alerts.sh

log_success "=== Complete monitoring stack restore completed ==="
```

### Backup Verification

```bash
#!/bin/bash
# verify-backup.sh

set -euo pipefail

BACKUP_DATE="${1:-$(ls /backups/prometheus | tail -1)}"

log_info "Verifying backup integrity for date: $BACKUP_DATE"

# Verify Prometheus backup
if [[ -f "/backups/prometheus/$BACKUP_DATE/prometheus-data-${BACKUP_DATE}.tar.gz" ]]; then
    log_info "Verifying Prometheus data backup..."
    if tar tzf "/backups/prometheus/$BACKUP_DATE/prometheus-data-${BACKUP_DATE}.tar.gz" >/dev/null; then
        log_success "Prometheus data backup is valid"
    else
        log_error "Prometheus data backup is corrupted"
    fi
fi

# Verify Grafana backup
if [[ -f "/backups/grafana/$BACKUP_DATE/grafana-database-${BACKUP_DATE}.sql" ]]; then
    log_info "Verifying Grafana database backup..."
    if head -1 "/backups/grafana/$BACKUP_DATE/grafana-database-${BACKUP_DATE}.sql" | grep -q "SQLite"; then
        log_success "Grafana database backup is valid"
    else
        log_error "Grafana database backup may be corrupted"
    fi
fi

# Verify AlertManager backup
if [[ -f "/backups/alertmanager/$BACKUP_DATE/alertmanager-data-alertmanager-0-${BACKUP_DATE}.tar.gz" ]]; then
    log_info "Verifying AlertManager backup..."
    if tar tzf "/backups/alertmanager/$BACKUP_DATE/alertmanager-data-alertmanager-0-${BACKUP_DATE}.tar.gz" >/dev/null; then
        log_success "AlertManager backup is valid"
    else
        log_error "AlertManager backup is corrupted"
    fi
fi

log_success "Backup verification completed"
```

This comprehensive backup and restore guide ensures robust data protection and disaster recovery capabilities for the O-RAN monitoring stack.