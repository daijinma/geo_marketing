# GEO 品牌曝光监测系统 (LLM Sentry) - 核心意图与需求文档

## 1. 项目愿景 (Core Vision)
本项目旨在构建一个自动化的 **GEO (Generative Engine Optimization)** 监测平台。通过模拟真实用户与 AI 搜索引擎（如 DeepSeek）的交互，深度分析 AI 在生成回答时的信息来源、搜索逻辑及品牌曝光情况，为品牌在 AI 时代的搜索优化提供数据支撑。

## 2. 核心需求 (Core Requirements)

### 2.1 数据采集 (Data Acquisition)
- **策略**: 采用 Web 自动化（Playwright）而非 API，以获取最真实的搜索结果并绕过 API 限制。
- **技术点**: 
    - 拦截并解析 **SSE (Server-Sent Events)** 数据流，直接从 API 响应中提取结构化数据。
    - 捕获 **AI 拓展搜索词 (Search Queries)**：了解 AI 如何拆解和理解用户的原始提问。
    - 捕获 **参考来源 (Citations)**：记录 AI 实际点击和参考的网页 URL、标题、摘要及站点名称。

### 2.2 任务管理 (Task Management)
- **配置化**: 通过 `config.yaml` 批量管理监控关键词。
- **自动化**: 支持仅输入关键词，由系统自动生成标准化的分析 Prompt。
- **稳定性**: 具备智能状态检测（如自动判断“联网搜索”是否开启），避免重复操作。

### 2.3 数据存储与分析 (Storage & Analytics)
- **持久化**: 使用 PostgreSQL 存储所有搜索记录、拓展词和引用链接。
- **深度洞察**:
    - **核心信任源分析**: 识别在多个关键词下被 AI 频繁引用的“行业权威站点”。
    - **搜索意图分词**: 对 AI 拓展词进行分词统计，挖掘 AI 关注的核心概念和潜在风险点。
    - **SoV (Share of Voice) 统计**: 量化品牌及其竞争对手在 AI 引用中的占比。

## 3. 技术栈 (Technical Stack)
- **语言**: Python 3.12
- **自动化**: Playwright (Chromium)
- **数据库**: PostgreSQL 15 (Docker 容器化)
- **环境管理**: uv
- **分析工具**: tabulate, re, collections

## 4. 战略价值 (Strategic Value)
- **内容分发指引**: 明确哪些网站是 AI 的“信任源”，指导品牌公关和内容投放。
- **搜索意图对齐**: 通过 AI 拓展词了解 AI 的理解偏好，优化品牌内容的关键词布局。
- **竞争态势监控**: 实时掌握竞争对手在 AI 搜索结果中的曝光强度。

---
*此文档作为项目核心意图的唯一事实来源，供后续开发及 LLM 上下文恢复使用。*
