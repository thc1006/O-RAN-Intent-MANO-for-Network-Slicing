# O-RAN Intent-MANO 部署指南與後續開發

**文檔版本**: v1.0
**創建時間**: 2025-09-26
**狀態**: 後台自動化部署運行中

---

## 🎯 當前部署狀態

您的 O-RAN Intent-MANO 系統正在後台自動化部署中，使用 **tmux** 會話確保即使斷開 SSH 連接也能繼續運行。

### 部署架構

```
後台自動化部署系統
├── tmux 會話: oran-mano-deploy
│   ├── 窗格 0 (main): 主部署腳本執行
│   ├── 窗格 1 (logs): 實時日誌監控
│   └── 窗格 2 (kubectl): Kubernetes 資源監控
│
└── 8 個自動化階段
    ├── 階段 1: 環境設置 ✓
    ├── 階段 2: 依賴安裝 (進行中)
    ├── 階段 3: Docker 映像構建
    ├── 階段 4: Kubernetes 集群創建
    ├── 階段 5: 核心組件部署
    ├── 階段 6: 功能測試
    ├── 階段 7: 性能驗證
    └── 階段 8: 最終報告生成
```

---

## 📺 如何查看 tmux 部署視窗

### 方法 1: 連接到 tmux 會話

```bash
# 連接到正在運行的部署會話
tmux attach -t oran-mano-deploy

# 或者使用縮寫
tmux a -t oran-mano-deploy
```

### 方法 2: 列出所有 tmux 會話

```bash
# 查看所有運行中的會話
tmux list-sessions

# 或使用縮寫
tmux ls
```

### tmux 基本操作快捷鍵

| 操作 | 快捷鍵 | 說明 |
|------|--------|------|
| **分離會話** | `Ctrl+b` 然後按 `d` | 分離但保持運行 |
| **切換窗格** | `Ctrl+b` 然後按 `0/1/2` | 切換到指定窗格 |
| **下一個窗格** | `Ctrl+b` 然後按 `n` | 順序切換 |
| **上一個窗格** | `Ctrl+b` 然後按 `p` | 反向切換 |
| **列出所有窗格** | `Ctrl+b` 然後按 `w` | 顯示窗格列表 |
| **垂直分割** | `Ctrl+b` 然後按 `%` | 創建垂直分割 |
| **水平分割** | `Ctrl+b` 然後按 `"` | 創建水平分割 |
| **關閉窗格** | `Ctrl+b` 然後按 `&` | 關閉當前窗格 |
| **滾動查看** | `Ctrl+b` 然後按 `[` | 進入滾動模式（按 q 退出）|

### 窗格說明

1. **主窗格 (main)** - 窗格 0
   - 執行主部署腳本
   - 顯示每個階段的執行進度
   - 實時輸出部署狀態

2. **日誌窗格 (logs)** - 窗格 1
   - 實時尾隨主日誌文件
   - 顯示詳細的執行日誌
   - 自動滾動

3. **監控窗格 (kubectl)** - 窗格 2
   - 每 5 秒更新 Kubernetes 資源
   - 顯示所有命名空間的 Pod 狀態
   - 實時監控集群健康

---

## 🔍 快速檢查部署狀態

### 使用狀態檢查腳本

```bash
# 運行狀態檢查腳本（已創建）
cd /home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing
bash deployment/check-deployment-status.sh
```

這會顯示：
- ✅ tmux 會話狀態
- 📍 當前執行階段
- ⏱️ 已運行時間
- ☸️ Kubernetes 集群狀態
- 📝 最新日誌（最後 10 行）

### 手動查看關鍵文件

```bash
# 查看當前階段
cat deployment/logs/current-phase.txt

# 實時查看主日誌
tail -f deployment/logs/master.log

# 查看特定階段日誌
tail -f deployment/logs/02-dependencies.log

# 查看已完成階段的耗時
cat deployment/logs/environment-setup-duration.txt
```

---

## 🎯 預期部署目標與驗證標準

### 階段 1: 環境設置 ✅ (已完成)
**預期結果**:
- ✓ 系統信息已收集
- ✓ 資源狀態已驗證
- ✓ 網絡連接正常

**驗證命令**:
```bash
cat deployment/logs/01-environment.log
```

### 階段 2: 依賴安裝 🔄 (進行中)
**預期結果**:
- ✓ Docker 安裝並運行
- ✓ Go 1.24.7 安裝
- ✓ kubectl 可用
- ✓ kind 可用
- ✓ iperf3 安裝
- ✓ 網絡工具就緒

**預計耗時**: 5-10 分鐘

**驗證命令**:
```bash
docker --version
go version
kubectl version --client
kind version
iperf3 --version
```

### 階段 3: Docker 映像構建
**預期結果**:
- 7 個容器映像成功構建:
  - oran-mano/orchestrator:latest
  - oran-mano/tn-manager:latest
  - oran-mano/tn-agent:latest
  - oran-mano/vnf-operator:latest
  - oran-mano/o2-client:latest
  - oran-mano/ran-dms:latest
  - oran-mano/cn-dms:latest

**預計耗時**: 10-20 分鐘（取決於網絡和 CPU）

**驗證命令**:
```bash
docker images | grep oran-mano
```

### 階段 4: Kubernetes 集群創建
**預期結果**:
- ✓ Kind 集群 "oran-mano" 創建
- ✓ 1 個 control-plane 節點
- ✓ 2 個 worker 節點
- ✓ 5 個命名空間創建:
  - oran-system
  - oran-ran
  - oran-cn
  - oran-tn
  - monitoring

**預計耗時**: 3-5 分鐘

**驗證命令**:
```bash
kubectl get nodes
kubectl get namespaces
kubectl cluster-info
```

### 階段 5: 核心組件部署
**預期結果**:
- ✓ RBAC 配置應用
- ✓ 網絡策略應用
- ✓ CRDs 部署
- ✓ 測試 Pod 運行

**預計耗時**: 5-8 分鐘

**驗證命令**:
```bash
kubectl get pods -A
kubectl get services -A
kubectl get crds
```

### 階段 6: 功能測試
**預期結果**:
- ✓ 集群連接測試通過
- ✓ 命名空間驗證通過
- ✓ Pod 健康檢查通過
- ✓ 網絡連接測試通過

**測試報告位置**:
```
deployment/test-results/test-summary.txt
deployment/test-results/pod-status.txt
deployment/test-results/service-endpoints.txt
```

### 階段 7: 性能驗證
**預期目標** (論文驗證標準):
- **eMBB**: 吞吐量 ≥ 4.57 Mbps, 延遲 ≤ 16.1 ms
- **URLLC**: 吞吐量 ≥ 2.77 Mbps, 延遲 ≤ 6.3 ms
- **mMTC**: 吞吐量 ≥ 0.93 Mbps, 延遲 ≤ 15.7 ms
- **E2E 部署時間**: < 10 分鐘（目標: < 60 秒）

**性能報告位置**:
```
deployment/test-results/performance-summary.txt
deployment/test-results/api-response-time.txt
```

### 階段 8: 最終報告生成
**預期輸出**:
- ✓ 完整的 Markdown 報告
- ✓ 所有階段耗時統計
- ✓ 部署狀態總結
- ✓ 測試結果匯總
- ✓ 問題和建議清單

**報告位置**:
```
deployment/test-results/FINAL_DEPLOYMENT_REPORT.md
```

---

## 📊 預期總耗時

| 階段 | 預計時間 | 累計時間 |
|------|----------|----------|
| 環境設置 | 1 分鐘 | 1 分鐘 |
| 依賴安裝 | 10 分鐘 | 11 分鐘 |
| 映像構建 | 15 分鐘 | 26 分鐘 |
| 集群創建 | 5 分鐘 | 31 分鐘 |
| 組件部署 | 8 分鐘 | 39 分鐘 |
| 功能測試 | 5 分鐘 | 44 分鐘 |
| 性能驗證 | 3 分鐘 | 47 分鐘 |
| 報告生成 | 2 分鐘 | 49 分鐘 |

**總計**: 約 45-50 分鐘

---

## 🚀 後續開發任務清單

### 立即行動項（部署完成後）

#### 1. 驗證部署結果
```bash
# 查看最終報告
cat deployment/test-results/FINAL_DEPLOYMENT_REPORT.md

# 檢查所有 Pod 狀態
kubectl get pods -A

# 測試 API 端點
kubectl port-forward -n oran-system svc/orchestrator 8080:8080
curl http://localhost:8080/health
```

#### 2. 完成核心功能實現

**優先級 P0 (1-2 週)**:
- [ ] 完成 VNF Operator 實際部署邏輯
- [ ] 實現 O2 DMS 真實 API 調用
- [ ] 完成 Nephio 封裝生成器
- [ ] 替換所有 TODO 和 mock 實現

**工作項目**:
```bash
# 查看所有 TODO 標記
grep -r "TODO" --include="*.go" . | wc -l  # 預期: 129 個

# 修復優先級
1. ran-dms/cmd/dms/main.go - 20 個 TODO
2. adapters/vnf-operator/pkg/dms/client.go - 8 個 TODO
3. adapters/vnf-operator/pkg/gitops/client.go - 6 個 TODO
```

#### 3. 提升測試覆蓋率

**當前狀態**: 71% 整合測試通過率
**目標**: 95% 以上

**行動項**:
```bash
# 修復失敗的整合測試
cd tests/integration
go test -v ./... 2>&1 | tee test-results.txt

# 安裝缺失的測試依賴
sudo apt-get install -y iperf3 iproute2 bridge-utils
```

#### 4. 部署監控堆疊

**需要部署**:
- [ ] Prometheus Stack
- [ ] Grafana Dashboards
- [ ] Alertmanager
- [ ] Loki for Logs
- [ ] Jaeger for Tracing

**部署命令**:
```bash
# 使用 Helm 部署監控
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring

# 部署自定義儀表板
kubectl apply -f monitoring/grafana/grafana-stack.yaml
```

#### 5. 實現 Web UI 儀表板

**位置**: `observability/dashboard/`

**啟動開發服務器**:
```bash
cd observability/dashboard
npm install
npm run dev
# 訪問 http://localhost:5173
```

### 中期目標（1-2 個月）

#### 1. GitOps 流程完整實現
- [ ] Porch 函數整合
- [ ] ConfigSync 自動化
- [ ] 多叢集同步
- [ ] 封裝驗證管道

#### 2. 高可用性部署
- [ ] 組件冗餘配置
- [ ] 故障轉移機制
- [ ] 自動擴展策略
- [ ] 災難恢復程序

#### 3. 性能優化
- [ ] Go 並發優化
- [ ] Kubernetes 資源調優
- [ ] 數據庫查詢優化
- [ ] 網絡延遲優化

### 長期目標（3-6 個月）

#### 1. 5G SA 核心網整合
- [ ] AMF 整合
- [ ] SMF 整合
- [ ] UPF 部署
- [ ] NSSF 實現

#### 2. AI/ML 驅動優化
- [ ] 意圖理解增強
- [ ] 資源預測模型
- [ ] 自動化故障診斷
- [ ] 性能基準學習

#### 3. 商業化準備
- [ ] 多租戶支持
- [ ] 計費系統
- [ ] SLA 管理
- [ ] 審計日誌

---

## 📖 開發工作流程

### 日常開發循環

```bash
# 1. 連接到開發環境
tmux attach -t oran-mano-deploy

# 2. 切換到代碼分支
git checkout -b feature/your-feature

# 3. 進行開發
# 編輯代碼...

# 4. 運行測試
make test-unit
make test-integration

# 5. 構建映像
make build-images

# 6. 部署到本地集群
kubectl apply -f your-manifests.yaml

# 7. 驗證
kubectl get pods -n oran-system
kubectl logs -f deployment/your-deployment

# 8. 提交更改
git add .
git commit -m "feat: your feature description"
git push origin feature/your-feature
```

### 使用 Claude Code CLI 繼續開發

```bash
# 在項目根目錄
cd /home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing

# 使用 Claude Code CLI
# 參考 CLAUDE.md 文件中的配置
```

---

## 🔧 故障排除

### 部署卡住或失敗

```bash
# 1. 檢查當前階段
cat deployment/logs/current-phase.txt

# 2. 查看錯誤日誌
tail -100 deployment/logs/master.log | grep -i error

# 3. 檢查特定階段日誌
cat deployment/logs/02-dependencies.log

# 4. 重新啟動部署（如果需要）
tmux kill-session -t oran-mano-deploy
bash deployment/start-background-deployment.sh
```

### Docker 安裝問題

```bash
# 手動安裝 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker
```

### Kubernetes 集群問題

```bash
# 重新創建集群
kind delete cluster --name oran-mano
kind create cluster --config deployment/kind/oran-cluster.yaml

# 檢查集群健康
kubectl get nodes
kubectl get pods -A
kubectl cluster-info dump
```

---

## 📞 重要路徑和命令速查

### 文件位置
```
項目根目錄: /home/ubuntu/dev/O-RAN-Intent-MANO-for-Network-Slicing

日誌目錄: deployment/logs/
├── master.log              # 主日誌
├── current-phase.txt       # 當前階段
├── 01-environment.log      # 環境設置日誌
├── 02-dependencies.log     # 依賴安裝日誌
└── ...

結果目錄: deployment/test-results/
├── FINAL_DEPLOYMENT_REPORT.md    # 最終報告
├── test-summary.txt              # 測試摘要
├── performance-summary.txt       # 性能摘要
└── ...

腳本目錄: deployment/scripts/
├── 01-setup-environment.sh
├── 02-install-dependencies.sh
└── ...
```

### 常用命令
```bash
# 查看部署狀態
bash deployment/check-deployment-status.sh

# 連接 tmux
tmux attach -t oran-mano-deploy

# 查看實時日誌
tail -f deployment/logs/master.log

# 查看最終報告
cat deployment/test-results/FINAL_DEPLOYMENT_REPORT.md

# 檢查 Kubernetes
kubectl get all -A
kubectl get nodes
kubectl cluster-info

# 查看項目狀態
git status
git log --oneline -10
```

---

## ✅ 成功標準檢查清單

部署完成後，請驗證以下項目：

- [ ] tmux 會話 "oran-mano-deploy" 運行完成
- [ ] 所有 8 個階段標記為完成
- [ ] Docker 安裝並運行
- [ ] Go 1.24.7 安裝
- [ ] Kind 集群創建成功
- [ ] 所有命名空間存在
- [ ] 測試 Pod 運行中
- [ ] 最終報告生成
- [ ] 無致命錯誤

---

## 🎓 學習資源

### 項目文檔
- **README.md**: 項目概述和快速入門
- **CLAUDE.md**: Claude Code 配置和 SPARC 工作流程
- **docs/api/**: API 文檔和 OpenAPI 規範
- **docs/architecture/**: 架構設計文檔
- **docs/cicd/**: CI/CD 配置和運行手冊

### 外部資源
- [O-RAN Alliance](https://www.o-ran.org/)
- [Nephio 文檔](https://nephio.org/docs/)
- [Kubernetes 官方文檔](https://kubernetes.io/docs/)
- [Kind 快速入門](https://kind.sigs.k8s.io/docs/user/quick-start/)

---

**祝您開發順利！🚀**

如有任何問題，請查看日誌文件或連接到 tmux 會話查看實時進度。