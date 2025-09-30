# 🚀 DevContainer 快速開始

## 立即重建（3 步驟）

### ⚡ 步驟 1: 打開 VS Code Command Palette

**Windows/Linux:**
```
Ctrl + Shift + P
```

**macOS:**
```
Cmd + Shift + P
```

### ⚡ 步驟 2: 執行重建命令

輸入並選擇：
```
Dev Containers: Rebuild Container Without Cache
```

或簡短版本：
```
>rebuild
```
（自動補全會顯示完整命令）

### ⚡ 步驟 3: 等待完成

**預計時間：**
- 首次構建：2-4 分鐘
- 後續重建：30-60 秒

**進度指示：**
```
[1/5] Pulling base image...           ⏳ 30s
[2/5] Installing features...          ⏳ 90s
[3/5] Installing tools...             ⏳ 90s
[4/5] Running bootstrap...            ⏳ 30s
[5/5] Starting container...           ⏳ 10s
```

---

## ✅ 驗證安裝

重建完成後，在終端執行：

```bash
# 快速驗證（30 秒）
bash .devcontainer/scripts/verify-devcontainer.sh
```

**預期輸出：**
```
🔍 DevContainer Verification Test Suite
========================================
✅ Passed:  45
❌ Failed:  0
⏭️  Skipped: 3
📊 Total:   48
📈 Success Rate: 100%
✅ All tests passed!
```

---

## 🎯 立即可用的功能

### 語言環境
- ✅ Go 1.24.7
- ✅ Python 3.11
- ✅ Node.js 20

### Kubernetes 工具
- ✅ kubectl 1.31.0
- ✅ helm 3.16.2
- ✅ kind 0.23.0

### 開發工具
- ✅ golangci-lint
- ✅ gosec
- ✅ kubebuilder
- ✅ pre-commit

### 容器工具
- ✅ Docker (via host)
- ✅ docker-compose

---

## 🚀 快速測試

### 測試 1: 創建 Kind 集群（1 分鐘）
```bash
kind create cluster --name test-cluster
kubectl cluster-info
kind delete cluster --name test-cluster
```

### 測試 2: Go 構建（30 秒）
```bash
cd orchestrator
go build -v ./...
```

### 測試 3: Docker 構建（30 秒）
```bash
docker build -t test:latest -f - . <<EOF
FROM alpine:latest
RUN echo "Success"
EOF
```

---

## 📊 性能提升驗證

### 驗證緩存加速

**首次 Go 構建：**
```bash
time go build ./...
# 預期: ~1-2 分鐘
```

**第二次 Go 構建（使用緩存）：**
```bash
time go build ./...
# 預期: ~5-10 秒
```

**性能提升：** 90%+ ⚡

---

## 🔒 安全驗證

```bash
# 檢查不是特權模式
docker inspect $(hostname) | grep '"Privileged": false'
# 預期: "Privileged": false

# 運行完整安全掃描
bash .devcontainer/scripts/devcontainer-security-check.sh
# 預期: ✅ Security checks completed successfully!
```

---

## 📝 常用命令

### 開發命令
```bash
make help              # 查看所有命令
make verify-env        # 驗證環境
make test              # 運行測試
make kind              # 創建 Kind 集群
```

### DevContainer 命令
```bash
# 重建容器
Ctrl+Shift+P → "Rebuild Container"

# 重新打開
Ctrl+Shift+P → "Reopen in Container"

# 查看日誌
Ctrl+Shift+P → "Show Container Log"
```

---

## ❌ 遇到問題？

### 快速修復

**問題：構建失敗**
```bash
# 清理並重建
docker system prune -f
# VS Code: Rebuild Container Without Cache
```

**問題：權限錯誤**
```bash
sudo chown -R vscode:vscode /workspace
```

**問題：Git 錯誤**
```bash
git config --global --add safe.directory /workspace
```

### 詳細診斷
```bash
# 運行完整驗證
bash .devcontainer/scripts/verify-devcontainer.sh

# 查看環境信息
make info
```

---

## 📚 詳細文檔

- **完整重建指南**: `.devcontainer/REBUILD_GUIDE.md`
- **變更日誌**: `docs/DEVCONTAINER_CHANGELOG.md`
- **安全評估**: `docs/DEVCONTAINER_SECURITY.md`

---

## 🎉 完成！

環境已準備就緒。開始開發：

```bash
# 創建開發集群
make kind

# 運行測試
make test

# 構建組件
make build

# 快樂編碼！ 🚀
```