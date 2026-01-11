VENV = .venv

.PHONY: setup db-up db-down db-logs db-reset db-upgrade run install sync playwright-install dev stats stats-full status clean help

help:
	@echo "LLM Sentry 开发指令集:"
	@echo "  make setup            - 安装所有依赖 (Python & Playwright)"
	@echo "  make db-up           - 启动 PostgreSQL 数据库容器"
	@echo "  make db-down         - 停止 PostgreSQL 数据库容器"
	@echo "  make db-reset        - 重置数据库 (删除旧数据并重建)"
	@echo "  make db-upgrade      - 升级数据库到最新版本 (v1.0 -> v2.0)"
	@echo "  make db-logs         - 查看数据库日志"
	@echo "  make install         - 创建虚拟环境并安装依赖"
	@echo "  make sync            - 同步依赖（检查并安装缺失的库）"
	@echo "  make playwright-install - 安装 Playwright 浏览器"
	@echo "  make run             - 执行 GEO 监测任务"
	@echo "  make dev             - 启动 API 开发服务器"
	@echo "  make stats           - 生成基础深度洞察报告（简单版）"
	@echo "  make stats-full      - 生成完整深度洞察报告（包含所有分析维度）"
	@echo "  make status          - 查看服务状态"
	@echo "  make clean           - 停止数据库并清理临时文件"

setup:
	./scripts/setup_monitor.sh

db-up:
	./scripts/start_db.sh

db-down:
	cd geo_db && docker-compose down

db-reset:
	./scripts/reset_db.sh

db-upgrade:
	cd geo_db && chmod +x upgrade_db.sh && ./upgrade_db.sh

db-logs:
	cd geo_db && docker-compose logs -f

run:
	./scripts/run_monitor.sh

# 安装：创建虚拟环境并同步依赖
install:
	@echo "正在创建虚拟环境..."
	cd llm_sentry_monitor && uv venv $(VENV)
	@$(MAKE) sync

# 同步依赖：检查并安装缺失的库
sync:
	@echo "正在检查并更新依赖..."
	cd llm_sentry_monitor && uv pip install -e .

# 安装 Playwright 浏览器
playwright-install:
	@echo "正在安装 Playwright 浏览器..."
	cd llm_sentry_monitor && uv run playwright install chromium
	@echo "✅ Playwright 浏览器安装完成"

# 启动开发服务器：先执行 sync 确保库是最新的
dev:
	@echo "正在启动服务..."
	@lsof -ti:8000 | xargs kill -9 2>/dev/null || true
	cd llm_sentry_monitor && PYTHONPATH=. uv run python api.py

stats:
	cd llm_sentry_monitor && uv run python stats.py

stats-full:
	cd llm_sentry_monitor && uv run python stats_full.py

status:
	@echo "--- Docker 容器状态 ---"
	@docker ps --filter "name=geo_db"
	@echo "\n--- 虚拟环境状态 ---"
	@if [ -d "llm_sentry_monitor/.venv" ]; then echo "✅ 虚拟环境已就绪"; else echo "❌ 虚拟环境未创建"; fi

clean:
	@echo "正在清理环境..."
	cd geo_db && make down
	rm -rf llm_sentry_monitor/browser_data/*
	@echo "✅ 清理完成 (保留了 .venv 以加快下次启动)"
