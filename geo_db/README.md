# LLM Sentry Database Service

本仓库是 `LLM Sentry` 项目的数据库服务模块，采用独立容器化部署。

## 1. 目录结构
```text
geo_db/
├── docker-compose.yml   # 容器编排配置
├── init.sql             # 数据库初始化脚本 (表结构定义)
└── README.md            # 本说明文件
```

## 2. 快速启动
确保本地已安装 Docker 和 Docker Compose。

```bash
cd geo_db
docker-compose up -d
```

## 3. 数据库配置
- **类型**: PostgreSQL 15
- **端口**: 5432
- **用户名**: `geo_admin`
- **密码**: `geo_password123`
- **数据库名**: `geo_monitor`

## 4. 数据持久化
数据存储在本地目录 `./postgres_data` 中，即使容器销毁，数据也会保留。

## 5. 数据库迁移

数据库支持版本化迁移，迁移脚本位于 `migrations/` 目录：

- `004_add_query_count_to_task_jobs.sql` - v2.2 版本更新
  - 为 `task_jobs` 表添加 `query_count` 字段（查询次数/执行轮数）
  - 默认值为 1，支持对同一查询条件执行多轮搜索

执行迁移：
```bash
cd geo_db
./upgrade_db.sh
```

## 6. 核心表结构

### task_jobs 表
- `id`: 任务ID
- `keywords`: 关键词列表（JSONB）
- `platforms`: 平台列表（JSONB）
- `query_count`: 查询次数（执行轮数），默认 1
- `status`: 任务状态（pending, done）
- `result_data`: 任务结果数据（JSONB）
