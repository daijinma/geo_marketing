# GEO Marketing - 项目导航 Makefile
# 
# 本 Makefile 提供快速导航到各个独立项目
# 每个项目都有自己的 Makefile 来管理命令

# 颜色输出
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: help

help:
	@echo "╔═══════════════════════════════════════════════════════════════╗"
	@echo "║  ${GREEN}GEO Marketing - 项目导航${NC}                                   ║"
	@echo "╚═══════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "本仓库包含以下独立项目，每个项目都有自己的 Makefile："
	@echo ""
	@echo "${YELLOW}📁 项目列表${NC}"
	@echo ""
	@echo "  ${BLUE}1. geo_db${NC} - PostgreSQL 数据库服务"
	@echo "     ${GREEN}cd geo_db && make help${NC}"
	@echo "     常用命令: make up, make down, make logs"
	@echo ""
	@echo "  ${BLUE}2. geo_server${NC} - Python 后端服务"
	@echo "     ${GREEN}cd geo_server && make help${NC}"
	@echo "     常用命令: make install, make dev, make run"
	@echo ""
	@echo "  ${BLUE}3. geo_client${NC} - Electron 桌面客户端"
	@echo "     ${GREEN}cd geo_client && make help${NC}"
	@echo "     常用命令: make setup, make dev, make build"
	@echo ""
	@echo "${YELLOW}🚀 快速开始${NC}"
	@echo ""
	@echo "  # 终端 1: 启动数据库"
	@echo "  cd geo_db && make up"
	@echo ""
	@echo "  # 终端 2: 启动后端服务"
	@echo "  cd geo_server && make install && make dev"
	@echo ""
	@echo "  # 终端 3: 启动客户端"
	@echo "  cd geo_client && make setup && make dev"
	@echo ""
	@echo "${YELLOW}📚 更多信息${NC}"
	@echo "  查看各项目的 README.md 了解详细说明"
	@echo ""
