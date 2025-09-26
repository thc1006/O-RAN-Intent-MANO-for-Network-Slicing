#!/bin/bash
################################################################################
# O-RAN Intent-MANO 自動化部署主控腳本
# 此腳本在 tmux 中後台運行完整的部署和測試流程
################################################################################

set -euo pipefail

# 配置
PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
DEPLOYMENT_DIR="${PROJECT_ROOT}/deployment"
LOG_DIR="${DEPLOYMENT_DIR}/logs"
RESULTS_DIR="${DEPLOYMENT_DIR}/test-results"
TMUX_SESSION="oran-mano-deploy"

# 顏色輸出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 創建必要目錄
mkdir -p "${LOG_DIR}" "${RESULTS_DIR}"

# 日誌函數
log() {
    local level=$1
    shift
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "[${timestamp}] [${level}] $*" | tee -a "${LOG_DIR}/master.log"
}

log_info() { log "INFO" "${BLUE}$*${NC}"; }
log_success() { log "SUCCESS" "${GREEN}$*${NC}"; }
log_warning() { log "WARNING" "${YELLOW}$*${NC}"; }
log_error() { log "ERROR" "${RED}$*${NC}"; }

# 階段標記函數
mark_phase_start() {
    local phase=$1
    log_info "=========================================="
    log_info "階段開始: ${phase}"
    log_info "=========================================="
    echo "${phase}" > "${LOG_DIR}/current-phase.txt"
    date '+%s' > "${LOG_DIR}/${phase}-start.timestamp"
}

mark_phase_complete() {
    local phase=$1
    local start_time=$(cat "${LOG_DIR}/${phase}-start.timestamp")
    local end_time=$(date '+%s')
    local duration=$((end_time - start_time))
    log_success "階段完成: ${phase} (耗時: ${duration}秒)"
    echo "completed" > "${LOG_DIR}/${phase}-status.txt"
    echo "${duration}" > "${LOG_DIR}/${phase}-duration.txt"
}

# 錯誤處理
trap 'log_error "腳本執行失敗，請查看日誌: ${LOG_DIR}/master.log"' ERR

################################################################################
# 主要執行流程
################################################################################

main() {
    log_info "==================================================="
    log_info "O-RAN Intent-MANO 自動化部署開始"
    log_info "==================================================="
    log_info "項目根目錄: ${PROJECT_ROOT}"
    log_info "日誌目錄: ${LOG_DIR}"
    log_info "結果目錄: ${RESULTS_DIR}"
    log_info "==================================================="

    # 記錄開始時間
    date '+%Y-%m-%d %H:%M:%S' > "${LOG_DIR}/deployment-start.txt"

    # 階段 1: 環境準備
    mark_phase_start "environment-setup"
    log_info "執行環境設置腳本..."
    bash "${DEPLOYMENT_DIR}/scripts/01-setup-environment.sh" 2>&1 | tee "${LOG_DIR}/01-environment.log"
    mark_phase_complete "environment-setup"

    # 階段 2: 依賴安裝
    mark_phase_start "dependency-installation"
    log_info "安裝系統依賴..."
    bash "${DEPLOYMENT_DIR}/scripts/02-install-dependencies.sh" 2>&1 | tee "${LOG_DIR}/02-dependencies.log"
    mark_phase_complete "dependency-installation"

    # 階段 3: 構建映像
    mark_phase_start "image-build"
    log_info "構建 Docker 映像..."
    bash "${DEPLOYMENT_DIR}/scripts/03-build-images.sh" 2>&1 | tee "${LOG_DIR}/03-images.log"
    mark_phase_complete "image-build"

    # 階段 4: 創建集群
    mark_phase_start "cluster-creation"
    log_info "創建 Kubernetes 集群..."
    bash "${DEPLOYMENT_DIR}/scripts/04-create-clusters.sh" 2>&1 | tee "${LOG_DIR}/04-clusters.log"
    mark_phase_complete "cluster-creation"

    # 階段 5: 部署組件
    mark_phase_start "component-deployment"
    log_info "部署核心組件..."
    bash "${DEPLOYMENT_DIR}/scripts/05-deploy-components.sh" 2>&1 | tee "${LOG_DIR}/05-deployment.log"
    mark_phase_complete "component-deployment"

    # 階段 6: 運行測試
    mark_phase_start "testing"
    log_info "執行測試套件..."
    bash "${DEPLOYMENT_DIR}/scripts/06-run-tests.sh" 2>&1 | tee "${LOG_DIR}/06-tests.log"
    mark_phase_complete "testing"

    # 階段 7: 性能驗證
    mark_phase_start "performance-validation"
    log_info "執行性能驗證..."
    bash "${DEPLOYMENT_DIR}/scripts/07-performance-tests.sh" 2>&1 | tee "${LOG_DIR}/07-performance.log"
    mark_phase_complete "performance-validation"

    # 階段 8: 生成報告
    mark_phase_start "report-generation"
    log_info "生成最終報告..."
    bash "${DEPLOYMENT_DIR}/scripts/08-generate-report.sh" 2>&1 | tee "${LOG_DIR}/08-report.log"
    mark_phase_complete "report-generation"

    # 記錄完成時間
    date '+%Y-%m-%d %H:%M:%S' > "${LOG_DIR}/deployment-end.txt"

    # 計算總耗時
    local start_ts=$(date -d "$(cat ${LOG_DIR}/deployment-start.txt)" '+%s')
    local end_ts=$(date -d "$(cat ${LOG_DIR}/deployment-end.txt)" '+%s')
    local total_duration=$((end_ts - start_ts))

    log_success "==================================================="
    log_success "部署和測試完成！"
    log_success "總耗時: ${total_duration}秒 ($((total_duration / 60))分鐘)"
    log_success "==================================================="
    log_success "查看完整報告: ${RESULTS_DIR}/FINAL_DEPLOYMENT_REPORT.md"
    log_success "查看主日誌: ${LOG_DIR}/master.log"
    log_success "==================================================="
}

# 執行主流程
main "$@"