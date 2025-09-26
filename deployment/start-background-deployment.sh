#!/bin/bash
################################################################################
# 啟動後台自動化部署
# 使用 tmux 確保即使斷開連接也能繼續運行
################################################################################

set -euo pipefail

PROJECT_ROOT="/home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing"
DEPLOYMENT_DIR="${PROJECT_ROOT}/deployment"
TMUX_SESSION="oran-mano-deploy"

# 顏色
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}=========================================="
echo "O-RAN Intent-MANO 後台部署啟動器"
echo -e "==========================================${NC}"

# 檢查 tmux
if ! command -v tmux &> /dev/null; then
    echo -e "${YELLOW}安裝 tmux...${NC}"
    sudo apt-get update -qq
    sudo apt-get install -y -qq tmux
fi

# 確保腳本可執行
chmod +x "${DEPLOYMENT_DIR}/automated-deployment-master.sh"
find "${DEPLOYMENT_DIR}/scripts" -name "*.sh" -exec chmod +x {} \;

# 創建或連接 tmux 會話
if tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
    echo -e "${YELLOW}警告: tmux 會話 '${TMUX_SESSION}' 已存在${NC}"
    echo "選項:"
    echo "  1) 連接到現有會話 (tmux attach -t ${TMUX_SESSION})"
    echo "  2) 殺死現有會話並重新開始 (tmux kill-session -t ${TMUX_SESSION})"
    echo ""
    read -p "選擇 (1/2): " choice

    if [ "$choice" = "2" ]; then
        tmux kill-session -t "${TMUX_SESSION}"
    else
        tmux attach -t "${TMUX_SESSION}"
        exit 0
    fi
fi

# 創建新的 tmux 會話
echo -e "${GREEN}創建 tmux 會話: ${TMUX_SESSION}${NC}"
tmux new-session -d -s "${TMUX_SESSION}" -n "main"

# 在 tmux 中運行主腳本
tmux send-keys -t "${TMUX_SESSION}:main" "cd ${PROJECT_ROOT}" C-m
tmux send-keys -t "${TMUX_SESSION}:main" "bash ${DEPLOYMENT_DIR}/automated-deployment-master.sh" C-m

# 創建監控窗格
tmux new-window -t "${TMUX_SESSION}" -n "logs"
tmux send-keys -t "${TMUX_SESSION}:logs" "cd ${PROJECT_ROOT}" C-m
tmux send-keys -t "${TMUX_SESSION}:logs" "tail -f ${DEPLOYMENT_DIR}/logs/master.log" C-m

# 創建 kubectl 監控窗格
tmux new-window -t "${TMUX_SESSION}" -n "kubectl"
tmux send-keys -t "${TMUX_SESSION}:kubectl" "watch -n 5 kubectl get pods -A" C-m

echo -e "${GREEN}=========================================="
echo "後台部署已啟動！"
echo "==========================================${NC}"
echo ""
echo "使用以下命令管理部署:"
echo ""
echo -e "${BLUE}1. 連接到會話查看進度:${NC}"
echo "   tmux attach -t ${TMUX_SESSION}"
echo ""
echo -e "${BLUE}2. 在會話中切換窗格:${NC}"
echo "   Ctrl+b n  (下一個窗格)"
echo "   Ctrl+b p  (上一個窗格)"
echo "   Ctrl+b 0  (主窗格)"
echo "   Ctrl+b 1  (日誌窗格)"
echo "   Ctrl+b 2  (kubectl 窗格)"
echo ""
echo -e "${BLUE}3. 從會話中分離 (保持運行):${NC}"
echo "   Ctrl+b d"
echo ""
echo -e "${BLUE}4. 查看實時日誌:${NC}"
echo "   tail -f ${DEPLOYMENT_DIR}/logs/master.log"
echo ""
echo -e "${BLUE}5. 檢查當前階段:${NC}"
echo "   cat ${DEPLOYMENT_DIR}/logs/current-phase.txt"
echo ""
echo -e "${BLUE}6. 查看最終報告:${NC}"
echo "   cat ${DEPLOYMENT_DIR}/test-results/FINAL_DEPLOYMENT_REPORT.md"
echo ""
echo -e "${YELLOW}提示: 您現在可以安全地斷開連接，部署將繼續在後台運行${NC}"
echo ""