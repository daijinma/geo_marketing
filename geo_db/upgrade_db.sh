#!/bin/bash
# 文件：geo_db/upgrade_db.sh
# 用途：升级数据库到最新版本

set -e

echo "🔄 开始升级数据库..."

# 进入数据库目录
cd "$(dirname "$0")"

# 检查 PostgreSQL 是否运行
if ! docker ps | grep -q geo_db; then
    echo "❌ 数据库容器未运行，请先执行: make db-up"
    exit 1
fi

# 获取容器名称（可能是 geo_db-postgres-1 或 geo_db_postgres_1）
CONTAINER_NAME=$(docker ps --filter "name=geo_db" --format "{{.Names}}" | head -1)

if [ -z "$CONTAINER_NAME" ]; then
    echo "❌ 找不到数据库容器"
    exit 1
fi

# 执行所有迁移脚本（按顺序）
echo "📝 执行迁移脚本..."

# 检查并执行 v2.0 迁移
if [ -f "migrations/001_upgrade_to_v2.sql" ]; then
    echo "  → 执行 v2.0 迁移..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/001_upgrade_to_v2.sql
fi

# 检查并执行 v2.1 迁移
if [ -f "migrations/002_add_task_jobs.sql" ]; then
    echo "  → 执行 v2.1 迁移（添加 task_jobs 表）..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/002_add_task_jobs.sql
fi

# 检查并执行 v2.2 迁移
if [ -f "migrations/004_add_query_count_to_task_jobs.sql" ]; then
    echo "  → 执行 v2.2 迁移（添加 query_count 字段）..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/004_add_query_count_to_task_jobs.sql
fi

# 检查并执行 v3.1 迁移
if [ -f "migrations/003_add_task_relations.sql" ]; then
    echo "  → 执行 v3.1 迁移（添加任务关联关系）..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/003_add_task_relations.sql
fi

# 检查并执行 GEO 内容优化器迁移
if [ -f "migrations/005_add_geo_content_optimizer_tables.sql" ]; then
    echo "  → 执行 GEO 内容优化器迁移（添加 topic_maps, fact_sources 等表）..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/005_add_geo_content_optimizer_tables.sql
fi

# 检查并执行用户认证迁移
if [ -f "migrations/006_add_users_and_auth_tables.sql" ]; then
    echo "  → 执行用户认证迁移（添加 users 和 auth_tokens 表）..."
    docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/006_add_users_and_auth_tables.sql
fi

echo "✅ 数据库升级完成！"
echo ""
echo "📊 当前数据库版本："
docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor -c "SELECT * FROM schema_version ORDER BY applied_at DESC LIMIT 5;"
