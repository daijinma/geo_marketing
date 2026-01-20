#!/bin/bash

# 打包脚本

echo "🚀 开始打包端界GEO客户端..."

# 检查环境
if ! command -v npm &> /dev/null; then
    echo "❌ 未找到 npm，请先安装 Node.js"
    exit 1
fi

if ! command -v cargo &> /dev/null; then
    echo "❌ 未找到 cargo，请先安装 Rust"
    exit 1
fi

# 安装依赖（如果未安装）
if [ ! -d "node_modules" ]; then
    echo "📦 安装 Node.js 依赖..."
    npm install
fi

# 构建前端
echo "🔨 构建前端..."
npm run build

# 构建Tauri应用
echo "🔨 构建 Tauri 应用..."
npm run tauri:build

echo "✅ 打包完成！"
echo "📦 应用包位置: src-tauri/target/release/bundle/"
