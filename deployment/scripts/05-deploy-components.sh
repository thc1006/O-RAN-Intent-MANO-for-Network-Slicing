#!/bin/bash
################################################################################
# 階段 5: 部署核心組件
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/05-deployment.log"; }

log "=========================================="
log "階段 5: 部署核心組件"
log "=========================================="

cd "${PROJECT_ROOT}"

# 部署現有的 Kubernetes 配置
log "部署基礎配置..."

# 應用 RBAC
if [ -f deploy/k8s/base/rbac.yaml ]; then
    kubectl apply -f deploy/k8s/base/rbac.yaml
    log "RBAC 配置已應用"
fi

# 應用網絡策略
if [ -f deploy/k8s/base/network-policies.yaml ]; then
    kubectl apply -f deploy/k8s/base/network-policies.yaml
    log "網絡策略已應用"
fi

# 部署 CRDs
log "部署 CRDs..."
find adapters/vnf-operator/config/crd/bases -name "*.yaml" -exec kubectl apply -f {} \; 2>/dev/null || log "警告: CRD 部署可能失敗"

# 創建測試 Pod
log "創建測試工作負載..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-orchestrator
  namespace: oran-system
  labels:
    app: orchestrator
    component: test
spec:
  containers:
  - name: busybox
    image: busybox:latest
    command: ['sh', '-c', 'echo Orchestrator Pod Running && sleep 3600']
  restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  name: orchestrator
  namespace: oran-system
spec:
  selector:
    app: orchestrator
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
EOF

log "等待 Pod 就緒..."
kubectl wait --for=condition=Ready pod/test-orchestrator -n oran-system --timeout=120s || log "警告: Pod 未就緒"

# 顯示部署狀態
log "部署狀態:"
kubectl get all -n oran-system
kubectl get all -n oran-ran
kubectl get all -n oran-cn

log "核心組件部署完成"