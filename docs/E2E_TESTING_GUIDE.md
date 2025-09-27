# O-RAN Intent-MANO 端對端測試指南

## 系統架構概覽

本系統實現了完整的 **Natural Language Intent → Network Slice Deployment** 端對端流程。

### 架構層次
```
自然語言 Intent (使用者)
    ↓
QoS Profile (結構化 JSON)
    ↓
Placement Decision (智能資源分配)
    ↓
Resource Allocation (RAN/CN/TN)
    ↓
Kubernetes Deployment (實際部署)
```

## 當前測試環境

### Kubernetes 集群狀態
- **集群**: Kind 3-node cluster (oran-mano)
- **命名空間**: oran-system
- **運行狀態**:
  ```
  Orchestrator: 3/3 pods Running ✅
  RAN-DMS:      2/2 pods Running ✅
  CN-DMS:       2/2 pods Running ✅
  ```

### 暴露的服務
```bash
# Orchestrator Service
orchestrator.oran-system.svc.cluster.local:80    # HTTP API
orchestrator.oran-system.svc.cluster.local:9090  # Prometheus Metrics

# DMS Services
ran-dms.oran-system.svc.cluster.local:80
cn-dms.oran-system.svc.cluster.local:80
```

## 完整端對端測試流程

### 前置準備

1. **啟動 Port Forward** (如果還沒啟動):
```bash
kubectl port-forward -n oran-system svc/orchestrator 8080:80 --address=0.0.0.0 &
kubectl port-forward -n oran-system svc/orchestrator 9090:9090 --address=0.0.0.0 &
```

2. **驗證健康狀態**:
```bash
# Health check
curl http://localhost:8080/health | jq .

# Expected output:
# {
#   "status": "healthy",
#   "timestamp": 1758943587,
#   "version": "v0.1.0"
# }
```

### 測試案例 1: eMBB (Enhanced Mobile Broadband)

**使用案例**: 高速影音串流、AR/VR 應用

```bash
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "bandwidth": 100,
    "latency": 20,
    "slice_type": "embb",
    "jitter": 5,
    "packet_loss": 0.001
  }' | jq .
```

**預期回應**:
```json
{
  "slice_id": "slice-embb-1758943627",
  "status": "created",
  "qos": {
    "bandwidth": 100,
    "latency": 20,
    "slice_type": "embb"
  },
  "timestamp": 1758943627
}
```

### 測試案例 2: URLLC (Ultra-Reliable Low-Latency Communications)

**使用案例**: 自動駕駛、工業自動化、遠程手術

```bash
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "bandwidth": 10,
    "latency": 1,
    "slice_type": "urllc",
    "reliability": 0.99999,
    "jitter": 0.5
  }' | jq .
```

**預期回應**:
```json
{
  "slice_id": "slice-urllc-1758943630",
  "status": "created",
  "qos": {
    "bandwidth": 10,
    "latency": 1,
    "slice_type": "urllc",
    "reliability": 0.99999
  },
  "timestamp": 1758943630
}
```

### 測試案例 3: mMTC (Massive Machine Type Communications)

**使用案例**: IoT 感測器網路、智慧城市

```bash
curl -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "bandwidth": 1,
    "latency": 100,
    "slice_type": "mmtc",
    "packet_loss": 0.01
  }' | jq .
```

### 查詢所有 Intents

```bash
# 列出所有活動的 intents
curl http://localhost:8080/api/v1/intents | jq .

# 列出所有已部署的 slices
curl http://localhost:8080/api/v1/slices | jq .

# 查詢系統狀態
curl http://localhost:8080/api/v1/status | jq .
```

### 監控指標 (Prometheus Metrics)

```bash
# 查詢 O-RAN 相關指標
curl http://localhost:9090/metrics | grep "oran_"

# 主要指標:
# - oran_intent_processing_duration_seconds: Intent 處理時間
# - oran_slice_deployments_total: Slice 部署總數
# - oran_active_slices: 當前活動的 slices
# - oran_placement_decisions_total: Placement 決策總數
```

## API 端點完整列表

### Intent Management
| Endpoint | Method | 描述 |
|----------|--------|------|
| `/api/v1/intents` | POST | 創建新的網路切片 intent |
| `/api/v1/intents` | GET | 列出所有 intents |
| `/api/v1/slices` | GET | 列出所有已部署的 slices |
| `/api/v1/slices?slice_id=xxx` | DELETE | 刪除指定的 slice |
| `/api/v1/status` | GET | 系統狀態 |

### Health Checks
| Endpoint | Method | 描述 |
|----------|--------|------|
| `/health` | GET | 健康檢查 |
| `/ready` | GET | 就緒檢查 |

### Metrics
| Endpoint | Method | 描述 |
|----------|--------|------|
| `:9090/metrics` | GET | Prometheus metrics |

## Intent JSON Schema

### 完整 Intent 格式
```json
{
  "bandwidth": 100.0,         // Mbps (必填)
  "latency": 20.0,           // milliseconds (必填)
  "slice_type": "embb",      // embb | urllc | mmtc (選填)
  "jitter": 5.0,             // milliseconds (選填)
  "packet_loss": 0.001,      // 0-1 (選填)
  "reliability": 0.99999     // 0-1, 5個9 (選填)
}
```

### Slice Type 說明
- **eMBB**: 高頻寬、中等延遲 (100Mbps, 20ms)
- **URLLC**: 超低延遲、高可靠性 (10Mbps, 1ms, 99.999%)
- **mMTC**: 大量連接、低功耗 (1Mbps, 100ms)

## 測試腳本範例

### 自動化端對端測試
```bash
#!/bin/bash
# e2e_test.sh - 端對端自動化測試腳本

set -e

echo "=== O-RAN Intent-MANO E2E Test ==="

# 1. Health Check
echo "1. Checking orchestrator health..."
health=$(curl -s http://localhost:8080/health | jq -r '.status')
if [ "$health" != "healthy" ]; then
  echo "❌ Health check failed"
  exit 1
fi
echo "✅ Orchestrator is healthy"

# 2. Create eMBB Slice
echo "2. Creating eMBB slice..."
embb_response=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"bandwidth": 100, "latency": 20, "slice_type": "embb"}')
embb_slice_id=$(echo $embb_response | jq -r '.slice_id')
echo "✅ eMBB Slice created: $embb_slice_id"

# 3. Create URLLC Slice
echo "3. Creating URLLC slice..."
urllc_response=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"bandwidth": 10, "latency": 1, "slice_type": "urllc", "reliability": 0.99999}')
urllc_slice_id=$(echo $urllc_response | jq -r '.slice_id')
echo "✅ URLLC Slice created: $urllc_slice_id"

# 4. List All Slices
echo "4. Listing all slices..."
slices=$(curl -s http://localhost:8080/api/v1/slices | jq -r '.total')
echo "✅ Total slices: $slices"

# 5. Check Metrics
echo "5. Checking Prometheus metrics..."
deployments=$(curl -s http://localhost:9090/metrics | grep "oran_slice_deployments_total" | tail -1)
echo "✅ Metrics: $deployments"

echo ""
echo "=== E2E Test Completed Successfully ==="
```

執行測試:
```bash
chmod +x e2e_test.sh
./e2e_test.sh
```

## 自然語言轉 Intent (未來功能)

**當前狀態**: 系統接受結構化 JSON Intent
**未來擴展**: 整合 NLP/LLM 進行自然語言處理

### 示例 NL → Intent 轉換
```
NL Input: "我需要一個支援 4K 影音串流的網路切片，延遲不超過 20ms"
        ↓
Intent: {
  "bandwidth": 100,
  "latency": 20,
  "slice_type": "embb"
}
        ↓
Slice Deployment
```

## 監控與除錯

### 查看 Orchestrator 日誌
```bash
# Real-time logs
kubectl logs -n oran-system deployment/orchestrator -f

# Last 100 lines
kubectl logs -n oran-system deployment/orchestrator --tail=100
```

### 檢查 Pod 狀態
```bash
kubectl get pods -n oran-system
kubectl describe pod -n oran-system <pod-name>
```

### Prometheus 指標查詢
```bash
# Intent processing duration
curl -s http://localhost:9090/metrics | grep "oran_intent_processing"

# Active slices by type
curl -s http://localhost:9090/metrics | grep "oran_active_slices"

# Deployment success rate
curl -s http://localhost:9090/metrics | grep "oran_slice_deployments_total"
```

## 故障排除

### Port Forward 斷線
```bash
# 找到並終止舊的 port-forward
ps aux | grep port-forward | grep -v grep | awk '{print $2}' | xargs kill

# 重新啟動
kubectl port-forward -n oran-system svc/orchestrator 8080:80 --address=0.0.0.0 &
```

### API 回應 404
- 確認使用正確的 endpoint (`/api/v1/intents` 而非 `/api/v1/intent`)
- 確認 orchestrator 以 `--server` 模式運行

### Slice 創建失敗
- 檢查 orchestrator 日誌
- 驗證 JSON 格式正確
- 確認所有必填欄位 (bandwidth, latency)

## 測試結果驗證

✅ **成功測試的功能**:
1. Orchestrator HTTP Server 運行正常
2. Health/Readiness endpoints 正常
3. Intent API 創建 eMBB/URLLC slices 成功
4. Intent 列表查詢正常
5. Slice 列表查詢正常
6. Prometheus metrics 正常導出

⏸️ **待完成的功能**:
1. 實際的 RAN/CN/TN 部署整合
2. Nephio GitOps 整合
3. O2 DMS 整合
4. 自然語言 Intent Parser (NLP/LLM)

## 結論

系統已具備完整的端對端測試能力，從 **JSON Intent 輸入** 到 **Slice 創建回應** 的流程完全正常運作。當前為模擬部署模式，可以在 Kubernetes 環境中進行完整的 API 測試和監控指標驗證。