# Monorepo 架构说明

本仓库采用 **Monorepo** 架构，将多个相关项目组织在同一个代码仓库中，便于统一管理和代码共享。

## 📁 项目结构

```
GEO/ (Root Monorepo)
├── geo_db/                          # 子项目 1: 数据库服务
│   ├── pyproject.toml              # 项目配置
│   ├── docker-compose.yml          # Docker 编排
│   ├── init.sql                    # 数据库初始化脚本 (v1.0)
│   ├── init_v2.sql                 # 数据库初始化脚本 (v2.0)
│   ├── upgrade_db.sh               # 数据库升级脚本
│   ├── migrations/                  # 数据库迁移脚本
│   │   ├── 001_upgrade_to_v2.sql   # v1.0 -> v2.0 迁移
│   │   └── 002_add_task_jobs.sql    # v2.0 -> v2.1 迁移
│   ├── Makefile                    # 数据库管理命令
│   └── README.md                   # 项目文档
│
├── llm_sentry_monitor/             # 子项目 2: 监控服务
│   ├── pyproject.toml              # 项目配置
│   ├── main.py                     # 入口文件（配置文件模式）
│   ├── api.py                      # FastAPI 应用（API 模式）
│   ├── config.yaml                 # 任务配置文件
│   ├── providers/                  # 多平台适配器
│   │   ├── base.py                 # 基础 Provider 抽象类
│   │   ├── deepseek_web.py         # DeepSeek 平台适配器
│   │   └── doubao_web.py            # 豆包平台适配器
│   ├── core/                       # 核心逻辑模块
│   │   ├── db.py                   # 数据库连接与操作
│   │   ├── parser.py               # 域名解析与分类
│   │   ├── task_executor.py        # 异步任务执行器
│   │   └── logger_config.py        # 日志配置
│   ├── stats.py                    # 基础统计分析报告
│   ├── stats_full.py               # 完整统计分析报告
│   ├── requirements.txt            # Python 依赖列表
│   └── README.md                   # 项目文档
│
├── scripts/                        # 共享脚本工具
│   ├── setup_monitor.sh            # 环境初始化脚本
│   ├── start_db.sh                 # 启动数据库脚本
│   ├── reset_db.sh                 # 重置数据库脚本
│   ├── run_monitor.sh              # 运行监控任务脚本
│   ├── curl_examples.sh             # API 测试脚本（cURL 示例）
│   └── analyze_sites.py             # 网站关联分析脚本
│
├── pyproject.toml                 # 根工作区配置
├── Makefile                        # 统一构建与运行指令
├── README.md                       # 根文档
└── MONOREPO.md                     # 本文件
```

## 🔧 工作区管理

### 使用 uv workspace

本项目使用 `uv` 作为 Python 包管理工具，支持 workspace 模式：

```bash
# 在根目录安装所有子项目依赖
uv sync

# 为特定子项目安装依赖
cd llm_sentry_monitor
uv sync

# 运行子项目
cd llm_sentry_monitor
uv run python main.py
```

### 子项目配置

每个子项目都有独立的 `pyproject.toml`，定义：
- 项目名称和版本
- 依赖列表
- 构建配置

根目录的 `pyproject.toml` 定义了 workspace 成员。

## 📦 添加新子项目

1. 在根目录创建新项目目录
2. 在新目录中创建 `pyproject.toml`
3. 在根目录 `pyproject.toml` 的 `[tool.uv.workspace]` 中添加新成员
4. 更新 `.workspace.json`（可选）

示例：

```toml
# 根目录 pyproject.toml
[tool.uv.workspace]
members = [
    "geo_db",
    "llm_sentry_monitor",
    "new_package",  # 新增
]
```

## 🚀 统一构建指令

使用根目录的 `Makefile` 可以统一管理所有子项目：

### 环境管理
```bash
make setup      # 安装所有依赖 (Python & Playwright)
make install    # 创建虚拟环境并安装依赖
make sync       # 同步依赖（检查并安装缺失的库）
```

### 数据库管理
```bash
make db-up      # 启动 PostgreSQL 数据库容器
make db-down    # 停止 PostgreSQL 数据库容器
make db-reset   # 重置数据库 (删除旧数据并重建)
make db-upgrade # 升级数据库到最新版本 (v1.0 -> v2.1)
make db-logs    # 查看数据库日志
```

### 任务执行
```bash
make run        # 执行 GEO 监测任务（配置文件模式）
make dev        # 启动 API 开发服务器（API 模式）
```

### 数据分析
```bash
make stats      # 生成基础深度洞察报告（简单版）
make stats-full # 生成完整深度洞察报告（包含所有分析维度）
```

### 其他
```bash
make status     # 查看服务状态
make clean      # 停止数据库并清理临时文件
make help       # 查看所有指令
```

## 📝 最佳实践

1. **独立版本管理**：每个子项目可以有自己的版本号
2. **共享代码**：公共工具和脚本放在 `scripts/` 目录
3. **统一依赖**：共同依赖可以在根目录管理
4. **独立文档**：每个子项目维护自己的 README.md
5. **数据库迁移**：使用 `migrations/` 目录管理数据库版本升级
6. **环境隔离**：使用 `.venv` 虚拟环境隔离依赖
7. **配置管理**：敏感配置使用 `.env` 文件（已加入 .gitignore）

## 🔄 数据库迁移系统

项目使用版本化的数据库迁移系统：

### 版本历史
- **v1.0**: 初始版本（`init.sql`）
- **v2.0**: 性能优化版本（`init_v2.sql` + `001_upgrade_to_v2.sql`）
  - 添加索引优化查询性能
  - 添加唯一约束防止重复数据
  - 添加级联删除
  - 扩展元数据字段
- **v2.1**: 任务管理系统（`002_add_task_jobs.sql`）
  - 新增 `task_jobs` 表支持异步任务管理
  - 支持任务状态追踪

### 迁移流程
```bash
# 自动升级到最新版本
make db-upgrade

# 或手动执行迁移脚本
cd geo_db
./upgrade_db.sh
```

## 🎯 运行模式

项目支持两种运行模式：

### 1. 配置文件模式（传统模式）
```bash
# 编辑 llm_sentry_monitor/config.yaml 配置任务
# 然后运行
make run
```

### 2. API 模式（推荐）
```bash
# 启动 API 服务器
make dev

# 通过 HTTP 请求提交任务
curl -X POST "http://localhost:8000/mock" \
  -H "Content-Type: application/json" \
  -d '{"keywords": ["关键词"], "platforms": ["deepseek"]}'
```

API 模式的优势：
- 支持异步任务执行
- 支持任务状态查询
- 支持自定义配置
- 提供 Swagger 文档（http://localhost:8000/docs）

## 📊 子项目详细说明

### geo_db - 数据库服务模块

**职责**：提供 PostgreSQL 数据库服务，负责数据持久化

**核心功能**：
- PostgreSQL 15 容器化部署
- 数据库表结构管理
- 数据库迁移系统
- 数据备份与恢复

**主要文件**：
- `docker-compose.yml`: Docker 容器配置
- `init_v2.sql`: 最新版本数据库初始化脚本
- `migrations/`: 数据库迁移脚本目录
- `upgrade_db.sh`: 自动化升级脚本

### llm_sentry_monitor - 监控抓取服务

**职责**：Web 自动化与数据采集，解析 AI 回答并存储

**核心功能**：
- 多平台适配（DeepSeek、豆包等）
- Web 自动化（Playwright）
- 引用解析与域名提取
- 数据统计分析
- REST API 服务

**主要模块**：
- `providers/`: 平台适配器（实现 BaseProvider 接口）
- `core/`: 核心业务逻辑
  - `db.py`: 数据库操作封装
  - `parser.py`: URL 解析与域名分类
  - `task_executor.py`: 异步任务执行
- `api.py`: FastAPI 应用
- `stats.py` / `stats_full.py`: 数据分析报告

## 🔗 相关文档

- [uv workspace 文档](https://github.com/astral-sh/uv/blob/main/docs/workspace.md)
- [Python Packaging User Guide](https://packaging.python.org/)
- [FastAPI 文档](https://fastapi.tiangolo.com/)
- [Playwright 文档](https://playwright.dev/python/)
- [PostgreSQL 文档](https://www.postgresql.org/docs/)

