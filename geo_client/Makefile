# GEO Client Makefile

NODE_PKG_MANAGER = pnpm

# 颜色输出
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help setup install compile dev dev-fast build electron-build type-check clean

help:
	@echo "╔════════════════════════════════════════════════════════╗"
	@echo "║  ${GREEN}GEO Client - 管理命令${NC}                            ║"
	@echo "╚════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "${YELLOW}📦 环境管理${NC}"
	@echo "  make setup       - 一键安装客户端（安装+编译）"
	@echo "  make install     - 安装客户端依赖（pnpm）"
	@echo "  make compile     - 编译 Electron 主进程代码"
	@echo ""
	@echo "${YELLOW}🚀 开发运行${NC}"
	@echo "  make dev         - 启动客户端开发服务器"
	@echo "  make dev-fast    - 快速启动客户端（跳过编译检查）"
	@echo ""
	@echo "${YELLOW}🔨 构建打包${NC}"
	@echo "  make build       - 构建客户端（生产构建）"
	@echo "  make electron-build - 打包客户端为可执行文件"
	@echo ""
	@echo "${YELLOW}🔍 代码检查${NC}"
	@echo "  make type-check  - 运行 TypeScript 类型检查"
	@echo ""
	@echo "${YELLOW}🧹 清理${NC}"
	@echo "  make clean       - 清理临时文件和构建产物"
	@echo ""
	@echo "${YELLOW}💡 快速开始${NC}"
	@echo "  make setup       - 首次安装（推荐）"
	@echo "  make dev         - 启动开发服务"
	@echo ""

setup:
	@echo "${GREEN}💻 一键安装客户端${NC}"
	@command -v $(NODE_PKG_MANAGER) >/dev/null 2>&1 || { echo "${RED}❌ pnpm 未安装，请运行: npm install -g pnpm${NC}"; exit 1; }
	@./scripts/setup.sh

install:
	@echo "${GREEN}📦 安装客户端依赖${NC}"
	@command -v $(NODE_PKG_MANAGER) >/dev/null 2>&1 || { echo "${RED}❌ pnpm 未安装，请运行: npm install -g pnpm${NC}"; exit 1; }
	@$(NODE_PKG_MANAGER) install
	@echo "${GREEN}✅ 客户端依赖安装完成${NC}"

compile:
	@echo "${GREEN}🔨 编译 Electron 主进程代码${NC}"
	@$(NODE_PKG_MANAGER) exec tsc -p electron
	@echo "${GREEN}✅ Electron 主进程编译完成${NC}"

dev:
	@echo "${GREEN}🚀 启动客户端开发服务器${NC}"
	@echo "  → 前端: http://localhost:1420"
	@echo "  → Electron 窗口将自动打开"
	@echo ""
	@if [ ! -f "dist-electron/main.js" ]; then \
		echo "${YELLOW}⚠️  检测到 Electron 主进程未编译，正在编译...${NC}"; \
		$(MAKE) compile; \
	fi
	@$(NODE_PKG_MANAGER) run electron:dev

dev-fast:
	@echo "${GREEN}🚀 快速启动客户端（跳过编译检查）${NC}"
	@$(NODE_PKG_MANAGER) run electron:dev

build:
	@echo "${GREEN}🔨 构建客户端（生产构建）${NC}"
	@$(NODE_PKG_MANAGER) run build
	@echo "${GREEN}✅ 客户端构建完成${NC}"

electron-build:
	@echo "${GREEN}📦 打包客户端为可执行文件${NC}"
	@$(NODE_PKG_MANAGER) run electron:build
	@echo "${GREEN}✅ 客户端打包完成（查看 release/ 目录）${NC}"

type-check:
	@echo "${GREEN}🔍 运行 TypeScript 类型检查${NC}"
	@$(NODE_PKG_MANAGER) run type-check

clean:
	@echo "${GREEN}🧹 清理临时文件${NC}"
	@rm -rf .vite 2>/dev/null || true
	@rm -rf dist 2>/dev/null || true
	@rm -rf dist-electron 2>/dev/null || true
	@rm -rf release 2>/dev/null || true
	@echo "${GREEN}✅ 清理完成（保留了 node_modules）${NC}"
	@echo ""
	@echo "如需完全清理 node_modules，请运行："
	@echo "  ${YELLOW}rm -rf node_modules${NC}"
