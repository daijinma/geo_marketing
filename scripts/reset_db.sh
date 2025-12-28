#!/bin/bash
# 重新初始化数据库（删除旧数据并重建）
echo "⚠️  警告：此操作将删除所有现有数据！"
read -p "确认继续？(y/N): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "已取消操作。"
    exit 0
fi

echo "正在停止数据库容器..."
cd "$(dirname "$0")/../geo_db" && docker-compose down

echo "正在删除旧数据..."
rm -rf postgres_data

echo "正在重新启动数据库..."
docker-compose up -d

echo "等待数据库初始化完成..."
sleep 5

echo "✅ 数据库已重新初始化完成！"
