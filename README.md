# LLM Sentry: GEO (生成式引擎优化) 实时监测平台

> **Monorepo 多项目仓库** - 本仓库包含 LLM Sentry 项目的所有子模块

`LLM Sentry` 是一个专为 **GEO (Generative Engine Optimization)** 设计的自动化监测与分析平台。它能够实时追踪品牌或内容在主流 AI 搜索引擎（如 DeepSeek、豆包）回答中的曝光情况，并量化分析其引用来源。

> 💻 **桌面客户端**: 项目提供 `geo_client2` 桌面应用（基于 Wails + Go + React），提供图形化界面进行任务管理、搜索监测、登录授权等功能，支持本地 SQLite 数据存储，无需部署后端服务即可使用。

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
├── geo_client2/             # 【桌面客户端】Wails + Go + React 桌面应用
│   ├── main.go              # Wails 入口
│   ├── backend/              # Go 后端（认证、任务、搜索、Provider）
│   ├── frontend/             # React 前端（React 18 + Vite + TypeScript）
│   ├── go.mod                # Go 依赖
│   └── README.md             # 客户端文档
│
├── scripts/                  # 共享脚本工具
├── pyproject.toml            # 工作区配置
├── .workspace.json           # 工作区元数据
├── Makefile                  # 统一构建与运行指令
├── README.md                 # 本文件
└── MONOREPO.md               # Monorepo 架构说明
```

### 子项目说明

- **`geo_db/`**: PostgreSQL 数据库服务模块，提供数据持久化能力
- **`llm_sentry_monitor/`**: 监控抓取服务，负责模拟用户行为并解析 AI 回答
- **`geo_client2/`**: 桌面客户端应用，基于 Wails v2 + Go + React 构建，提供图形化界面进行 GEO 监测任务管理

#### geo_client2 桌面客户端

`geo_client2` 是一个轻量级桌面应用，提供以下功能：

**核心功能**：
- ✅ **任务管理**：创建、执行、取消、重试监测任务
- ✅ **搜索监测**：支持 DeepSeek、豆包等平台的 AI 搜索监测
- ✅ **登录授权**：管理各平台的登录状态，支持持久化登录
- ✅ **本地存储**：使用 SQLite 本地数据库，无需外部数据库服务
- ✅ **设置管理**：配置无头模式、超时时间等运行参数
- ✅ **日志查看**：实时查看任务执行日志

**技术特点**：
- 🚀 **轻量级**：包体积约 10-20MB（相比 Electron 版本减少 85%+）
- ⚡ **高性能**：Go 后端 + 系统原生 WebView，启动速度快
- 🔒 **本地优先**：数据存储在本地，保护隐私
- 🎨 **现代 UI**：React + Tailwind + shadcn/ui，界面美观

**快速开始**：
```bash
cd geo_client2
# 安装依赖
cd frontend && pnpm install
cd .. && go mod tidy

# 开发模式
wails dev

# 构建应用
wails build
```

详细使用说明请参阅 [geo_client2/README.md](./geo_client2/README.md)

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
2. **联网搜索**：驱动浏览器自动开启"联网模式"，获取 AI 实时检索后的回答。
3. **多轮执行**：支持对同一查询条件执行多轮搜索（`query_count` 参数），用于测试结果稳定性或获取多次查询的平均数据。
4. **引用提取**：解析 Markdown 文本及页面 DOM，提取所有 `[1][2]` 形式的参考资料链接。
5. **量化分析**：将 URL 还原为域名，存入数据库并计算各站点的收录占比。

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

## 5. API 使用

项目提供 REST API 接口用于任务管理。启动服务后，可通过 API 创建监测任务：

### 创建任务接口

**POST** `/mock`

请求参数：
- `keywords`: 搜索关键词列表（必填）
- `platforms`: 平台列表，可选值：`["deepseek", "doubao", "bocha"]`（默认：`["deepseek"]`）
- `query_count`: 查询次数（执行轮数），默认 `1`。设置为大于 1 的值时，系统会对每个关键词-平台组合执行多轮搜索
- `settings`: 可选设置
  - `headless`: 是否无头模式（默认：`false`）
  - `timeout`: 超时时间（毫秒，默认：`60000`）
  - `delay_between_tasks`: 任务间延迟（秒，默认：`5`）

示例请求：
```json
{
  "keywords": ["AI搜索引擎", "生成式搜索"],
  "platforms": ["deepseek", "doubao"],
  "query_count": 3,
  "settings": {
    "headless": false,
    "delay_between_tasks": 5
  }
}
```

### 多轮执行功能

`query_count` 参数允许对同一查询条件执行多次搜索，适用于：
- **稳定性测试**：验证 AI 回答的一致性
- **数据采集**：获取多次查询的平均结果，减少单次查询的随机性
- **趋势分析**：观察不同时间点的搜索结果变化

执行逻辑：系统会按照 `query_count` 的值，对每个 `(关键词, 平台)` 组合循环执行指定轮数。

---
*LLM Sentry - 助力品牌在生成式搜索时代赢得先机*
