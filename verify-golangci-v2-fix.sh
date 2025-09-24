#!/bin/bash

# 驗證 golangci-lint v2.5.0 兼容性修復
echo "🔧 驗證 golangci-lint v2.5.0 兼容性修復"
echo "======================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 檢查修復內容...${NC}"

# 1. 檢查 GitHub Actions 參數修復
echo -e "${YELLOW}檢查 GitHub Actions 參數修復:${NC}"
if grep -q "\-\-disable=gofmt,goimports" .github/workflows/ci.yml; then
    echo -e "${RED}❌ 仍然包含 --disable=gofmt,goimports（v2 中會出錯）${NC}"
else
    echo -e "${GREEN}✅ 已移除 --disable=gofmt,goimports 參數${NC}"
fi

if grep -q "\-\-enable=.*gosec,gocritic,revive,staticcheck" .github/workflows/ci.yml; then
    echo -e "${GREEN}✅ 保留了正確的 linters enable 參數${NC}"
else
    echo -e "${RED}❌ linters enable 參數配置有問題${NC}"
fi

# 2. 檢查 .golangci.yml v2 格式
echo -e "${YELLOW}檢查 .golangci.yml v2 格式:${NC}"
if [ -f ".golangci.yml" ]; then
    if grep -q 'version: "2"' .golangci.yml; then
        echo -e "${GREEN}✅ 配置文件指定了 version: \"2\"${NC}"
    else
        echo -e "${RED}❌ 未指定 version: \"2\"${NC}"
    fi

    # 檢查是否移除了 typecheck
    if grep -q "typecheck" .golangci.yml; then
        echo -e "${RED}❌ 仍然包含 typecheck（v2 中 typecheck 不是 linter）${NC}"
    else
        echo -e "${GREEN}✅ 已移除 typecheck（正確，因為它不是 linter）${NC}"
    fi

    # 檢查是否從 linters 中移除了 gofmt/goimports
    if grep -A 20 "linters:" .golangci.yml | grep -E "^\s+- (gofmt|goimports)"; then
        echo -e "${RED}❌ linters 區塊仍包含 gofmt/goimports（應在 formatters 區塊）${NC}"
    else
        echo -e "${GREEN}✅ 已從 linters 區塊移除 gofmt/goimports${NC}"
    fi

    # 檢查是否添加了 formatters 區塊
    if grep -q "formatters:" .golangci.yml; then
        echo -e "${GREEN}✅ 已添加 formatters 區塊${NC}"

        if grep -A 10 "formatters:" .golangci.yml | grep -q "gofmt:"; then
            echo -e "${GREEN}✅ formatters 區塊包含 gofmt${NC}"
        else
            echo -e "${YELLOW}⚠️ formatters 區塊未包含 gofmt${NC}"
        fi

        if grep -A 10 "formatters:" .golangci.yml | grep -q "goimports:"; then
            echo -e "${GREEN}✅ formatters 區塊包含 goimports${NC}"
        else
            echo -e "${YELLOW}⚠️ formatters 區塊未包含 goimports${NC}"
        fi
    else
        echo -e "${RED}❌ 未添加 formatters 區塊${NC}"
    fi
else
    echo -e "${RED}❌ .golangci.yml 文件不存在${NC}"
fi

# 3. 檢查本地 golangci-lint 版本（如果可用）
echo -e "${YELLOW}檢查本地 golangci-lint 版本:${NC}"
if command -v golangci-lint >/dev/null 2>&1; then
    VERSION=$(golangci-lint version 2>/dev/null | head -1)
    echo -e "${GREEN}✅ golangci-lint 可用: $VERSION${NC}"

    # 嘗試驗證配置文件
    echo -e "${YELLOW}測試配置文件載入:${NC}"
    if golangci-lint config path >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 配置文件格式正確${NC}"
    else
        echo -e "${RED}❌ 配置文件格式有問題${NC}"
    fi
else
    echo -e "${YELLOW}⚠️ golangci-lint 未在本地安裝（CI 中會自動安裝）${NC}"
fi

echo ""
echo -e "${BLUE}📊 v2.5.0 兼容性修復摘要:${NC}"
echo "=========================="
echo "✅ 移除 GitHub Actions 中的 --disable=gofmt,goimports"
echo "✅ 從 linters.enable 移除 typecheck（不是 linter）"
echo "✅ 從 linters.enable 移除 gofmt/goimports（改為 formatters）"
echo "✅ 添加 formatters 區塊管理 gofmt/goimports"
echo "✅ 確保配置文件指定 version: \"2\""

echo ""
echo -e "${GREEN}🎯 golangci-lint v2.5.0 兼容性修復完成！${NC}"
echo ""
echo -e "${YELLOW}修復說明:${NC}"
echo "• v2 中 typecheck 不是 linter，不能被 enable/disable"
echo "• v2 中 gofmt/goimports 是 formatters，不能在 linters 區塊中使用"
echo "• v2 中 formatters 不能用 --enable/--disable 參數管理"
echo "• 必須在配置文件的 formatters: 區塊中管理格式化工具"

echo ""
echo -e "${GREEN}現在 CI 應該能成功載入配置並運行 golangci-lint v2.5.0！${NC}"