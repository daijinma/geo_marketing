# Monorepo 架构说明

本仓库采用 **Monorepo** 架构，将多个相关项目组织在同一个代码仓库中，便于统一管理和代码共享。

## 📁 项目结构

```
GEO/
├── geo_db/                    # 子项目 1: 数据库服务
│   ├── pyproject.toml         # 项目配置
│   ├── docker-compose.yml     # Docker 编排
│   ├── init.sql               # 数据库初始化
│   └── README.md              # 项目文档
│
├── llm_sentry_monitor/        # 子项目 2: 监控服务
│   ├── pyproject.toml         # 项目配置
│   ├── main.py                # 入口文件
│   ├── providers/             # 提供者模块
│   ├── core/                  # 核心逻辑
│   └── README.md              # 项目文档
│
├── scripts/                   # 共享脚本
├── pyproject.toml             # 根工作区配置
├── .workspace.json            # 工作区元数据
├── Makefile                   # 统一构建指令
└── README.md                  # 根文档
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

```bash
make setup      # 初始化所有子项目
make db-up      # 启动数据库服务
make run        # 运行监控服务
make help       # 查看所有指令
```

## 📝 最佳实践

1. **独立版本管理**：每个子项目可以有自己的版本号
2. **共享代码**：公共工具和脚本放在 `scripts/` 目录
3. **统一依赖**：共同依赖可以在根目录管理
4. **独立文档**：每个子项目维护自己的 README.md

## 🔗 相关文档

- [uv workspace 文档](https://github.com/astral-sh/uv/blob/main/docs/workspace.md)
- [Python Packaging User Guide](https://packaging.python.org/)

