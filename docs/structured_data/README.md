# 结构化数据文档

本目录包含结构化数据（Schema.org）实施相关的文档、模板和示例。

## 目录结构

- `README.md` - 本文件
- `schema实施计划.yaml` - Schema 实施计划模板
- `templates/` - Schema 模板目录
  - `organization_schema.yaml` - Organization Schema 模板
  - `service_schema.yaml` - Service Schema 模板
  - `faq_schema.yaml` - FAQ Schema 模板
  - `article_schema.yaml` - Article Schema 模板
- `examples/` - Schema 示例目录
  - `organization_schema.json` - Organization Schema 示例（JSON-LD）
  - `service_schema.json` - Service Schema 示例（JSON-LD）
  - `faq_schema.json` - FAQ Schema 示例（JSON-LD）

## 使用说明

### 1. Schema 规划

1. 使用 `schema实施计划.yaml` 作为起点
2. 确定需要实施的页面和 Schema 类型
3. 规划实施优先级和时间

### 2. Schema 生成

1. 填写相应的 Schema 模板（YAML 格式）
2. 使用工具将 YAML 转换为 JSON-LD
3. 将 JSON-LD 嵌入到页面中

### 3. Schema 验证

1. 使用 Google 结构化数据测试工具验证
2. 修复验证错误
3. 记录验证结果

## 工作流程

```
Schema 规划
  ↓
填写 Schema 模板（YAML）
  ↓
转换为 JSON-LD
  ↓
嵌入到页面
  ↓
验证 Schema
  ↓
监控 Schema 效果
```

