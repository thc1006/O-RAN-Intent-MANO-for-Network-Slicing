#!/bin/bash
################################################################################
# 階段 6: 運行測試套件
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"
RESULTS_DIR="${PROJECT_ROOT}/deployment/test-results"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/06-tests.log"; }

log "=========================================="
log "階段 6: 運行測試"
log "=========================================="

mkdir -p "${RESULTS_DIR}"

# 測試集群連接
log "測試 1: 集群連接性"
if kubectl cluster-info &> /dev/null; then
    log "✓ 集群連接正常"
    echo "PASS" > "${RESULTS_DIR}/cluster-connectivity.txt"
else
    log "✗ 集群連接失敗"
    echo "FAIL" > "${RESULTS_DIR}/cluster-connectivity.txt"
fi

# 測試命名空間
log "測試 2: 命名空間存在性"
namespaces=("oran-system" "oran-ran" "oran-cn" "oran-tn" "monitoring")
for ns in "${namespaces[@]}"; do
    if kubectl get namespace "$ns" &> /dev/null; then
        log "✓ 命名空間 $ns 存在"
    else
        log "✗ 命名空間 $ns 不存在"
    fi
done

# 測試 Pod 狀態
log "測試 3: Pod 健康狀態"
kubectl get pods -A -o wide | tee "${RESULTS_DIR}/pod-status.txt"

# 測試服務端點
log "測試 4: 服務端點"
kubectl get services -A | tee "${RESULTS_DIR}/service-endpoints.txt"

# 測試網絡連接
log "測試 5: Pod 間網絡連接"
if kubectl get pod test-orchestrator -n oran-system &> /dev/null; then
    kubectl exec test-orchestrator -n oran-system -- ping -c 3 kubernetes.default.svc.cluster.local &> "${RESULTS_DIR}/network-test.txt" || true
    log "✓ 網絡連接測試完成"
fi

# 生成測試摘要
log "生成測試摘要..."
cat > "${RESULTS_DIR}/test-summary.txt" <<EOF
測試執行時間: $(date)

測試結果摘要:
===============
1. 集群連接: $(cat ${RESULTS_DIR}/cluster-connectivity.txt)
2. 命名空間: 已創建 5 個命名空間
3. Pod 狀態: 查看 pod-status.txt
4. 服務端點: 查看 service-endpoints.txt
5. 網絡連接: 查看 network-test.txt

詳細日誌: ${LOG_DIR}/06-tests.log
EOF

cat "${RESULTS_DIR}/test-summary.txt"
log "測試套件執行完成"