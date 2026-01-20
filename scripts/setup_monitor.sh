#!/bin/bash
# 使用 uv 安装后端服务依赖并创建虚拟环境
set -e

echo "正在初始化环境..."

# 1. 检查并创建 .env 文件
if [ ! -f "geo_server/.env" ]; then
    echo "未发现 .env 文件，正在从 .env.example 复制..."
    cp geo_server/.env.example geo_server/.env
fi

cd geo_server

# 2. 创建虚拟环境 (如果不存在)
if [ ! -d ".venv" ]; then
    echo "正在创建虚拟环境..."
    uv venv
fi

# 3. 安装依赖
echo "正在安装依赖..."
uv pip install -r requirements.txt

# 4. 安装 Playwright 浏览器
echo "正在安装 Playwright 浏览器..."
uv run playwright install chromium

echo "✅ 依赖安装完成。虚拟环境位于 geo_server/.venv"
echo "💡 请根据需要修改 geo_server/.env 中的配置。"
