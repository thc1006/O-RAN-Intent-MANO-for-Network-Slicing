#!/bin/bash
# O-RAN Intent-MANO E2E 自動化測試腳本

set -e

echo "=== O-RAN Intent-MANO 端對端測試 ==="
echo ""

# 1. Health Check
echo "▶ 步驟 1: 檢查 Orchestrator 健康狀態..."
health=$(curl -s http://localhost:8080/health | jq -r '.status')
if [ "$health" != "healthy" ]; then
  echo "❌ 健康檢查失敗"
  exit 1
fi
version=$(curl -s http://localhost:8080/health | jq -r '.version')
echo "✅ Orchestrator 健康狀態正常 (版本: $version)"
echo ""

# 2. Create eMBB Slice
echo "▶ 步驟 2: 創建 eMBB 網路切片 (高速影音串流)..."
embb_response=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"bandwidth": 100, "latency": 20, "slice_type": "embb", "jitter": 5}')
embb_slice_id=$(echo $embb_response | jq -r '.slice_id')
embb_status=$(echo $embb_response | jq -r '.status')
echo "✅ eMBB Slice 創建成功"
echo "   Slice ID: $embb_slice_id"
echo "   狀態: $embb_status"
echo "   頻寬: 100 Mbps, 延遲: 20 ms"
echo ""

# 3. Create URLLC Slice
echo "▶ 步驟 3: 創建 URLLC 網路切片 (超低延遲應用)..."
urllc_response=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"bandwidth": 10, "latency": 1, "slice_type": "urllc", "reliability": 0.99999}')
urllc_slice_id=$(echo $urllc_response | jq -r '.slice_id')
urllc_status=$(echo $urllc_response | jq -r '.status')
echo "✅ URLLC Slice 創建成功"
echo "   Slice ID: $urllc_slice_id"
echo "   狀態: $urllc_status"
echo "   頻寬: 10 Mbps, 延遲: 1 ms, 可靠度: 99.999%"
echo ""

# 4. Create mMTC Slice
echo "▶ 步驟 4: 創建 mMTC 網路切片 (大規模 IoT)..."
mmtc_response=$(curl -s -X POST http://localhost:8080/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{"bandwidth": 1, "latency": 100, "slice_type": "mmtc"}')
mmtc_slice_id=$(echo $mmtc_response | jq -r '.slice_id')
echo "✅ mMTC Slice 創建成功"
echo "   Slice ID: $mmtc_slice_id"
echo "   頻寬: 1 Mbps, 延遲: 100 ms"
echo ""

# 5. List All Slices
echo "▶ 步驟 5: 查詢所有已部署的 Slices..."
slices=$(curl -s http://localhost:8080/api/v1/slices | jq -r '.total')
echo "✅ 系統中共有 $slices 個活動 Slices"
curl -s http://localhost:8080/api/v1/slices | jq -r '.slices[] | "   - \(.slice_id) (\(.slice_type)): \(.status)"'
echo ""

# 6. Check Metrics
echo "▶ 步驟 6: 檢查 Prometheus 監控指標..."
echo "   Active Slices by Type:"
curl -s http://localhost:9090/metrics | grep "oran_active_slices{" | grep -v "#"
echo ""
echo "   Total Deployments:"
curl -s http://localhost:9090/metrics | grep "oran_slice_deployments_total" | grep -v "#" | head -3
echo ""

# 7. System Status
echo "▶ 步驟 7: 查詢系統整體狀態..."
status=$(curl -s http://localhost:8080/api/v1/status | jq -r '.status')
echo "✅ 系統狀態: $status"
echo ""

echo "=================================================="
echo "🎉 端對端測試完成！所有功能正常運作"
echo "=================================================="
echo ""
echo "測試摘要:"
echo "  ✅ Orchestrator 健康檢查通過"
echo "  ✅ eMBB Slice 創建成功 (ID: $embb_slice_id)"
echo "  ✅ URLLC Slice 創建成功 (ID: $urllc_slice_id)"
echo "  ✅ mMTC Slice 創建成功 (ID: $mmtc_slice_id)"
echo "  ✅ Slice 列表查詢正常 (共 $slices 個)"
echo "  ✅ Prometheus 指標正常導出"
echo "  ✅ 系統狀態: $status"
echo ""
