#!/bin/bash
################################################################################
# 階段 2: 安裝系統依賴
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
LOG_DIR="${PROJECT_ROOT}/deployment/logs"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_DIR}/02-dependencies.log"; }

log "=========================================="
log "階段 2: 安裝依賴"
log "=========================================="

# 更新系統
log "更新系統套件列表..."
sudo apt-get update -qq

# 安裝基礎工具
log "安裝基礎工具..."
sudo apt-get install -y -qq \
    curl \
    wget \
    git \
    jq \
    ca-certificates \
    gnupg \
    lsb-release \
    apt-transport-https \
    software-properties-common

# 安裝 Docker
if ! command -v docker &> /dev/null; then
    log "安裝 Docker..."
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
    sudo sh /tmp/get-docker.sh
    sudo usermod -aG docker $USER
    log "Docker 安裝完成"
else
    log "Docker 已安裝: $(docker --version)"
fi

# 啟動 Docker
sudo systemctl start docker
sudo systemctl enable docker
log "Docker 服務已啟動"

# 安裝 Go 1.24.7
if ! command -v go &> /dev/null; then
    log "安裝 Go 1.24.7..."
    cd /tmp
    wget -q https://go.dev/dl/go1.24.7.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.24.7.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    log "Go 安裝完成: $(/usr/local/go/bin/go version)"
else
    log "Go 已安裝: $(go version)"
fi

# 安裝網絡工具
log "安裝網絡工具..."
sudo apt-get install -y -qq \
    iperf3 \
    iproute2 \
    bridge-utils \
    net-tools \
    iputils-ping

# 安裝 tmux
if ! command -v tmux &> /dev/null; then
    log "安裝 tmux..."
    sudo apt-get install -y -qq tmux
fi

log "所有依賴安裝完成"
log "已安裝工具版本:"
log "  - Docker: $(docker --version)"
log "  - Go: $(/usr/local/go/bin/go version)"
log "  - kubectl: $(kubectl version --client --short)"
log "  - kind: $(kind version)"
log "  - iperf3: $(iperf3 --version | head -1)"