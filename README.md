# LLM Sentry: GEO (生成式引擎优化) 实时监测平台

> **Monorepo 多项目仓库** - 本仓库包含 LLM Sentry 项目的所有子模块

`LLM Sentry` 是一个专为 **GEO (Generative Engine Optimization)** 设计的自动化监测与分析平台。它能够实时追踪品牌或内容在主流 AI 搜索引擎（如 DeepSeek、豆包）回答中的曝光情况，并量化分析其引用来源。

## 📦 项目结构 (Monorepo)

本仓库采用 monorepo 架构，包含以下独立子项目：

```text
GEO/ (Root Monorepo)
├── geo_db/                  # 【数据库服务】PostgreSQL 容器化部署
│   ├── pyproject.toml       # 项目配置
│   ├── docker-compose.yml
│   ├── init.sql
│   └── README.md
│
├── llm_sentry_monitor/      # 【监控抓取服务】Web 自动化与数据采集
│   ├── pyproject.toml       # 项目配置
│   ├── main.py
│   ├── providers/           # 多模型适配器 (DeepSeek, 豆包)
│   ├── core/                # 域名清洗与引用解析引擎
│   └── README.md
│
├── scripts/                 # 共享脚本工具
│   ├── curl_examples.sh    # API 测试脚本（cURL 示例）
├── pyproject.toml           # 工作区配置
├── .workspace.json          # 工作区元数据
├── Makefile                 # 统一构建与运行指令
├── README.md                # 本文件
└── MONOREPO.md              # Monorepo 架构说明
```

### 子项目说明

- **`geo_db/`**: PostgreSQL 数据库服务模块，提供数据持久化能力
- **`llm_sentry_monitor/`**: 监控抓取服务，负责模拟用户行为并解析 AI 回答

> 📖 关于 Monorepo 架构的详细说明，请参阅 [MONOREPO.md](./MONOREPO.md)

## 1. 核心价值
在 AI 驱动搜索的时代，传统的 SEO 正在向 GEO 演进。本项目旨在解决以下核心痛点：
- **曝光监测**：实时了解 AI 在回答特定行业问题时，是否提及了你的品牌。
- **引用溯源**：精准提取 AI 回答中的参考链接，分析哪些站点（知乎、官网、自媒体）被 AI 采纳为 RAG 上下文。
- **声量占比 (SoV)**：量化统计不同品牌在 AI 搜索引擎中的收录权重与占比。

## 2. 技术架构

项目采用解耦的服务化架构，分为数据持久化层和业务执行层：

```text
GEO/ (Root)
├── geo_db/                  # 【数据库服务】
│   ├── docker-compose.yml   # PostgreSQL 15 容器化部署
│   └── init.sql             # 自动化表结构初始化 (Records & Citations)
└── llm_sentry_monitor/      # 【监控抓取服务】
    ├── main.py              # 任务调度与业务逻辑入口
    ├── providers/           # 多模型适配器 (基于 Playwright 的网页自动化)
    └── core/                # 域名清洗与引用解析引擎
```

### 关键技术栈
- **自动化引擎**：Playwright (模拟真实用户行为，绕过 API 限制，获取最真实的联网搜索结果)。
- **数据存储**：PostgreSQL (利用 JSONB 存储复杂引用元数据)。
- **解析算法**：基于 `tldextract` 的顶级域名清洗与 SoV 统计模型。

## 3. 核心流程逻辑
1. **模拟提问**：通过设计 Prompt 矩阵（对比、建议、查询），模拟真实用户向 DeepSeek/豆包提问。
2. **联网搜索**：驱动浏览器自动开启“联网模式”，获取 AI 实时检索后的回答。
3. **引用提取**：解析 Markdown 文本及页面 DOM，提取所有 `[1][2]` 形式的参考资料链接。
4. **量化分析**：将 URL 还原为域名，存入数据库并计算各站点的收录占比。

## 4. 快速开始指南 (极简模式)

为了简化操作，我们提供了 `Makefile` 指令：

### 1. 环境初始化 (使用 uv 管理虚拟环境)
```bash
make setup
```
该指令会自动创建 `.venv` 虚拟环境并安装所有依赖。

### 2. 启动数据库
```bash
make db
```

### 3. 执行监测任务
```bash
make run
```

### 4. 帮助与清理
```bash
make help   # 查看所有指令
make clean  # 停止数据库并清理缓存
```

> **提示**：首次运行 `make run` 时建议在非无头模式下手动完成登录。

## 5. API 服务（新增）

项目现已支持 REST API 接口，可以通过 HTTP 请求提交任务并查询状态。

### 启动 API 服务器
```bash
make dev
```

### API 接口

#### 创建任务 (POST /mock)
```bash
curl -X POST "http://localhost:8000/mock" \
  -H "Content-Type: application/json" \
  -d '{
    "keywords": ["土巴兔装修靠谱嘛", "装修公司推荐"],
    "platforms": ["deepseek"]
  }'
```

#### 查询任务状态 (GET /status)
```bash
curl -X GET "http://localhost:8000/status?id=1"
```

更多 API 示例请参考：`scripts/curl_examples.sh`

### API 文档
- Swagger UI: http://localhost:8000/docs
- ReDoc: http://localhost:8000/redoc

## 6. 更新日志

### 2024-01-XX - API 化改造与任务管理系统

#### 新增功能
- ✅ **REST API 接口**：提供 `/mock` 和 `/status` 接口，支持通过 HTTP 请求管理任务
- ✅ **异步任务管理**：任务在后台异步执行，支持状态查询
- ✅ **数据库迁移 v2.1**：新增 `task_jobs` 表，用于存储任务状态和结果
- ✅ **FastAPI 集成**：使用 FastAPI 构建现代化 API 服务
- ✅ **Makefile 增强**：新增 `make dev` 命令启动 API 服务器
- ✅ **cURL 测试脚本**：提供完整的 API 测试示例

#### 技术改进
- 将基于配置文件的批量任务执行改为基于 API 的异步任务管理
- 支持自定义关键词和平台配置
- 任务状态实时追踪（none, pending, done）
- 保留原有 `run_tasks()` 函数，保持向后兼容

#### 文件变更
- **新增文件**：
  - `llm_sentry_monitor/api.py` - FastAPI 应用，提供 REST API 接口
  - `llm_sentry_monitor/core/task_executor.py` - 任务执行器，封装异步任务逻辑
  - `geo_db/migrations/002_add_task_jobs.sql` - 数据库迁移脚本 v2.1
  - `scripts/curl_examples.sh` - API 测试脚本（cURL 示例）
  - `llm_sentry_monitor/CURL_EXAMPLES.md` - API 使用文档
- **更新文件**：
  - `Makefile` - 添加 `install`, `sync`, `dev` 命令
  - `llm_sentry_monitor/pyproject.toml` - 添加 FastAPI 和 uvicorn 依赖
  - `llm_sentry_monitor/main.py` - 添加 API 服务器启动选项，修复 platform_name_map
  - `geo_db/upgrade_db.sh` - 支持执行多个迁移脚本
  - `README.md` - 添加 API 服务说明和更新日志

#### 使用方式
```bash
# 1. 启动 API 服务器
make dev

# 2. 运行 API 测试脚本
./scripts/curl_examples.sh

# 3. 或使用 curl 直接调用
curl -X POST "http://localhost:8000/mock" \
  -H "Content-Type: application/json" \
  -d '{"keywords": ["关键词"], "platforms": ["deepseek"]}'
```

---
*LLM Sentry - 助力品牌在生成式搜索时代赢得先机*
