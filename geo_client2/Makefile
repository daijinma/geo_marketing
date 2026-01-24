# GEO Client2 Makefile - Go + Wails 项目

# 本地 Go 环境配置
GO_ENV = .go
GO_BIN = $(CURDIR)/$(GO_ENV)/bin
GO_PATH = $(CURDIR)/$(GO_ENV)

# 版本信息
VERSION := $(shell grep '"version"' frontend/package.json | head -1 | sed 's/.*"version": "\(.*\)".*/\1/')
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X geo_client2/backend.Version=$(VERSION) -X geo_client2/backend.BuildTime=$(BUILD_TIME)

# 颜色输出
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
BLUE = \033[0;34m
NC = \033[0m # No Color

# 设置本地 Go 环境变量
export GOBIN = $(GO_BIN)
export PATH := $(GO_BIN):$(PATH)

.PHONY: help install check-go check-wails check-env sync dev build build-win build-mac install-deps clean

help:
	@echo "╔════════════════════════════════════════════════════════╗"
	@echo "║  ${GREEN}GEO Client2 - Go + Wails 管理命令${NC}                  ║"
	@echo "╚════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "${YELLOW}📦 环境管理${NC}"
	@echo "  make install     - 创建本地 Go 环境并安装依赖（包括 Wails）"
	@echo "  make sync        - 同步依赖（go mod tidy + 前端依赖）"
	@echo "  make check-go    - 检查 Go 环境"
	@echo "  make check-wails - 检查 Wails 是否已安装"
	@echo "  make check-env   - 检查本地虚拟环境状态"
	@echo ""
	@echo "${YELLOW}🚀 开发运行${NC}"
	@echo "  make dev         - 启动开发服务器（Wails dev）"
	@echo ""
	@echo "${YELLOW}🔨 构建打包${NC}"
	@echo "  make build       - 构建应用（当前平台）"
	@echo "  make build-win   - 构建 Windows 版本"
	@echo "  make build-mac   - 构建 macOS 版本"
	@echo ""
	@echo "${YELLOW}🧹 清理${NC}"
	@echo "  make clean       - 清理临时文件和构建产物"
	@echo ""
	@echo "${YELLOW}💡 快速开始${NC}"
	@echo "  make install     - 首次安装（推荐）"
	@echo "  make dev         - 启动开发服务"
	@echo ""
	@echo "${BLUE}本地 Go 环境: $(GO_ENV)/${NC}"
	@echo "${BLUE}Go 工具目录: $(GO_BIN)${NC}"
	@echo "${BLUE}Wails 将安装到: $(GO_BIN)/wails${NC}"

check-go:
	@echo "${GREEN}🔍 检查 Go 环境${NC}"
	@command -v go >/dev/null 2>&1 || { \
		echo "${RED}❌ Go 未安装，请访问: https://go.dev/dl/${NC}"; \
		exit 1; \
	}
	@echo "  → Go 版本: $$(go version)"
	@echo "  → Go 路径: $$(which go)"
	@echo "${GREEN}✅ Go 环境正常${NC}"

check-env:
	@echo "${GREEN}🔍 检查本地虚拟环境${NC}"
	@if [ -d "$(GO_ENV)" ]; then \
		echo "  ✅ 本地 Go 环境: $(GO_ENV)/"; \
		echo "  → Go 工具目录: $(GO_BIN)"; \
		if [ -d "$(GO_BIN)" ]; then \
			echo "  → 已安装工具:"; \
			ls -1 $(GO_BIN) 2>/dev/null | sed 's/^/    - /' || echo "    (无)"; \
		fi; \
		echo "  → Go 版本: $$(go version)"; \
		echo "  → Go 模块路径: $$(go list -m)"; \
	else \
		echo "  ⚠️  本地 Go 环境未创建。运行: ${YELLOW}make install${NC}"; \
	fi

check-wails:
	@echo "${GREEN}🔍 检查 Wails${NC}"
	@if command -v wails >/dev/null 2>&1; then \
		echo "  ✅ Wails 已安装"; \
		echo "  → Wails 路径: $$(which wails)"; \
		wails version 2>/dev/null || echo "  → 版本: (无法获取)"; \
	else \
		echo "  ⚠️  Wails 未安装"; \
		echo "  → 运行 ${YELLOW}make install${NC} 安装 Wails 到本地环境"; \
		exit 1; \
	fi


install: check-go
	@echo "${GREEN}📦 安装项目依赖${NC}"
	@echo "  → 创建本地 Go 环境目录..."
	@mkdir -p $(GO_BIN)
	@echo "  → 同步 Go 模块依赖..."
	@go mod download
	@go mod tidy
	@echo "  → 安装 Wails CLI 到本地环境..."
	@go install github.com/wailsapp/wails/v2/cmd/wails@latest
	@echo "  → 安装前端依赖..."
	@cd frontend && pnpm install
	@echo "${GREEN}✅ 安装完成${NC}"
	@echo ""
	@echo "${BLUE}本地 Go 环境已创建: $(GO_ENV)/${NC}"
	@echo "${BLUE}Go 工具已安装到: $(GO_BIN)${NC}"
	@echo "${BLUE}Wails 已安装到: $(GO_BIN)/wails${NC}"

sync: check-go
	@echo "${GREEN}🔄 同步项目依赖${NC}"
	@echo "  → 同步 Go 模块..."
	@go mod download
	@go mod tidy
	@echo "  → 同步前端依赖..."
	@cd frontend && pnpm install
	@echo "${GREEN}✅ 依赖同步完成${NC}"

dev: check-go check-wails
	@echo "${GREEN}🚀 启动开发服务器${NC}"
	@echo "  → 使用本地 Go 环境: $(GO_ENV)/"
	@echo "  → Wails 路径: $$(which wails)"
	@echo "  → 前端开发服务器将自动启动"
	@echo "  → 版本: $(VERSION)"
	@echo "  → 构建时间: $(BUILD_TIME)"
	@echo ""
	@PATH="$(GO_BIN):$$PATH" wails dev -ldflags "$(LDFLAGS)"

build: check-go check-wails
	@echo "${GREEN}🔨 构建应用（当前平台）${NC}"
	@echo "  → 版本: $(VERSION)"
	@echo "  → 构建时间: $(BUILD_TIME)"
	@cd frontend && pnpm run build
	@PATH="$(GO_BIN):$$PATH" wails build -ldflags "$(LDFLAGS)"
	@echo "${GREEN}✅ 构建完成${NC}"

build-win: check-go check-wails
	@echo "${GREEN}🔨 构建 Windows 版本${NC}"
	@echo "  → 版本: $(VERSION)"
	@echo "  → 构建时间: $(BUILD_TIME)"
	@cd frontend && pnpm run build
	@PATH="$(GO_BIN):$$PATH" wails build -platform windows/amd64 -ldflags "$(LDFLAGS)"
	@echo "${GREEN}✅ Windows 版本构建完成${NC}"

build-mac: check-go check-wails
	@echo "${GREEN}🔨 构建 macOS 版本${NC}"
	@echo "  → 版本: $(VERSION)"
	@echo "  → 构建时间: $(BUILD_TIME)"
	@cd frontend && pnpm run build
	@PATH="$(GO_BIN):$$PATH" wails build -platform darwin/universal -ldflags "$(LDFLAGS)"
	@echo "${GREEN}✅ macOS 版本构建完成${NC}"

install-deps: sync

clean:
	@echo "${GREEN}🧹 清理临时文件${NC}"
	@rm -rf frontend/dist 2>/dev/null || true
	@rm -rf frontend/.vite 2>/dev/null || true
	@rm -rf wailsjs 2>/dev/null || true
	@rm -rf build 2>/dev/null || true
	@rm -rf bin 2>/dev/null || true
	@echo "${GREEN}✅ 清理完成（保留了本地 Go 环境）${NC}"
	@echo ""
	@echo "如需完全清理本地 Go 环境，请运行："
	@echo "  ${YELLOW}rm -rf $(GO_ENV)${NC}"
