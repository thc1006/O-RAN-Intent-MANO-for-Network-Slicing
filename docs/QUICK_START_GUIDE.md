# O-RAN 自然語言網路切片快速入門指南

## 🎯 目標

本指南將教您如何使用**自然語言**部署網路切片，並**實時觀察**從 intent → NLP 解析 → QoS 映射 → Argo CD → Kubernetes 的完整流程。

## 📋 前提條件

確保以下服務正在運行：

```bash
# 檢查服務狀態
curl http://localhost:8082/health  # NLP Service
curl http://localhost:8080/health  # Orchestrator

# 如果未運行，請啟動：
# 終端 1: NLP Service
cd nlp && python nlp_service.py

# 終端 2: Orchestrator
cd orchestrator && go run cmd/orchestrator/main.go --server

# 終端 3 (可選): WebSocket Server for Web UI
python websocket_server.py
```

---

## 🚀 方法一：命令行方式（推薦用於學習流程）

### 步驟 1: 發送自然語言請求

```bash
# 範例 1: 部署 eMBB 高帶寬切片
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy high-bandwidth video streaming for 100 users",
    "session_id": "demo-001"
  }' | jq .
```

### 步驟 2: 觀察響應（完整轉換過程）

**輸出範例：**
```json
{
  "success": true,
  "slice_id": "slice-embb-1759252082",
  "intent": {
    "raw_text": "Deploy high-bandwidth video streaming for 100 users",
    "parsed_as": "eMBB",
    "confidence": 0.426
  },
  "qos_profile": {
    "slice_type": "eMBB",
    "throughput_mbps": 50,
    "latency_ms": 10,
    "packet_loss": 0.001,
    "priority": 5
  },
  "deployment": {
    "namespace": "oran-slice-embb",
    "status": "success"
  },
  "processing_time_ms": 389
}
```

### 📊 流程分析

從上面的輸出可以看到完整的轉換過程：

```
1️⃣ 自然語言輸入
   "Deploy high-bandwidth video streaming for 100 users"

   ↓ [NLP Service 解析 - 約 300ms]

2️⃣ NLP 解析結果
   - Slice Type: eMBB (Enhanced Mobile Broadband)
   - Confidence: 0.426

   ↓ [QoS 映射 - 約 5ms]

3️⃣ QoS 參數映射
   - Throughput: 50 Mbps (高帶寬)
   - Latency: 10 ms (視頻串流可接受)
   - Packet Loss: 0.001 (99.9% 可靠性)
   - Priority: 5 (中等優先級)

   ↓ [Manifest 生成 - 約 10ms]

4️⃣ Kubernetes Manifest 生成
   - Namespace: oran-slice-embb
   - Resources: ConfigMap, Deployment, Service

   ↓ [Argo CD 部署 - 約 64ms]

5️⃣ GitOps 部署
   - Argo CD Application 創建
   - Slice ID: slice-embb-1759252082

   ✅ 總處理時間: 389ms
```

---

## 🌐 方法二：Web UI 方式（推薦用於日常使用）

### 步驟 1: 啟動 Web 服務

```bash
# 確保 WebSocket 服務器正在運行
python websocket_server.py

# 或使用整合服務器（包含 REST + WebSocket）
python integrated_server.py
```

### 步驟 2: 打開瀏覽器

```bash
# 方式 A: 使用簡單的 Python HTTP 服務器
cd web
python -m http.server 8000

# 然後訪問：
# http://localhost:8000/index.html
```

或者直接使用整合服務器（已包含 Web UI）：
```
http://localhost:8080
```

### 步驟 3: 使用 Web 界面

1. **連接狀態**：左上角會顯示 "Connected" 綠點
2. **輸入自然語言**：在底部輸入框中輸入，例如：
   ```
   Deploy high-bandwidth video streaming for 100 users
   ```
3. **查看實時處理**：
   - 聊天窗口顯示處理步驟
   - 右側面板顯示切片參數
   - 系統消息顯示每個階段狀態

### 📱 Web UI 功能

**主要面板：**
- 💬 聊天歷史記錄
- ⏳ 實時處理指示器
- ✅ 成功/錯誤通知

**側邊面板：**
- 📊 切片類型和 QoS 參數
- 🎯 當前活動切片列表
- 💡 範例請求按鈕

---

## 🔍 方法三：觀察完整部署流程

### 監控選項 1: 查看 Orchestrator 日誌

```bash
# 實時查看處理日誌
tail -f orchestrator/orchestrator.log

# 您將看到：
# 2025/10/01 01:08:02 Processing natural language intent: Deploy high-bandwidth video streaming...
# 2025/10/01 01:08:02 ✓ NLP parsed as eMBB (50.00 Mbps, 10.0 ms latency)
# 2025/10/01 01:08:02 ✓ Argo CD Application created: slice-embb-1759252082
```

### 監控選項 2: Argo CD Web UI

```bash
# 1. 獲取 Argo CD 密碼
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 -d

# 2. 端口轉發
kubectl port-forward svc/argocd-server -n argocd 8443:443

# 3. 訪問 https://localhost:8443
# 用戶名: admin
# 密碼: [從步驟 1 獲取]
```

**在 Argo CD UI 中觀察：**
1. 新的 Application 出現（名稱如 `slice-embb-1759252082`）
2. 同步狀態：OutOfSync → Syncing → Synced
3. 健康狀態：Progressing → Healthy
4. 部署的資源樹狀圖

### 監控選項 3: Kubernetes 資源

```bash
# 查看創建的命名空間
kubectl get namespaces | grep oran-slice

# 查看切片資源
kubectl get all -n oran-slice-embb

# 查看 Argo CD Applications
kubectl get applications -n argocd

# 實時監控 pods
kubectl get pods -n oran-slice-embb -w
```

### 監控選項 4: Prometheus Metrics

```bash
# 端口轉發到 Orchestrator metrics 端點
# （Orchestrator 運行在 :8080，metrics 在 :9090）
curl http://localhost:9090/metrics | grep intent

# 查看指標：
# intent_processing_duration_seconds
# intent_processing_total
# nlp_request_duration_seconds
```

---

## 📝 完整範例演示

### 場景 1: 4K 視頻串流（eMBB）

```bash
# 1. 發送自然語言請求
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy 4K video streaming slice for sports broadcasting",
    "session_id": "demo-embb"
  }' | jq .

# 2. 同時在另一個終端監控日誌
tail -f orchestrator/orchestrator.log

# 3. 查看 NLP 服務處理
curl http://localhost:8082/api/v1/history | jq .

# 4. 驗證 Kubernetes 部署
kubectl get all -n oran-slice-embb
```

**預期輸出流程：**
```
📥 接收 Intent: "Deploy 4K video streaming slice..."
   ↓
🧠 NLP 解析: 識別為 eMBB 類型
   ↓
⚙️ QoS 映射:
   - Throughput: 100 Mbps (4K 需求)
   - Latency: 10 ms
   - Reliability: 99.9%
   ↓
📦 生成 Manifest:
   - Namespace: oran-slice-embb
   - ConfigMap: slice-config
   - Deployment: slice-vnf
   ↓
🚀 Argo CD 部署:
   - Application: slice-embb-[timestamp]
   - Status: Synced
   ↓
✅ Kubernetes 資源創建完成
```

### 場景 2: 自動駕駛車輛（URLLC）

```bash
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy ultra-low latency slice for autonomous vehicles",
    "session_id": "demo-urllc"
  }' | jq .
```

**觀察重點差異：**
```json
{
  "qos_profile": {
    "slice_type": "URLLC",
    "throughput_mbps": 100,
    "latency_ms": 1,        // 極低延遲！
    "packet_loss": 0.00001,  // 極高可靠性！
    "priority": 10           // 最高優先級！
  }
}
```

### 場景 3: 智慧城市 IoT（mMTC）

```bash
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy IoT sensor network for smart city monitoring",
    "session_id": "demo-mmtc"
  }' | jq .
```

**觀察重點差異：**
```json
{
  "qos_profile": {
    "slice_type": "mMTC",
    "throughput_mbps": 1,     // 低帶寬
    "latency_ms": 16.1,       // 可接受延遲
    "packet_loss": 0.01,      // 標準可靠性
    "priority": 3             // 低優先級
  }
}
```

---

## 🎨 支援的自然語言模式

### eMBB（高帶寬）關鍵詞
```
✅ "video streaming"
✅ "4K/8K"
✅ "high bandwidth"
✅ "broadband"
✅ "multimedia"
✅ "AR/VR"
```

### URLLC（低延遲）關鍵詞
```
✅ "ultra-low latency"
✅ "low latency"
✅ "real-time"
✅ "autonomous vehicle"
✅ "industrial automation"
✅ "remote surgery"
```

### mMTC（大規模 IoT）關鍵詞
```
✅ "IoT"
✅ "sensor"
✅ "smart city"
✅ "massive"
✅ "monitoring"
✅ "metering"
```

---

## 🛠️ 進階：端到端追蹤

### 使用 Session ID 追蹤完整流程

```bash
# 1. 創建帶有 session ID 的請求
SESSION_ID="trace-$(date +%s)"

curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d "{
    \"intent\": \"Deploy video streaming for 100 users\",
    \"session_id\": \"$SESSION_ID\"
  }" | jq .

# 2. 在 NLP 服務歷史中查找該 session
curl http://localhost:8082/api/v1/history | jq ".[] | select(.session_id == \"$SESSION_ID\")"

# 3. 在 orchestrator 日誌中 grep
grep "$SESSION_ID" orchestrator/orchestrator.log

# 4. 在 Argo CD 中查找對應的 Application
kubectl get applications -n argocd | grep "$SESSION_ID"
```

---

## 📊 性能基準

基於最新的性能測試結果：

| 階段 | 平均耗時 | 說明 |
|------|----------|------|
| HTTP 請求 | ~10ms | 網路延遲 |
| NLP 解析 | ~300ms | FastAPI + 正則匹配 |
| QoS 映射 | ~5ms | 參數計算 |
| Manifest 生成 | ~10ms | YAML 模板 |
| Argo CD API | ~64ms | Kubernetes API 調用 |
| **總計** | **~389ms** | **端到端延遲** |

---

## 🐛 故障排除

### 問題 1: NLP 服務無響應

```bash
# 檢查服務狀態
curl http://localhost:8082/health

# 查看日誌
cd nlp && cat nlp_service.log

# 重啟服務
pkill -f nlp_service.py
python nlp_service.py
```

### 問題 2: Orchestrator 無法連接 NLP

```bash
# 檢查環境變量
echo $NLP_SERVICE_URL

# 如果為空，設置：
export NLP_SERVICE_URL="http://localhost:8082"

# 重啟 orchestrator
cd orchestrator && go run cmd/orchestrator/main.go --server
```

### 問題 3: Argo CD 部署失敗

```bash
# 檢查 Argo CD 是否運行
kubectl get pods -n argocd

# 查看 Application 狀態
kubectl describe application slice-embb-xxx -n argocd

# 查看同步錯誤
kubectl logs -n argocd deployment/argocd-application-controller
```

---

## 🎯 下一步

現在您已經知道如何：
1. ✅ 使用自然語言部署網路切片
2. ✅ 觀察 NLP → QoS → Argo CD → K8s 的完整流程
3. ✅ 監控部署狀態和性能指標
4. ✅ 追蹤端到端請求

**建議練習：**
1. 嘗試不同的自然語言表達方式
2. 觀察不同切片類型的 QoS 參數差異
3. 使用 Web UI 體驗實時互動
4. 查看 Argo CD 中的資源樹狀圖
5. 運行性能基準測試（`tests/performance/benchmark_nlp_e2e.sh`）

**生產部署：**
- 參考 `docs/COMPLETE_DEPLOYMENT_GUIDE.md`
- 查看 `deploy/k8s/` 中的 Kubernetes 配置
- 閱讀 `docs/PERFORMANCE_ANALYSIS.md` 了解優化建議

---

**文檔版本：** v1.0
**最後更新：** 2025-10-01
**狀態：** ✅ 已驗證（450 請求，100% 成功率）
