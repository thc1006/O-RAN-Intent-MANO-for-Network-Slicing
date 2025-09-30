#!/bin/bash
#
# 實時觀察自然語言 → Kubernetes 完整流程
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     🔍 實時流程追蹤：自然語言 → Kubernetes 部署                  ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to display section
section() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Get session ID from WebSocket log
get_latest_session() {
    tail -20 "$PROJECT_ROOT/websocket.log" | grep "New client connected" | tail -1 | sed 's/.*session-/session-/'
}

section "1️⃣ WebSocket 連接狀態"
echo "最近的連接："
tail -5 "$PROJECT_ROOT/websocket.log" | grep -E "(connection open|New client connected)" || echo "無連接記錄"

section "2️⃣ 最近處理的請求（WebSocket）"
echo "最後 3 個請求："
tail -50 "$PROJECT_ROOT/websocket.log" | grep -E "(Received intent|Processing intent|Intent processed)" | tail -9

section "3️⃣ Orchestrator 處理流程"
echo "最近處理的請求和結果："
tail -30 "$PROJECT_ROOT/orchestrator/orchestrator.log" | grep -E "(Processing natural|NLP parsed|Argo CD Application created)" | tail -12

section "4️⃣ NLP 服務統計"
curl -s http://localhost:8082/health | jq -r '"狀態: \(.status)\n已處理: \(.total_intents_processed) 個請求\n運行時間: \(.uptime_seconds) 秒"'

section "5️⃣ 最近創建的 Slice ID"
echo "從 Orchestrator 日誌提取："
tail -20 "$PROJECT_ROOT/orchestrator/orchestrator.log" | grep "Argo CD Application created" | tail -5 | sed 's/.*created: /  ✓ /'

section "6️⃣ Kubernetes 資源狀態"
echo "Argo CD Applications (最近 5 個)："
if command -v kubectl &> /dev/null; then
    kubectl get applications -n argocd 2>/dev/null | tail -6 || echo "  ⚠️ Kubernetes 未配置或 Argo CD 未運行"
else
    echo "  ⚠️ kubectl 命令不可用"
fi

echo ""
section "7️⃣ 實時監控命令"
cat << 'COMMANDS'
開啟新終端並運行以下命令進行實時監控：

# 終端 1: WebSocket 實時日誌
tail -f websocket.log

# 終端 2: Orchestrator 實時日誌
tail -f orchestrator/orchestrator.log

# 終端 3: Kubernetes 實時監控（每 2 秒刷新）
watch -n 2 'kubectl get applications -n argocd | tail -10'

# 終端 4: 特定 namespace 的資源
kubectl get all -n oran-slice-embb -w
kubectl get all -n oran-slice-urllc -w
kubectl get all -n oran-slice-mmtc -w
COMMANDS

echo ""
section "📊 完整流程時間線示意圖"
cat << 'TIMELINE'
  0ms    用戶在前端輸入自然語言
   ↓
  10ms   WebSocket 接收並轉發到 Orchestrator
   ↓
  20ms   Orchestrator 調用 NLP Service
   ↓
  320ms  NLP Service 解析並返回切片類型和 QoS 參數
   ↓
  325ms  Orchestrator 生成 Kubernetes Manifest
   ↓
  335ms  Orchestrator 調用 Argo CD API 創建 Application
   ↓
  400ms  Argo CD 同步並部署到 Kubernetes
   ↓
  500ms  Kubernetes 創建 Namespace、ConfigMap、Deployment 等資源
   ↓
  ✅    前端顯示成功消息和切片參數
TIMELINE

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  提示：在前端發送一個新的請求，然後重新運行此腳本查看結果    ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""
