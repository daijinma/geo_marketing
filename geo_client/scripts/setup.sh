#!/bin/bash

# Geo Client 一键安装脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

echo "🚀 Geo Client 安装脚本"
echo "========================"
echo ""

# 检查 Node.js
echo "🔍 检查 Node.js..."
if ! command -v node &> /dev/null; then
    echo "❌ 未找到 Node.js，请先安装 Node.js 18+"
    echo "   下载地址: https://nodejs.org/"
    exit 1
fi

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "⚠️  Node.js 版本过低（当前: $(node -v)），建议 18+"
fi

echo "✅ Node.js 已安装: $(node -v)"
echo ""

# 检查包管理器
echo "🔍 检查包管理器..."
if command -v pnpm &> /dev/null; then
    PKG_MANAGER="pnpm"
    echo "✅ 使用 pnpm: $(pnpm -v)"
elif command -v npm &> /dev/null; then
    PKG_MANAGER="npm"
    echo "✅ 使用 npm: $(npm -v)"
else
    echo "❌ 未找到 npm 或 pnpm"
    exit 1
fi

echo ""

# 配置 Electron 镜像（可选）
if [ "$USE_CHINA_MIRROR" = "1" ] || [ "$USE_CHINA_MIRROR" = "true" ]; then
    echo "🌏 配置国内镜像源..."
    export ELECTRON_MIRROR=https://npmmirror.com/mirrors/electron/
    export PLAYWRIGHT_DOWNLOAD_HOST=https://npmmirror.com/mirrors/playwright/
    echo "✅ 已配置镜像源"
    echo ""
fi

# 安装依赖
echo "📦 安装 Node.js 依赖..."
if [ "$PKG_MANAGER" = "pnpm" ]; then
    pnpm install
else
    npm install
fi

echo "✅ 依赖安装完成"
echo ""

# 安装 Playwright 浏览器
echo "🌐 安装 Playwright 浏览器..."
if npx playwright install chromium; then
    echo "✅ Playwright 浏览器安装完成"
else
    echo "⚠️  Playwright 浏览器安装失败，可以稍后手动运行:"
    echo "   npx playwright install chromium"
fi

echo ""

# 编译 Electron 主进程代码
echo "🔨 编译 Electron 主进程代码..."
if [ "$PKG_MANAGER" = "pnpm" ]; then
    pnpm exec tsc -p electron
else
    npm exec tsc -p electron
fi

echo "✅ 编译完成"
echo ""

# 创建环境变量文件（如果不存在）
if [ ! -f ".env.development" ]; then
    echo "📝 创建 .env.development 文件..."
    cat > .env.development << EOF
VITE_API_BASE_URL=http://127.0.0.1:8000
VITE_APP_ENV=development
EOF
    echo "✅ .env.development 已创建"
fi

echo ""
echo "🎉 安装完成！"
echo ""
echo "快速开始："
echo "  开发模式: $PKG_MANAGER run electron:dev"
echo "  构建应用: $PKG_MANAGER run electron:build"
echo ""
echo "如果遇到问题，请查看 README.md 的故障排除部分"
echo ""
