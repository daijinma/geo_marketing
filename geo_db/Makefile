# GEO Database Makefile

# 颜色输出
GREEN = \033[0;32m
YELLOW = \033[0;33m
RED = \033[0;31m
NC = \033[0m # No Color

.PHONY: help up down logs ps clean reset upgrade status

help:
	@echo "╔════════════════════════════════════════════════════════╗"
	@echo "║  ${GREEN}GEO Database - 管理命令${NC}                          ║"
	@echo "╚════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "${YELLOW}🗄️  数据库服务${NC}"
	@echo "  make up        - 启动 PostgreSQL 数据库容器"
	@echo "  make down      - 停止 PostgreSQL 数据库容器"
	@echo "  make logs      - 查看数据库日志（实时）"
	@echo "  make ps        - 查看容器状态"
	@echo "  make status    - 查看数据库服务状态"
	@echo "  make reset     - 重置数据库（删除所有数据并重建）"
	@echo "  make upgrade   - 升级数据库到最新版本"
	@echo "  make clean     - 清理所有数据和容器"
	@echo ""

up:
	@echo "${GREEN}🗄️  启动 PostgreSQL 数据库${NC}"
	@docker-compose up -d
	@echo "${GREEN}✅ 数据库启动成功${NC}"
	@echo "  → 主机: localhost"
	@echo "  → 端口: 5432"
	@echo "  → 数据库: geo_sentry"
	@echo "  → 用户: geo_user"

down:
	@echo "${YELLOW}⏹️  停止 PostgreSQL 数据库${NC}"
	@docker-compose down
	@echo "${GREEN}✅ 数据库已停止${NC}"

logs:
	@echo "${GREEN}📋 查看数据库日志（Ctrl+C 退出）${NC}"
	@docker-compose logs -f

ps:
	@docker-compose ps

status:
	@echo "${GREEN}🔍 数据库服务状态${NC}"
	@docker ps --filter "name=geo_db" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "${RED}❌ 数据库未运行${NC}"
	@echo ""
	@lsof -i:5432 >/dev/null 2>&1 && echo "${GREEN}✅ PostgreSQL 端口 5432 已开放${NC}" || echo "${RED}❌ PostgreSQL 端口 5432 未开放${NC}"

reset:
	@echo "${RED}⚠️  警告：此操作将删除所有数据！${NC}"
	@read -p "确认重置数据库？(输入 yes 继续): " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		echo "${YELLOW}正在停止容器...${NC}"; \
		docker-compose down -v; \
		echo "${YELLOW}正在清理数据...${NC}"; \
		rm -rf postgres_data/*; \
		echo "${YELLOW}正在重新启动...${NC}"; \
		docker-compose up -d; \
		echo "${GREEN}✅ 数据库重置完成${NC}"; \
	else \
		echo "${YELLOW}已取消操作${NC}"; \
	fi

upgrade:
	@echo "${GREEN}⬆️  升级数据库到最新版本${NC}"
	@chmod +x upgrade_db.sh
	@./upgrade_db.sh

clean:
	@echo "${RED}🧹 清理数据库容器和数据${NC}"
	@docker-compose down -v
	@rm -rf postgres_data/*
	@echo "${GREEN}✅ 清理完成${NC}"
