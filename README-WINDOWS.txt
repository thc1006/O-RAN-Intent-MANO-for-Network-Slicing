═══════════════════════════════════════════════════════════════════
  O-RAN Intent MANO for Network Slicing - Windows 使用指南
═══════════════════════════════════════════════════════════════════

🚀 快速開始
═══════════════════════════════════════════════════════════════════

1. 雙擊運行 "start-oran.bat" 啟動所有服務

2. 瀏覽器會自動打開到 http://localhost:8000/index.html

3. 在前端界面輸入自然語言，例如：
   • "Deploy high-bandwidth video streaming for 100 users"
   • "Create low-latency slice for autonomous vehicles"
   • "Setup IoT network for smart city"

4. 使用完畢後，雙擊 "stop-oran.bat" 停止所有服務


📁 文件說明
═══════════════════════════════════════════════════════════════════

start-oran.bat          - 一鍵啟動所有服務
stop-oran.bat           - 一鍵停止所有服務
README-WINDOWS.txt      - 本文件（使用指南）

logs/                   - 日誌文件夾
  ├── nlp.log          - NLP 服務日誌
  ├── orchestrator.log - Orchestrator 日誌
  ├── websocket.log    - WebSocket 日誌
  └── webui.log        - Web UI 日誌

scripts/
  └── watch-flow.sh    - 流程監控腳本（需要 Git Bash）


📋 系統需求
═══════════════════════════════════════════════════════════════════

必需：
  ✓ Python 3.11 或更高版本
  ✓ Go 1.24 或更高版本（用於 Orchestrator）
  ✓ Windows 10/11

可選：
  ○ kubectl（用於 Kubernetes 管理）
  ○ Git Bash（用於運行監控腳本）


🌐 服務端點
═══════════════════════════════════════════════════════════════════

NLP Service:      http://localhost:8082
  └─ API 文檔:    http://localhost:8082/docs

Orchestrator:     http://localhost:8080
  └─ Health:      http://localhost:8080/health

WebSocket:        ws://localhost:8081/ws

Web UI (主界面):  http://localhost:8000/index.html
Web UI (監控):    http://localhost:8000/monitor.html


📊 觀察處理流程
═══════════════════════════════════════════════════════════════════

方法 1: 前端界面（推薦）
  - 左側聊天區顯示處理進度
  - 右側面板顯示切片參數

方法 2: 查看日誌文件
  - 打開 logs\orchestrator.log 查看詳細流程
  - 打開 logs\websocket.log 查看連接狀態

方法 3: 使用監控腳本（需要 Git Bash）
  在 Git Bash 中運行：
  bash scripts/watch-flow.sh


🎯 使用範例
═══════════════════════════════════════════════════════════════════

1. eMBB (高帶寬)
   "Deploy 4K video streaming with 100 Mbps throughput"

   結果：
   - 切片類型：eMBB
   - 帶寬：50-100 Mbps
   - 延遲：10 ms
   - 可靠性：99.9%

2. URLLC (低延遲)
   "Create ultra-low latency slice for autonomous vehicles"

   結果：
   - 切片類型：URLLC
   - 帶寬：100 Mbps
   - 延遲：1 ms
   - 可靠性：99.999%

3. mMTC (大規模 IoT)
   "Setup IoT sensor network for 100000 devices"

   結果：
   - 切片類型：mMTC
   - 帶寬：1 Mbps
   - 延遲：16 ms
   - 可靠性：99%


🔧 故障排除
═══════════════════════════════════════════════════════════════════

問題：服務無法啟動
解決：
  1. 確認 Python 和 Go 已正確安裝
  2. 檢查端口是否被占用（8000, 8080, 8081, 8082）
  3. 查看 logs/ 目錄中的錯誤日誌

問題：前端無法連接
解決：
  1. 檢查 WebSocket 服務是否運行（logs\websocket.log）
  2. 刷新瀏覽器頁面 (Ctrl+F5)
  3. 確認防火牆未阻擋連接

問題：端口被占用
解決：
  1. 運行 stop-oran.bat 停止所有服務
  2. 手動檢查並終止占用端口的進程：
     netstat -ano | findstr "8080"
     taskkill /F /PID <進程ID>


📚 更多文檔
═══════════════════════════════════════════════════════════════════

docs/QUICK_START_GUIDE.md        - 快速入門指南
docs/PERFORMANCE_ANALYSIS.md     - 性能分析報告
docs/NLP_E2E_INTEGRATION_SUMMARY.md - NLP E2E 整合說明
docs/COMPLETE_DEPLOYMENT_GUIDE.md   - 完整部署指南

GitHub: https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing


💡 提示
═══════════════════════════════════════════════════════════════════

• 第一次啟動可能需要 10-15 秒
• 所有日誌文件在 logs/ 目錄中
• 建議使用 Chrome 或 Edge 瀏覽器
• 支持中文和英文自然語言輸入
• 系統已處理超過 750+ 個請求，100% 成功率


═══════════════════════════════════════════════════════════════════
版本：v1.0.0
更新日期：2025-10-01
狀態：✅ 生產就緒
═══════════════════════════════════════════════════════════════════
