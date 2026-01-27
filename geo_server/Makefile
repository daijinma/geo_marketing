# LLM Sentry Monitor Makefile

VENV = .venv

# 颜色输出
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help install sync playwright dev run stats stats-full clean test

help:
	@echo "╔════════════════════════════════════════════════════════╗"
	@echo "║  ${GREEN}LLM Sentry Monitor - 管理命令${NC}                   ║"
	@echo "╚════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "${YELLOW}📦 环境管理${NC}"
	@echo "  make install      - 创建虚拟环境并安装依赖"
	@echo "  make sync         - 同步依赖（检查并安装缺失的库）"
	@echo "  make playwright   - 安装 Playwright 浏览器"
	@echo "  make clean        - 清理临时文件和缓存"
	@echo ""
	@echo "${YELLOW}🚀 运行服务${NC}"
	@echo "  make dev          - 启动 API 开发服务器（端口 8000）"
	@echo "  make run          - 执行 GEO 监测任务（配置文件模式）"
	@echo ""
	@echo "${YELLOW}📊 数据分析${NC}"
	@echo "  make stats        - 生成基础深度洞察报告"
	@echo "  make stats-full   - 生成完整深度洞察报告"
	@echo ""
	@echo "${YELLOW}💡 快速开始${NC}"
	@echo "  make install && make playwright  - 首次安装（推荐）"
	@echo "  make dev                         - 启动 API 服务"
	@echo ""

install:
	@echo "${GREEN}📦 安装监控服务依赖${NC}"
	@command -v uv >/dev/null 2>&1 || { echo "${RED}❌ uv 未安装，请参考: https://github.com/astral-sh/uv${NC}"; exit 1; }
	@echo "  → 创建虚拟环境..."
	@uv venv $(VENV)
	@$(MAKE) sync
	@echo "${GREEN}✅ 安装完成${NC}"

sync:
	@echo "${GREEN}🔄 同步监控服务依赖${NC}"
	@echo "  → 安装 requirements.txt 中的依赖..."
	@uv pip install -r requirements.txt
	@echo "  → 安装项目本身（可编辑模式）..."
	@uv pip install -e .
	@echo "${GREEN}✅ 依赖同步完成${NC}"

playwright:
	@echo "${GREEN}🎭 安装 Playwright 浏览器${NC}"
	@uv run playwright install chromium
	@echo "${GREEN}✅ Playwright 浏览器安装完成${NC}"

dev:
	@echo "${GREEN}🚀 启动监控 API 服务器 (http://localhost:8000)${NC}"
	@echo "  → API 文档: http://localhost:8000/docs"
	@echo "  → 使用 .env.development 配置"
	@echo ""
	@lsof -ti:8000 | xargs kill -9 2>/dev/null || true
	@ENV_FILE=.env.development PYTHONPATH=. uv run uvicorn api.app:app --host 0.0.0.0 --port 8000 --reload

run:
	@echo "${GREEN}🚀 执行 GEO 监测任务（配置文件模式）${NC}"
	@if [ ! -f "config.yaml" ]; then \
		echo "${RED}❌ 配置文件 config.yaml 不存在${NC}"; \
		echo "请创建配置文件或使用 API 模式"; \
		exit 1; \
	fi
	@PYTHONPATH=. uv run python main.py

stats:
	@echo "${GREEN}📊 生成基础深度洞察报告${NC}"
	@uv run python stats.py

stats-full:
	@echo "${GREEN}📊 生成完整深度洞察报告${NC}"
	@uv run python stats_full.py

test:
	@echo "${GREEN}🧪 运行测试${NC}"
	@uv run pytest

clean:
	@echo "${GREEN}🧹 清理临时文件${NC}"
	@rm -rf browser_data/* 2>/dev/null || true
	@rm -rf __pycache__ 2>/dev/null || true
	@rm -rf **/__pycache__ 2>/dev/null || true
	@rm -rf .pytest_cache 2>/dev/null || true
	@rm -rf .vite 2>/dev/null || true
	@echo "${GREEN}✅ 清理完成（保留了虚拟环境）${NC}"
	@echo ""
	@echo "如需完全清理虚拟环境，请运行："
	@echo "  ${YELLOW}rm -rf $(VENV)${NC}"
