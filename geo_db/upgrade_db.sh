#!/bin/bash
# 文件：geo_db/upgrade_db.sh
# 用途：升级数据库到 v2.0

set -e

echo "🔄 开始升级数据库到 v2.0..."

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

echo "📝 执行迁移脚本..."
docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor < migrations/001_upgrade_to_v2.sql

echo "✅ 数据库升级完成！"
echo ""
echo "📊 当前数据库版本："
docker exec -i "$CONTAINER_NAME" psql -U geo_admin -d geo_monitor -c "SELECT * FROM schema_version ORDER BY applied_at DESC LIMIT 1;"
