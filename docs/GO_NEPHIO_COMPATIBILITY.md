# Go 1.24.7 + Nephio R3 版本相容性指南

## 核心版本要求

### Go 語言版本
- **必須使用**: Go 1.24.7
- **Toolchain**: go1.24.7
- **理由**: Nephio R3 採用最新 Go 以支援 Generic 與 WASM 實驗

### Nephio R3 相關版本

| 元件 | 版本 | 說明 |
|------|------|------|
| **Nephio** | R3 (v3.0.0) | 2024年10月發布 |
| **Kubernetes** | 1.26-1.29 | 叢集版本要求 |
| **containerd** | ≥ 1.6 | CRI v1 支援 |
| **kpt** | ≥ v1.0.0-beta.43 | 必須支援 Go 1.24 |
| **kpt-fn** | 最新版 | Function SDK |
| **Porch** | ≥ v4.1.0 | API breaking change |
| **porchctl** | 對應 Porch 版本 | CLI 工具 |
| **Config Sync** | 最新穩定版 | GitOps 引擎 |

### GitHub Actions 相容性

| Action | 版本 | 說明 |
|--------|------|------|
| **actions/setup-go** | @v5 | 支援 Go 1.24.7 |
| **actions/checkout** | @v5 | 最新穩定版 |
| **golangci/golangci-lint-action** | @v8 | 必須 ≥ v8 |
| **golangci-lint** | v2.5.0 | 支援 Go 1.24 |

## 升級檢查清單

### 1. Go 模組更新
```bash
# 更新 go.mod
go mod edit -go=1.24.7
go mod tidy

# 更新 go.work
go work edit -go=1.24.7
go work sync
```

### 2. Dockerfile 更新
```dockerfile
# 基礎映像必須使用
FROM golang:1.24.7-alpine AS builder
ENV GOTOOLCHAIN=go1.24.7
```

### 3. GitHub Actions 更新
```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.24.7'

- uses: golangci/golangci-lint-action@v8
  with:
    version: v2.5.0
```

### 4. 環境變數
```bash
export GO_VERSION=1.24.7
export GOTOOLCHAIN=go1.24.7
```

## 已知問題與解決方案

### 問題 1: golangci-lint 版本不相容
- **症狀**: `golangci-lint` 報錯 "requires Go >= 1.24"
- **解決**: 升級至 golangci-lint v2.5.0 或更高版本

### 問題 2: go.work 版本衝突
- **症狀**: CI 失敗，提示 "go.work lists go 1.22"
- **解決**: 更新 go.work 並執行 `go work sync`

### 問題 3: Docker 構建失敗
- **症狀**: Docker build 顯示 Go 版本不匹配
- **解決**: 確保 Dockerfile 使用 `golang:1.24.7-alpine`

### 問題 4: Porch API 不相容
- **症狀**: `revision` 欄位型別錯誤
- **解決**: 升級至 Porch v4.1.0+ 並更新 CRD

## 驗證步驟

1. **本地驗證**
```bash
go version  # 應顯示 go1.24.7
go mod tidy
go test ./...
```

2. **Docker 構建驗證**
```bash
docker build -t test:go1.24.7 .
docker run --rm test:go1.24.7 go version
```

3. **CI 驗證**
- 確認所有 GitHub Actions 通過
- 檢查 golangci-lint 正常運行
- 驗證 Docker 映像構建成功

## 相關文檔連結

- [Nephio R3 Release Notes](https://docs.nephio.org/docs/release-notes/r3/)
- [Go 1.24 Release Notes](https://go.dev/doc/go1.24)
- [golangci-lint Compatibility](https://github.com/golangci/golangci-lint)
- [Porch v4.1.0 Breaking Changes](https://github.com/nephio-project/porch/releases/tag/v4.1.0)

## 維護注意事項

1. **定期更新**: 每季度檢查 Nephio 和 Go 更新
2. **測試優先**: 升級前在開發環境完整測試
3. **版本鎖定**: 生產環境使用固定版本
4. **回退計劃**: 保留舊版本 Docker 映像和配置

---

最後更新: 2025-09-25
維護者: O-RAN Intent-MANO Team