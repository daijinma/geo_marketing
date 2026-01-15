# 数据库升级说明

## 升级脚本说明

### 完整升级脚本（推荐）
- **文件**: `000_complete_upgrade_to_v3.1.sql`
- **用途**: 从 v1.0 一次性升级到 v3.1，整合了所有迁移步骤
- **适用场景**: 
  - 新部署的数据库
  - 需要一次性完成所有升级的数据库
  - 从 v1.0 直接升级到最新版本

### 分步升级脚本
如果需要逐步升级，可以按顺序执行：
1. `001_upgrade_to_v2.sql` - v1.0 → v2.0
2. `002_add_task_jobs.sql` - v2.0 → v2.1
3. `004_add_query_count_to_task_jobs.sql` - v2.1 → v2.2
4. `003_add_task_relations.sql` - v2.2 → v3.1

## 使用方法

### 方法1：使用完整升级脚本（推荐）
```bash
cd geo_db
docker exec -i <容器名> psql -U geo_admin -d geo_monitor < migrations/000_complete_upgrade_to_v3.1.sql
```

### 方法2：使用升级脚本
```bash
cd geo_db
bash upgrade_db.sh
```

### 方法3：在数据库客户端中执行
直接打开 `000_complete_upgrade_to_v3.1.sql` 文件，在数据库客户端中执行。

## 升级内容

### v2.0 升级
- 增强 `search_records` 表（添加 prompt、response_time_ms、search_status 等字段）
- 增强 `search_queries` 表（添加 query_order 字段）
- 增强 `citations` 表（添加唯一约束、级联删除）
- 创建 `domain_stats` 表（域名统计）
- 创建触发器函数和触发器
- 创建所有必要的索引

### v2.1 升级
- 创建 `task_jobs` 表（任务管理）
- 创建 `task_query` 表（任务查询关联）
- 创建 `executor_sub_query_log` 表（子查询日志）

### v2.2 升级
- 为 `task_jobs` 表添加 `query_count` 字段

### v3.1 升级
- 为 `search_records` 表添加 `task_id` 和 `task_query_id` 字段
- 为 `executor_sub_query_log` 表添加 `record_id` 和 `citation_id` 字段
- 创建所有关联索引

## 注意事项

1. **备份数据**: 执行升级前请先备份数据库
2. **版本检查**: 脚本使用 `IF NOT EXISTS` 和 `ON CONFLICT` 确保可以安全重复执行
3. **事务保护**: 所有升级都在事务中执行，失败会自动回滚
4. **向后兼容**: 所有新字段都允许 NULL，旧数据不受影响

## 验证升级结果

执行完成后，可以运行以下 SQL 验证：

```sql
-- 查看版本记录
SELECT version, applied_at, description 
FROM schema_version 
ORDER BY applied_at DESC;

-- 检查表结构
\d search_records
\d task_jobs
\d executor_sub_query_log

-- 检查索引
\di
```

