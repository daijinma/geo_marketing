#!/bin/bash
# 启动数据库服务
set -e

echo "正在启动数据库服务..."

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行，请先启动 Docker。"
    exit 1
fi

cd geo_db && docker-compose up -d

echo "✅ 数据库服务已在后台运行。"
echo "📊 端口: 5432"
