# DevContainer 重建指南

## 🚀 快速重建步驟

### 方法 1: VS Code Command Palette (推薦)

1. **打開 Command Palette**
   - Windows/Linux: `Ctrl+Shift+P`
   - macOS: `Cmd+Shift+P`

2. **執行重建命令**
   ```
   Dev Containers: Rebuild Container
   ```
   或
   ```
   Dev Containers: Rebuild Container Without Cache
   ```
   (無緩存重建，首次建議使用)

3. **等待構建完成**
   - 預計時間：2-4 分鐘（首次）
   - 後續重建：30-60 秒

### 方法 2: VS Code 狀態列

1. 點擊左下角的綠色圖標 `><`
2. 選擇 "Rebuild Container"

### 方法 3: 命令行 (高級用戶)

```bash
# 1. 確保在專案根目錄
cd C:\Users\thc1006\Desktop\dev\O-RAN-Intent-MANO-for-Network-Slicing

# 2. 停止並移除現有容器
docker ps -a | grep devcontainer | awk '{print $1}' | xargs docker stop
docker ps -a | grep devcontainer | awk '{print $1}' | xargs docker rm

# 3. 移除舊的命名卷（可選，清理緩存）
docker volume ls | grep devcontainer | awk '{print $2}' | xargs docker volume rm

# 4. 使用 devcontainer CLI 重建
npx @devcontainers/cli build --workspace-folder .

# 5. 啟動容器
npx @devcontainers/cli up --workspace-folder .
```

## 📋 構建過程監控

### 預期構建階段

#### 階段 1: 基礎映像拉取 (30-60 秒)
```
[+] Pulling image mcr.microsoft.com/devcontainers/universal:2-linux
```

#### 階段 2: Features 安裝 (1-2 分鐘)
```
[+] Installing features:
  - python:1 (version 3.11)
  - go:1 (version 1.24.7)
  - node:1 (version 20)
  - docker-outside-of-docker:1
  - kubectl-helm-minikube:1
  - kind:1 (version 0.23.0)
  - pre-commit:2
  - github-cli:1
```

#### 階段 3: onCreateCommand (並行執行，1-2 分鐘)
```
[+] Running onCreateCommand:
  ✓ install-tools: Installing Go tools (golangci-lint, gosec, etc.)
  ✓ install-kubebuilder: Downloading Kubebuilder 3.15.1
  ✓ install-system-tools: Installing iperf3
```

#### 階段 4: postCreateCommand (30-60 秒)
```
[+] Running postCreateCommand:
  ✓ Bootstrap script execution
  ✓ Pre-commit hooks installation
  ✓ Security check execution
```

#### 階段 5: postStartCommand (5-10 秒)
```
[+] Running postStartCommand:
  ✓ Environment verification
  ✓ Git safe directory configuration
```

## ✅ 構建成功指標

### 終端輸出應顯示：
```
✅ DevContainer started successfully
✅ Post-create setup complete!
✅ Security checks completed successfully!

Go version: go1.24.7 linux/amd64
Python version: 3.11.x
Node version: v20.x.x
kubectl version: v1.31.0
kind version: v0.23.0
helm version: v3.16.2

Run 'make help' to see available commands
```

## 🔍 驗證測試

重建完成後，執行以下測試：

### 1. 基本環境驗證
```bash
# 檢查 Go 版本
go version
# 預期: go version go1.24.7 linux/amd64

# 檢查 Python 版本
python --version
# 預期: Python 3.11.x

# 檢查 Node 版本
node --version
# 預期: v20.x.x
```

### 2. Kubernetes 工具驗證
```bash
# kubectl
kubectl version --client --short
# 預期: Client Version: v1.31.0

# helm
helm version --short
# 預期: v3.16.2

# kind
kind version
# 預期: kind v0.23.0
```

### 3. Go 開發工具驗證
```bash
# golangci-lint
golangci-lint version
# 預期: 應有版本輸出

# gosec
gosec --version
# 預期: 應有版本輸出

# kubebuilder
kubebuilder version
# 預期: Version: 3.15.1

# setup-envtest
setup-envtest --help
# 預期: 應有幫助輸出
```

### 4. Docker 功能驗證
```bash
# 檢查 Docker 可用性
docker version
# 預期: 應顯示客戶端和服務器版本

# 測試構建
docker build -t test:latest -f - . <<EOF
FROM alpine:latest
RUN echo "Test build successful"
EOF
# 預期: Successfully built

# 測試 docker-compose
docker-compose version
# 預期: 應有版本輸出
```

### 5. Kind 集群測試
```bash
# 創建測試集群
kind create cluster --name devcontainer-test --wait 300s

# 驗證集群
kubectl cluster-info --context kind-devcontainer-test

# 清理
kind delete cluster --name devcontainer-test
```

### 6. 安全配置驗證
```bash
# 檢查容器不是特權模式
docker inspect $(hostname) | grep -i '"Privileged": false'
# 預期: 應找到 "Privileged": false

# 檢查能力配置
docker inspect $(hostname) | grep -A 20 "CapAdd"
# 預期: 應看到 NET_ADMIN, SYS_PTRACE

# 運行安全掃描
bash /workspace/.devcontainer/scripts/devcontainer-security-check.sh
# 預期: 應通過所有檢查
```

### 7. 專案構建測試
```bash
# 驗證環境
make verify-env
# 預期: Environment verified

# 運行 Go 模組下載
go mod download
# 預期: 應成功下載

# 構建專案（示例）
cd orchestrator && go build -v ./...
# 預期: 應成功構建
```

### 8. 性能驗證
```bash
# 測試 Go 構建緩存
time go build ./...
# 第一次: ~1-2 分鐘
# 第二次: ~5-10 秒（使用緩存）

# 檢查卷掛載
docker volume ls | grep devcontainer
# 預期: 應看到 4-5 個命名卷
```

## ❌ 常見問題排查

### 問題 1: Features 安裝失敗
**症狀:**
```
Error: Failed to install feature 'docker-outside-of-docker'
```

**解決方案:**
```bash
# 1. 檢查 Docker 服務運行
docker info

# 2. 清理 Docker 緩存
docker system prune -a -f

# 3. 重新拉取基礎映像
docker pull mcr.microsoft.com/devcontainers/universal:2-linux

# 4. 重建（無緩存）
Dev Containers: Rebuild Container Without Cache
```

### 問題 2: onCreateCommand 超時
**症狀:**
```
Error: Command 'install-tools' timed out
```

**解決方案:**
```bash
# 1. 增加超時時間（devcontainer.json）
"onCreateCommand": {
  "install-tools": {
    "command": "go install ...",
    "timeout": 600000  // 10 分鐘
  }
}

# 2. 或手動安裝工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 問題 3: 權限錯誤
**症狀:**
```
Error: Permission denied
```

**解決方案:**
```bash
# 1. 檢查用戶
whoami
# 應該是 'vscode'

# 2. 修復文件權限
sudo chown -R vscode:vscode /workspace

# 3. 檢查卷權限
docker volume inspect devcontainer-go-mod-cache
```

### 問題 4: Docker 命令不可用
**症狀:**
```
bash: docker: command not found
```

**解決方案:**
```bash
# 1. 驗證 feature 安裝
ls -la /usr/local/bin/docker

# 2. 檢查 Docker socket
ls -la /var/run/docker.sock

# 3. 手動安裝 Docker CLI（如果需要）
curl -fsSL https://get.docker.com | sh
```

### 問題 5: Git 操作失敗
**症狀:**
```
fatal: detected dubious ownership in repository
```

**解決方案:**
```bash
# 添加安全目錄
git config --global --add safe.directory /workspace

# 或在 postStartCommand 中自動執行（已配置）
```

### 問題 6: 卷掛載問題（Windows）
**症狀:**
```
Error: Cannot mount volume
```

**解決方案:**
```bash
# 1. 確保 Docker Desktop 有正確的文件共享權限
# Settings → Resources → File Sharing → 添加專案目錄

# 2. 重啟 Docker Desktop

# 3. 清理卷並重建
docker volume prune -f
```

## 🔄 重建最佳實踐

### 何時需要完整重建（無緩存）：

1. **更新 devcontainer.json features**
2. **更改基礎映像**
3. **修改 onCreateCommand**
4. **首次使用新配置**

### 何時可以快速重建（有緩存）：

1. **僅修改 VS Code 設置**
2. **更新擴展列表**
3. **修改 postStartCommand**
4. **調整環境變數**

### 緩存管理：

```bash
# 查看卷使用情況
docker system df -v

# 清理未使用的卷（保留命名卷）
docker volume prune -f

# 完全清理（包括命名卷，慎用）
docker volume rm $(docker volume ls -q | grep devcontainer)
```

## 📊 性能基準

### 預期構建時間（首次）：

| 階段 | 時間 | 備註 |
|------|------|------|
| 映像拉取 | 30-60s | 取決於網路速度 |
| Features 安裝 | 60-120s | 7 個 features |
| onCreateCommand | 60-120s | 並行安裝工具 |
| postCreateCommand | 30-60s | Bootstrap + hooks |
| **總計** | **3-6 分鐘** | 首次構建 |

### 預期構建時間（後續）：

| 階段 | 時間 | 備註 |
|------|------|------|
| 映像檢查 | 5-10s | 使用緩存 |
| Features 檢查 | 10-20s | 已安裝 |
| onCreateCommand | 跳過 | 僅首次運行 |
| postCreateCommand | 10-20s | 快速驗證 |
| **總計** | **30-60 秒** | 使用緩存 |

## 📝 構建日誌位置

- **VS Code 輸出面板**: 查看 → 輸出 → "Dev Containers"
- **Docker 日誌**: `docker logs <container_id>`
- **命令輸出**: 集成終端會顯示所有命令輸出

## 🆘 獲取幫助

如果遇到問題：

1. **查看完整日誌**
   ```bash
   # DevContainer 日誌
   VS Code → 查看 → 輸出 → Dev Containers

   # Docker 日誌
   docker logs $(docker ps -q --filter "name=devcontainer")
   ```

2. **運行診斷**
   ```bash
   # 環境信息
   make info

   # 安全檢查
   bash .devcontainer/scripts/devcontainer-security-check.sh
   ```

3. **回滾到之前版本**
   ```bash
   git checkout HEAD~1 -- .devcontainer/
   # 然後重建
   ```

4. **報告問題**
   - 包含完整的錯誤日誌
   - `docker info` 輸出
   - `docker version` 輸出
   - 操作系統信息

---

**準備好了嗎？** 執行 `Ctrl+Shift+P` → `Dev Containers: Rebuild Container Without Cache`