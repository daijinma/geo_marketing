#!/bin/bash
# 重新初始化数据库（删除旧数据并重建）
echo "⚠️  警告：此操作将删除所有现有数据！"
read -p "确认继续？(y/N): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "已取消操作。"
    exit 0
fi

# 切换到 geo_db 目录
GEO_DB_DIR="$(dirname "$0")/../geo_db"
cd "$GEO_DB_DIR"

echo "正在停止数据库容器..."
docker-compose down

echo "正在删除旧数据..."
rm -rf postgres_data

echo "正在重新启动数据库..."
docker-compose up -d

echo "等待数据库初始化完成..."
sleep 5

echo "🔄 执行数据库升级（应用所有迁移脚本）..."
# 执行升级脚本
chmod +x upgrade_db.sh && ./upgrade_db.sh

echo "✅ 数据库已重新初始化并升级完成！"
