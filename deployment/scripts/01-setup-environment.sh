#!/bin/bash
################################################################################
# 階段 1: 環境設置和驗證
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/01-environment.log"; }

log "=========================================="
log "階段 1: 環境設置"
log "=========================================="

# 檢查系統信息
log "系統信息:"
uname -a
cat /etc/os-release | grep -E "^NAME=|^VERSION="

# 檢查資源
log "資源狀態:"
log "磁盤空間:"
df -h / | tail -1
log "內存:"
free -h
log "CPU:"
nproc

# 檢查網絡
log "網絡連接:"
ping -c 3 8.8.8.8 || log "警告: 外網連接可能有問題"

# 設置環境變量
export DEBIAN_FRONTEND=noninteractive
export NEEDRESTART_MODE=a

log "環境設置完成"