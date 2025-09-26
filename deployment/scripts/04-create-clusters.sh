#!/bin/bash
################################################################################
# 階段 4: 創建 Kubernetes 集群
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/04-clusters.log"; }

log "=========================================="
log "階段 4: 創建 Kubernetes 集群"
log "=========================================="

# 檢查 Docker 是否運行
if ! sudo docker ps &> /dev/null; then
    log "錯誤: Docker 未運行，嘗試啟動..."
    sudo systemctl start docker
    sleep 5
fi

# 刪除現有集群（如果存在）
if kind get clusters 2>/dev/null | grep -q "oran-mano"; then
    log "刪除現有集群..."
    kind delete cluster --name oran-mano
fi

# 創建新集群
log "創建 Kind 集群..."
kind create cluster --config "${PROJECT_ROOT}/deployment/kind/oran-cluster.yaml" --wait 300s

# 等待集群就緒
log "等待集群就緒..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# 創建命名空間
log "創建命名空間..."
kubectl create namespace oran-system || true
kubectl create namespace oran-ran || true
kubectl create namespace oran-cn || true
kubectl create namespace oran-tn || true
kubectl create namespace monitoring || true

# 顯示集群信息
log "集群信息:"
kubectl cluster-info
kubectl get nodes -o wide
kubectl get namespaces

log "Kubernetes 集群創建完成"