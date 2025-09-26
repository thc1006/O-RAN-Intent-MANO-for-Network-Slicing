#!/bin/bash
################################################################################
# 階段 3: 構建 Docker 映像
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/03-images.log"; }

log "=========================================="
log "階段 3: 構建 Docker 映像"
log "=========================================="

cd "${PROJECT_ROOT}"

# 由於沒有 Docker 和 Go，我們先跳過映像構建
# 只創建佔位符和配置檔案

log "準備部署配置..."

# 創建 Kind 集群配置
mkdir -p deployment/kind

cat > deployment/kind/oran-cluster.yaml <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: oran-mano
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "node-role.kubernetes.io/control-plane="
  extraPortMappings:
  - containerPort: 30000
    hostPort: 30000
    protocol: TCP
  - containerPort: 30001
    hostPort: 30001
    protocol: TCP
- role: worker
  labels:
    node-type: edge
- role: worker
  labels:
    node-type: regional
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
EOF

log "Kind 集群配置已創建"
log "由於環境限制，跳過映像構建步驟"
log "將在下一階段直接使用 kubectl 部署 YAML manifests"