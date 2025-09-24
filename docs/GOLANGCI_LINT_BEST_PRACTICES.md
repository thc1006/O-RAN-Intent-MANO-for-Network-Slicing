# golangci-lint v2.5.0 最佳實踐配置指南

## 為什麼選擇 v2.5.0？
- **Go 1.24.7 支援**: v2.5.0 是第一個完整支援 Go 1.24 的版本
- **新增 linters**: godoclint, unqueryvet, iotamixing
- **增強功能**: 改進 embeddedstructfieldcheck, ginkgolinter, ineffassign
- **穩定性**: 修復了多個 Go 1.24 相容性問題

## 安裝方式（優先順序）

### 1. GitHub Actions（推薦）
```yaml
# .github/workflows/lint.yml
name: golangci-lint
on:
  push:
    branches: [main, develop]
  pull_request:

permissions:
  contents: read
  pull-requests: read  # 用於 only-new-issues

jobs:
  golangci:
    name: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24.7'
          cache: true
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: v2.5.0
          args: --timeout=10m --config=.golangci.yml
          skip-cache: false
          skip-pkg-cache: false
          skip-build-cache: false
          only-new-issues: true  # PR 只顯示新問題
```

### 2. 本地安裝（二進制）
```bash
# 推薦：使用官方安裝腳本
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.5.0

# 驗證安裝
golangci-lint --version
```

### 3. Docker（CI/CD 環境）
```dockerfile
# Dockerfile
FROM golangci/golangci-lint:v2.5.0-alpine AS linter
WORKDIR /app
COPY . .
RUN golangci-lint run --timeout=10m

# 或直接運行
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.5.0 golangci-lint run
```

### 4. Makefile 整合
```makefile
# Makefile
GOLANGCI_VERSION := v2.5.0

.PHONY: lint-install
lint-install:
	@if ! command -v golangci-lint &> /dev/null || [ "$$(golangci-lint version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')" != "$(GOLANGCI_VERSION)" ]; then \
		echo "Installing golangci-lint $(GOLANGCI_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_VERSION); \
	fi

.PHONY: lint
lint: lint-install
	golangci-lint run --timeout=10m --config=.golangci.yml
```

## 配置文件最佳實踐

### .golangci.yml（完整配置）
```yaml
# .golangci.yml
run:
  go: '1.24'  # 指定 Go 版本
  timeout: 10m
  tests: true
  skip-dirs:
    - vendor
    - third_party
    - testdata
    - examples
    - .git
  skip-files:
    - ".*\\.pb\\.go$"
    - ".*\\.gen\\.go$"
    - "mock_.*\\.go$"

output:
  format: colored-line-number
  print-issued-lines: true
  print-linter-name: true
  unique-by-line: true
  path-prefix: ""
  sort-results: true

linters:
  enable:
    # 預設啟用
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    # 額外推薦
    - bodyclose
    - dupl
    - exhaustive
    - exportloopref
    - gci
    - gocognit
    - goconst
    - gocritic
    - gocyclo
    - godot
    - gofmt
    - goimports
    - goprintffuncname
    - gosec
    - lll
    - misspell
    - nakedret
    - prealloc
    - revive
    - rowserrcheck
    - sqlclosecheck
    - stylecheck
    - unconvert
    - unparam
    - whitespace
    # v2.5.0 新增
    - godoclint
    - unqueryvet
    - iotamixing
  disable:
    - depguard  # 太嚴格
    - exhaustivestruct  # 已棄用
    - gochecknoglobals  # 某些全局變量是必要的
    - gochecknoinits  # init 函數有時需要
    - goerr113  # 太嚴格
    - gomnd  # 魔術數字檢查太嚴格
    - wsl  # 空行規則太嚴格

linters-settings:
  gocognit:
    min-complexity: 30

  gocyclo:
    min-complexity: 15

  govet:
    check-shadowing: true
    enable-all: true

  lll:
    line-length: 140

  misspell:
    locale: US

  prealloc:
    simple: true
    for-loops: true
    range-loops: true

  revive:
    confidence: 0.8
    rules:
      - name: blank-imports
      - name: context-as-argument
      - name: context-keys-type
      - name: dot-imports
      - name: error-return
      - name: error-strings
      - name: error-naming
      - name: exported
      - name: if-return
      - name: increment-decrement
      - name: var-naming
      - name: var-declaration
      - name: range
      - name: receiver-naming
      - name: time-naming
      - name: unexported-return
      - name: indent-error-flow
      - name: errorf
      - name: empty-block
      - name: superfluous-else
      - name: unreachable-code

  gosec:
    severity: medium
    confidence: medium
    excludes:
      - G101  # 硬編碼憑證（測試用）
      - G204  # 命令注入（已控制）

  goimports:
    local-prefixes: github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing

  gci:
    sections:
      - standard
      - default
      - prefix(github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing)

  godoclint:
    minimum-function-doc-length: 10
    check-private: false

issues:
  exclude-rules:
    # 測試文件可以有較低的標準
    - path: _test\.go
      linters:
        - gocyclo
        - errcheck
        - dupl
        - gosec
        - gocognit

    # 生成的文件排除
    - path: \.pb\.go
      linters:
        - golint
        - stylecheck

    # Mock 文件排除
    - path: mock_.*\.go
      linters:
        - golint

  # 修復建議
  fix: false

  # 每個 linter 的最大問題數
  max-issues-per-linter: 0

  # 總的最大問題數
  max-same-issues: 0

  # 新功能：只顯示新增的問題（用於 PR）
  new: false
  new-from-rev: ""
  new-from-patch: ""
```

## CI/CD 整合最佳實踐

### 1. Pre-commit Hook
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v2.5.0
    hooks:
      - id: golangci-lint
```

### 2. VS Code 整合
```json
// .vscode/settings.json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": [
    "--config=.golangci.yml",
    "--fast"
  ],
  "go.lintOnSave": "workspace"
}
```

### 3. 並行執行優化
```yaml
# GitHub Actions 並行
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v8
  with:
    version: v2.5.0
    args: --timeout=10m --concurrency=4
```

## 效能優化技巧

### 1. 快取策略
```yaml
# GitHub Actions 快取
- uses: actions/cache@v4
  with:
    path: |
      ~/.cache/golangci-lint
      ~/.cache/go-build
    key: ${{ runner.os }}-golangci-lint-${{ hashFiles('**/go.sum') }}
```

### 2. 只檢查變更文件
```bash
# 本地開發
golangci-lint run --new-from-rev=main --timeout=5m
```

### 3. 快速模式
```bash
# 開發時使用快速模式
golangci-lint run --fast
```

## 常見問題排查

### 問題 1: timeout
```bash
# 增加 timeout
golangci-lint run --timeout=20m
```

### 問題 2: 記憶體不足
```bash
# 限制並行度
golangci-lint run --concurrency=2
```

### 問題 3: 版本衝突
```bash
# 清理快取
rm -rf ~/.cache/golangci-lint
golangci-lint cache clean
```

## 團隊協作建議

1. **版本統一**: 所有開發者使用相同版本 (v2.5.0)
2. **配置共享**: .golangci.yml 納入版本控制
3. **漸進式採用**: 先 warning 後 error
4. **定期更新**: 每季度評估升級
5. **文檔維護**: 記錄排除規則原因

## 監控指標

- **執行時間**: < 5 分鐘（本地）< 10 分鐘（CI）
- **誤報率**: < 5%
- **覆蓋率**: > 80% 的代碼被檢查
- **修復率**: > 90% 的問題被修復

## 參考資源

- [官方文檔](https://golangci-lint.run/)
- [GitHub Action](https://github.com/golangci/golangci-lint-action)
- [配置參考](https://golangci-lint.run/usage/configuration/)
- [Linters 列表](https://golangci-lint.run/usage/linters/)

---

最後更新: 2025-09-25
版本: golangci-lint v2.5.0 + Go 1.24.7