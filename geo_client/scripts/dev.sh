#!/bin/bash

# 优化的开发启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TAURI_DIR="$PROJECT_DIR/src-tauri"

cd "$PROJECT_DIR"

# 检查 Rust 依赖是否已构建
check_rust_deps() {
    if [ ! -d "$TAURI_DIR/target" ] || [ -z "$(ls -A "$TAURI_DIR/target" 2>/dev/null)" ]; then
        return 1
    fi
    
    # 检查是否有编译产物
    if [ -d "$TAURI_DIR/target/debug" ] || [ -d "$TAURI_DIR/target/release" ]; then
        return 0
    fi
    
    return 1
}

# 检查 Cargo.lock 是否与 Cargo.toml 同步
check_cargo_sync() {
    if [ ! -f "$TAURI_DIR/Cargo.lock" ]; then
        return 1
    fi
    
    # 检查 Cargo.lock 是否比 Cargo.toml 新（简单检查）
    if [ "$TAURI_DIR/Cargo.lock" -nt "$TAURI_DIR/Cargo.toml" ]; then
        return 0
    fi
    
    # 如果 Cargo.toml 更新了，可能需要重新构建
    return 1
}

echo "🔍 检查依赖状态..."

# 检查 Rust 依赖
if check_rust_deps && check_cargo_sync; then
    echo "✅ Rust 依赖已构建，跳过依赖安装"
    echo "💡 提示: 如需强制重新构建，请运行: cargo clean"
else
    echo "📦 Rust 依赖需要构建，将在开发过程中自动处理"
fi

# 检查 Node.js 依赖
if [ -d "node_modules" ] && [ -n "$(ls -A node_modules 2>/dev/null)" ]; then
    echo "✅ Node.js 依赖已安装"
else
    echo "📦 安装 Node.js 依赖..."
    # 检测包管理器
    if command -v pnpm &> /dev/null; then
        pnpm install
    elif command -v npm &> /dev/null; then
        npm install
    else
        echo "❌ 未找到包管理器（pnpm 或 npm）"
        exit 1
    fi
fi

echo ""
echo "🚀 启动开发服务器..."

# 运行 tauri dev
exec npm run tauri:dev
