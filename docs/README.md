# GEO 实施文档中心

本目录包含 GEO（Generative Engine Optimization）实施相关的所有文档、模板和工具说明。

## 📁 文档结构

```
docs/
├── README.md                          # 本文档（文档索引）
├── GEO实施工程化方案.md               # 主方案文档（必读）
├── 数据库扩展方案.md                  # 数据库扩展方案
│
├── semantic_footprint/                # 语义足迹扩展文档
│   ├── README.md
│   ├── 话题映射模板.md
│   ├── 内容规划矩阵模板.csv
│   ├── Prompt映射模板.md
│   └── 装修行业话题地图.md
│
├── fact_density/                      # 事实密度提升文档
│   ├── README.md
│   ├── 数据源收集清单模板.yaml
│   ├── 装修行业数据收集清单.yaml
│   ├── 引用格式规范.md
│   └── 质量评分标准.md
│
└── structured_data/                   # 结构化数据文档
    ├── README.md
    ├── schema实施计划.yaml
    ├── templates/                     # Schema 模板
    │   ├── organization_schema.yaml
    │   ├── service_schema.yaml
    │   ├── faq_schema.yaml
    │   └── article_schema.yaml
    └── examples/                      # Schema 示例
        ├── organization_schema.json
        ├── service_schema.json
        └── faq_schema.json
```

---

## 📖 快速开始

### 1. 阅读主方案文档

**必读**：[GEO实施工程化方案.md](./GEO实施工程化方案.md)

这是整个 GEO 实施的核心文档，包含：
- 工作方向概览
- 核心策略实施方案
- 工程化实施步骤
- 技术架构与工具链
- 实施里程碑

### 2. 了解数据库扩展

**必读**：[数据库扩展方案.md](./数据库扩展方案.md)

了解数据库表结构扩展方案，包括：
- 新增表结构
- 视图和函数
- 数据迁移脚本

### 3. 开始实施

根据你的需求，参考相应的子文档：

- **语义足迹扩展**：参考 `semantic_footprint/` 目录
- **事实密度提升**：参考 `fact_density/` 目录
- **结构化数据实施**：参考 `structured_data/` 目录

---

## 🎯 核心策略

### 1. 扩大语义足迹 (Expanding Semantic Footprint)

**目标**：从单一关键词扩展到话题集群

**相关文档**：
- [语义足迹扩展 README](./semantic_footprint/README.md)
- [话题映射模板](./semantic_footprint/话题映射模板.md)
- [内容规划矩阵模板](./semantic_footprint/内容规划矩阵模板.csv)

### 2. 提高事实密度 (Increasing Fact-Density)

**目标**：增加统计数据、引用文献和独特见解

**相关文档**：
- [事实密度提升 README](./fact_density/README.md)
- [数据源收集清单模板](./fact_density/数据源收集清单模板.yaml)
- [引用格式规范](./fact_density/引用格式规范.md)
- [质量评分标准](./fact_density/质量评分标准.md)

### 3. 强化结构化数据 (Enhancing Structured Data)

**目标**：系统化实施 Schema.org 标记

**相关文档**：
- [结构化数据 README](./structured_data/README.md)
- [Schema 实施计划](./structured_data/schema实施计划.yaml)
- [Schema 模板](./structured_data/templates/)

---

## 📋 实施流程

### Phase 1: 基础设施搭建（2-3 周）

1. 阅读 [数据库扩展方案.md](./数据库扩展方案.md)
2. 执行数据库迁移脚本
3. 建立文档目录结构
4. 开发基础工具

### Phase 2: 内容优化实施（3-4 周）

1. 使用 [话题映射模板](./semantic_footprint/话题映射模板.md) 生成话题地图
2. 使用 [数据源收集清单模板](./fact_density/数据源收集清单模板.yaml) 收集数据源
3. 使用 [Schema 模板](./structured_data/templates/) 实施结构化数据
4. 实施内容增强

### Phase 3: 自动化流程完善（2-3 周）

1. 开发自动化工具
2. 建立质量评分系统
3. 实施批量验证

### Phase 4: 监控与反馈集成（1-2 周）

1. 与 `llm_sentry_monitor` 集成
2. 建立反馈循环
3. 生成监控报告

---

## 🔧 工具与模板

### 模板文件

- **话题映射**：`semantic_footprint/话题映射模板.md`
- **内容规划**：`semantic_footprint/内容规划矩阵模板.csv`
- **数据源收集**：`fact_density/数据源收集清单模板.yaml`
- **Schema 实施**：`structured_data/schema实施计划.yaml`

### 示例文件

- **装修行业话题地图**：`semantic_footprint/装修行业话题地图.md`
- **装修行业数据收集清单**：`fact_density/装修行业数据收集清单.yaml`
- **Schema 示例**：`structured_data/examples/`

---

## 📚 相关资源

- [GEO 原则文档](../why/GEO原则.md)
- [为什么增加博查](../why/为什么增加博查.md)
- [项目主 README](../README.md)

---

## 💡 使用建议

1. **先读主文档**：从 [GEO实施工程化方案.md](./GEO实施工程化方案.md) 开始
2. **使用模板**：根据模板文件开始工作，不要从零开始
3. **参考示例**：查看示例文件了解最佳实践
4. **持续更新**：根据实际情况更新文档和模板

---

**文档版本**：v1.0  
**最后更新**：2025-01-16  
**维护者**：GEO 项目组

