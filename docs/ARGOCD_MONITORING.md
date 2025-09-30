# Argo CD 推拉過程監控指南

## 📋 目錄
1. [Argo CD UI 訪問](#argo-cd-ui-訪問)
2. [查看推拉過程的方法](#查看推拉過程的方法)
3. [創建測試應用](#創建測試應用)
4. [實時監控命令](#實時監控命令)
5. [Troubleshooting](#troubleshooting)

---

## 🌐 Argo CD UI 訪問

### 登入資訊
- **URL**: https://localhost:8082
- **用戶名**: `admin`
- **密碼**: `bTJTYSfHmLBvN-NY`

> ⚠️ **注意**: 瀏覽器會警告 SSL 證書不受信任，點擊「進階」→「繼續前往」即可

### UI 啟動狀態
```bash
# Port-forward 已在背景運行
# 如需停止：
ps aux | grep "port-forward" | grep argocd
kill <PID>

# 如需重新啟動：
kubectl port-forward svc/argocd-server -n argocd 8082:443
```

---

## 👁️ 查看推拉過程的方法

### 方法 1: Argo CD UI（推薦）

登入後您可以看到：

1. **Applications 列表**
   - 顯示所有已部署的應用
   - 實時同步狀態
   - Health status

2. **應用詳情頁面**
   - Git commit 資訊
   - 同步歷史記錄
   - 資源拓撲圖
   - Event 日誌

3. **實時推拉過程**
   - 點擊任何應用
   - 查看 "Sync Status" 區塊
   - 顯示：
     - 📥 Pull: 從 Git 拉取最新配置
     - ⚙️ Diff: 比較當前狀態與目標狀態
     - 📤 Push: 應用變更到 Kubernetes

### 方法 2: CLI 命令

```bash
# 查看所有應用
kubectl get applications -n argocd

# 查看特定應用詳情
kubectl describe application <app-name> -n argocd

# 實時監控應用同步
kubectl get application <app-name> -n argocd -w

# 查看應用日誌
kubectl logs -n argocd deployment/argocd-application-controller -f
```

### 方法 3: Repo Server 日誌（查看 Git 拉取）

```bash
# 查看 repo-server 日誌（處理 Git 拉取）
kubectl logs -n argocd deployment/argocd-repo-server -f

# 過濾 Git 相關操作
kubectl logs -n argocd deployment/argocd-repo-server -f | grep -i "git"
```

### 方法 4: Application Controller 日誌（查看同步過程）

```bash
# 查看同步和部署過程
kubectl logs -n argocd statefulset/argocd-application-controller -f

# 過濾特定應用
kubectl logs -n argocd statefulset/argocd-application-controller -f | grep -i "slice"
```

---

## 🚀 創建測試應用來演示推拉過程

### 步驟 1: 準備 Git Repository

創建一個簡單的 Kubernetes manifest：

```yaml
# test-app.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test-slice
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: test-slice
data:
  slice_type: "eMBB"
  throughput: "1000"
```

推送到 Git：
```bash
git add test-app.yaml
git commit -m "Add test slice configuration"
git push
```

### 步驟 2: 創建 Argo CD Application

```bash
# 方式 A: 使用 YAML
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: test-slice-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/YOUR_USERNAME/O-RAN-Intent-MANO-for-Network-Slicing
    targetRevision: main
    path: deployments/test
  destination:
    server: https://kubernetes.default.svc
    namespace: test-slice
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
EOF
```

```bash
# 方式 B: 使用 Argo CD CLI
argocd app create test-slice-app \
  --repo https://github.com/YOUR_USERNAME/O-RAN-Intent-MANO-for-Network-Slicing \
  --path deployments/test \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace test-slice \
  --sync-policy automated
```

### 步驟 3: 觀察推拉過程

**在 Argo CD UI 中**：
1. 打開 `test-slice-app`
2. 您會看到：
   - 🔄 **Syncing**: Argo CD 正在從 Git 拉取
   - 📊 **OutOfSync → Synced**: 狀態變化
   - 📦 **Resources**: 創建的 Kubernetes 資源

**使用命令行**：
```bash
# 實時監控同步
watch kubectl get application test-slice-app -n argocd

# 查看事件
kubectl describe application test-slice-app -n argocd

# 查看創建的資源
kubectl get all -n test-slice
```

### 步驟 4: 觸發自動推拉

修改 Git 中的配置：
```bash
# 修改 test-app.yaml
sed -i 's/throughput: "1000"/throughput: "2000"/' test-app.yaml
git commit -am "Update throughput"
git push
```

**觀察 Argo CD 自動：**
1. 📥 Pull: 檢測到 Git 變更
2. ⚙️ Diff: 計算差異
3. 🔄 Sync: 自動同步到 Kubernetes
4. ✅ Healthy: 確認應用健康

---

## 📊 實時監控命令

### 完整監控 Dashboard

```bash
# 在不同終端運行以下命令

# Terminal 1: Argo CD 整體狀態
watch -n 2 'kubectl get applications -n argocd'

# Terminal 2: Repo Server 日誌（Git 拉取）
kubectl logs -n argocd deployment/argocd-repo-server -f --tail=50

# Terminal 3: Application Controller 日誌（同步）
kubectl logs -n argocd statefulset/argocd-application-controller -f --tail=50

# Terminal 4: 目標 Namespace 資源
watch -n 2 'kubectl get all -n test-slice'
```

### 關鍵日誌過濾

```bash
# 只看 Git 操作
kubectl logs -n argocd deployment/argocd-repo-server -f | \
  grep -E "Fetching|Cloning|Checkout|git"

# 只看同步操作
kubectl logs -n argocd statefulset/argocd-application-controller -f | \
  grep -E "Syncing|Sync operation|sync-status"

# 只看錯誤
kubectl logs -n argocd deployment/argocd-repo-server -f | grep -i error
```

---

## 🔍 查看特定網路切片的 Argo CD 工作流程

### 當您通過 Web UI 部署切片時：

**完整流程**：
```
用戶輸入 NL intent
    ↓
Go Orchestrator 解析
    ↓
創建 Argo CD Application
    ↓
Argo CD 工作流程開始：
    ↓
1. 📥 Repo Server: git clone/fetch
   └─ 日誌: kubectl logs -n argocd deployment/argocd-repo-server -f
    ↓
2. 📄 Repo Server: 讀取 manifests
   └─ 日誌會顯示: "manifest generation completed"
    ↓
3. ⚙️ Application Controller: 計算 diff
   └─ 日誌: kubectl logs -n argocd statefulset/argocd-application-controller -f
    ↓
4. 🔄 Application Controller: 執行 sync
   └─ 日誌會顯示: "Sync operation to <commit-hash> started"
    ↓
5. 📦 創建 Kubernetes 資源
   └─ kubectl get all -n <slice-namespace>
    ↓
6. ✅ Health Assessment
   └─ Argo CD UI 顯示綠色勾勾
```

### 實際命令示範

```bash
# 1. 查看當前有哪些應用
kubectl get applications -n argocd

# 2. 查看特定切片應用
kubectl describe application slice-embb-001 -n argocd

# 3. 實時監控同步狀態
kubectl get application slice-embb-001 -n argocd -w

# 4. 查看同步歷史
kubectl get application slice-embb-001 -n argocd -o jsonpath='{.status.history}'

# 5. 查看 Git commit 資訊
kubectl get application slice-embb-001 -n argocd -o jsonpath='{.status.sync.revision}'
```

---

## 🛠️ Troubleshooting

### 問題 1: 看不到任何 Application

**原因**: 目前系統還沒有配置 Argo CD 自動創建應用

**解決方案**：
1. 檢查 Orchestrator 是否配置了 Argo CD 整合
2. 查看 Orchestrator 日誌：
   ```bash
   tail -100 orchestrator.log | grep -i argocd
   ```

### 問題 2: Application 卡在 Syncing 狀態

```bash
# 檢查 repo-server 日誌
kubectl logs -n argocd deployment/argocd-repo-server

# 常見問題：
# - Git 認證失敗
# - 網路無法訪問 Git repository
# - Manifest 格式錯誤
```

### 問題 3: Git 拉取失敗

```bash
# 查看詳細錯誤
kubectl describe application <app-name> -n argocd | grep -A 10 "Conditions"

# 檢查 Git credentials
kubectl get secret -n argocd | grep repo
```

### 問題 4: 無法訪問 UI

```bash
# 檢查 port-forward 是否運行
ps aux | grep "port-forward"

# 重新啟動 port-forward
kubectl port-forward svc/argocd-server -n argocd 8082:443

# 檢查 Argo CD Server 狀態
kubectl get pods -n argocd | grep server
kubectl logs -n argocd deployment/argocd-server
```

---

## 📈 監控指標

### Prometheus Metrics

Argo CD 提供 Prometheus metrics：

```bash
# 訪問 metrics
kubectl port-forward svc/argocd-metrics -n argocd 8083:8082

# 然後訪問: http://localhost:8083/metrics
```

**關鍵指標**：
- `argocd_app_sync_total`: 同步次數
- `argocd_app_sync_status`: 同步狀態
- `argocd_git_request_duration_seconds`: Git 操作耗時
- `argocd_app_reconcile_duration_seconds`: 協調耗時

---

## 🎯 最佳實踐

### 1. 啟用詳細日誌

```bash
# 修改 argocd-cmd-params-cm
kubectl edit configmap argocd-cmd-params-cm -n argocd

# 添加：
data:
  controller.log.level: "debug"
  server.log.level: "debug"
  reposerver.log.level: "debug"

# 重啟 pods
kubectl rollout restart deployment argocd-repo-server -n argocd
kubectl rollout restart deployment argocd-server -n argocd
```

### 2. 配置 Webhook

讓 Git 主動通知 Argo CD（而非輪詢）：

```bash
# 在 GitHub repository 設置 Webhook
# Payload URL: https://<argocd-server>/api/webhook
# Content type: application/json
# Events: Just the push event
```

### 3. 啟用 Sync Windows

控制同步時間：

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  syncPolicy:
    syncOptions:
    - CreateNamespace=true
    syncWindows:
    - kind: allow
      schedule: '0 8 * * *'  # 每天 8:00 AM
      duration: 1h
```

---

## 📚 參考資源

- [Argo CD 官方文檔](https://argo-cd.readthedocs.io/)
- [監控最佳實踐](https://argo-cd.readthedocs.io/en/stable/operator-manual/metrics/)
- [Troubleshooting Guide](https://argo-cd.readthedocs.io/en/stable/operator-manual/troubleshooting/)

---

## 🔗 相關文件

- `docs/E2E_IMPLEMENTATION_GUIDE.md`: 完整實現指南
- `docs/CLAUDE_CLI_INTEGRATION.md`: Claude CLI 整合文檔
- `web/monitor.html`: 網路切片監控頁面

---

**最後更新**: 2025-09-30
**Argo CD 版本**: v2.8+
**Kubernetes 版本**: v1.27+