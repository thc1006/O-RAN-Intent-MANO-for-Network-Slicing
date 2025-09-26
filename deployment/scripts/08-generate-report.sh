#!/bin/bash
################################################################################
# 階段 8: 生成最終報告
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"
RESULTS_DIR="${PROJECT_ROOT}/deployment/test-results"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/08-report.log"; }

log "=========================================="
log "階段 8: 生成最終報告"
log "=========================================="

# 收集所有階段的耗時
collect_timings() {
    echo "階段耗時統計:"
    echo "============="
    for phase in environment-setup dependency-installation image-build cluster-creation component-deployment testing performance-validation; do
        if [ -f "${LOG_DIR}/${phase}-duration.txt" ]; then
            duration=$(cat "${LOG_DIR}/${phase}-duration.txt")
            echo "  - ${phase}: ${duration}秒"
        fi
    done
}

# 生成 Markdown 報告
cat > "${RESULTS_DIR}/FINAL_DEPLOYMENT_REPORT.md" <<EOF
# O-RAN Intent-MANO 部署驗證報告

**生成時間**: $(date '+%Y-%m-%d %H:%M:%S')
**項目版本**: v1.0.0
**執行者**: 自動化部署系統

---

## 執行摘要

本報告記錄了 O-RAN Intent-Based MANO for Network Slicing 系統的自動化部署和驗證過程。

### 部署時間線

- **開始時間**: $(cat ${LOG_DIR}/deployment-start.txt 2>/dev/null || echo "未記錄")
- **結束時間**: $(cat ${LOG_DIR}/deployment-end.txt 2>/dev/null || echo "未記錄")
- **總耗時**: $(if [ -f ${LOG_DIR}/deployment-start.txt ] && [ -f ${LOG_DIR}/deployment-end.txt ]; then start_ts=\$(date -d "\$(cat ${LOG_DIR}/deployment-start.txt)" '+%s'); end_ts=\$(date -d "\$(cat ${LOG_DIR}/deployment-end.txt)" '+%s'); echo "\$((end_ts - start_ts))秒"; else echo "計算中..."; fi)

### 階段執行統計

$(collect_timings)

---

## 環境信息

### 系統配置
- **操作系統**: Ubuntu 22.04 LTS
- **內核版本**: $(uname -r)
- **CPU**: $(nproc) 核心
- **內存**: $(free -h | awk '/^Mem:/ {print $2}')
- **磁盤**: $(df -h / | awk 'NR==2 {print $2}')

### 已安裝工具
- **kubectl**: $(kubectl version --client --short 2>/dev/null || echo "未安裝")
- **kind**: $(kind version 2>/dev/null || echo "未安裝")
- **Docker**: $(docker --version 2>/dev/null || echo "未安裝")
- **Go**: $(go version 2>/dev/null || echo "未安裝")

---

## 部署狀態

### Kubernetes 集群

\`\`\`
$(kubectl cluster-info 2>/dev/null || echo "集群未就緒")
\`\`\`

### 節點狀態

\`\`\`
$(kubectl get nodes -o wide 2>/dev/null || echo "無節點信息")
\`\`\`

### 命名空間

\`\`\`
$(kubectl get namespaces 2>/dev/null || echo "無命名空間信息")
\`\`\`

### Pod 狀態

\`\`\`
$(kubectl get pods -A 2>/dev/null || echo "無 Pod 信息")
\`\`\`

---

## 測試結果

### 功能測試

$(cat ${RESULTS_DIR}/test-summary.txt 2>/dev/null || echo "測試結果未生成")

### 性能測試

$(cat ${RESULTS_DIR}/performance-summary.txt 2>/dev/null || echo "性能結果未生成")

---

## 問題和建議

### 識別的問題

1. **Docker/Go 未預裝**: 需要在部署前安裝
2. **映像構建**: 需要完整的構建環境
3. **實際應用部署**: 當前僅部署了測試 Pod

### 改進建議

1. **容器化**: 將所有組件完全容器化
2. **CI/CD**: 設置自動化構建管道
3. **監控**: 部署完整的監控堆疊
4. **文檔**: 完善操作手冊

---

## 下一步行動

1. ✅ 完成依賴安裝
2. ✅ 創建 Kubernetes 集群
3. ⏳ 構建應用程序映像
4. ⏳ 部署實際應用程序
5. ⏳ 執行完整的性能測試
6. ⏳ 設置監控和告警

---

## 附錄

### 日誌文件位置

- 主日誌: \`${LOG_DIR}/master.log\`
- 環境設置: \`${LOG_DIR}/01-environment.log\`
- 依賴安裝: \`${LOG_DIR}/02-dependencies.log\`
- 映像構建: \`${LOG_DIR}/03-images.log\`
- 集群創建: \`${LOG_DIR}/04-clusters.log\`
- 組件部署: \`${LOG_DIR}/05-deployment.log\`
- 測試執行: \`${LOG_DIR}/06-tests.log\`
- 性能驗證: \`${LOG_DIR}/07-performance.log\`

### 測試結果文件

- 測試摘要: \`${RESULTS_DIR}/test-summary.txt\`
- 性能摘要: \`${RESULTS_DIR}/performance-summary.txt\`
- Pod 狀態: \`${RESULTS_DIR}/pod-status.txt\`
- 服務端點: \`${RESULTS_DIR}/service-endpoints.txt\`

---

**報告結束**
EOF

log "最終報告已生成: ${RESULTS_DIR}/FINAL_DEPLOYMENT_REPORT.md"
log "所有日誌文件位於: ${LOG_DIR}/"
log "所有測試結果位於: ${RESULTS_DIR}/"