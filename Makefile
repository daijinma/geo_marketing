.PHONY: setup db-up db-down db-logs db-reset db-upgrade run stats stats-full status clean help

help:
	@echo "LLM Sentry 开发指令集:"
	@echo "  make setup      - 安装所有依赖 (Python & Playwright)"
	@echo "  make db-up      - 启动 PostgreSQL 数据库容器"
	@echo "  make db-down    - 停止 PostgreSQL 数据库容器"
	@echo "  make db-reset   - 重置数据库 (删除旧数据并重建)"
	@echo "  make db-upgrade - 升级数据库到最新版本 (v1.0 -> v2.0)"
	@echo "  make db-logs    - 查看数据库日志"
	@echo "  make run        - 执行 GEO 监测任务"
	@echo "  make stats      - 生成基础深度洞察报告（简单版）"
	@echo "  make stats-full - 生成完整深度洞察报告（包含所有分析维度）"
	@echo "  make status     - 查看服务状态"
	@echo "  make clean      - 停止数据库并清理临时文件"

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
