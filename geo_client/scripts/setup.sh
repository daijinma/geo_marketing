#!/bin/bash

# 项目初始化脚本

echo "🚀 Geo Client 项目初始化..."

# 检测网络连接性
check_network() {
    local url=$1
    local timeout=${2:-5}
    
    if curl --connect-timeout "$timeout" --silent --head --fail "$url" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 检查命令是否存在并获取版本
check_command() {
    local cmd=$1
    local min_version=$2
    
    if command -v "$cmd" &> /dev/null; then
        local version=$($cmd --version 2>/dev/null | head -n1)
        echo "$version"
        return 0
    else
        return 1
    fi
}

# 检查版本是否符合要求（简单版本号比较）
check_version() {
    local current=$1
    local required=$2
    
    # 提取主版本号进行比较
    local current_major=$(echo "$current" | grep -oE '[0-9]+' | head -n1)
    local required_major=$(echo "$required" | grep -oE '[0-9]+' | head -n1)
    
    if [ -n "$current_major" ] && [ -n "$required_major" ]; then
        if [ "$current_major" -ge "$required_major" ]; then
            return 0
        fi
    fi
    return 1
}

# 安装 Rust（支持镜像源）
install_rust() {
    echo "📦 正在安装 Rust..."
    
    # 检测官方源是否可用
    local official_url="https://static.rust-lang.org"
    local use_mirror=false
    
    if ! check_network "$official_url"; then
        echo "⚠️  无法连接到 Rust 官方源，尝试使用国内镜像..."
        use_mirror=true
    fi
    
    # 设置镜像源环境变量（rustup 安装脚本会读取这些变量）
    if [ "$use_mirror" = true ]; then
        # 使用清华大学镜像源
        export RUSTUP_DIST_SERVER="https://mirrors.tuna.tsinghua.edu.cn/rustup"
        export RUSTUP_UPDATE_ROOT="https://mirrors.tuna.tsinghua.edu.cn/rustup/rustup"
        echo "📡 使用清华大学镜像源: $RUSTUP_DIST_SERVER"
    fi
    
    # 执行安装（安装脚本会自动使用环境变量中的镜像源）
    local install_script_url="https://sh.rustup.rs"
    if curl --proto '=https' --tlsv1.2 -sSf "$install_script_url" | sh -s -- -y; then
        echo "✅ Rust 安装成功！"
        # 加载 Rust 环境
        if [ -f "$HOME/.cargo/env" ]; then
            source "$HOME/.cargo/env"
        fi
        return 0
    else
        echo "❌ Rust 安装失败"
        if [ "$use_mirror" = true ]; then
            echo "💡 提示: 如果镜像源也无法访问，请检查网络连接或手动设置代理"
            echo "   也可以尝试其他镜像源，如中科大:"
            echo "   export RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static"
            echo "   export RUSTUP_UPDATE_ROOT=https://mirrors.ustc.edu.cn/rust-static/rustup"
        fi
        return 1
    fi
}

# 检查并安装 Node.js
echo "🔍 检查 Node.js..."
if node_version=$(check_command node); then
    echo "✅ Node.js 已安装: $node_version"
    # 检查版本是否符合要求（Node.js 18+）
    if ! check_version "$node_version" "18"; then
        echo "⚠️  Node.js 版本过低，需要 18+，当前: $node_version"
        echo "   请手动升级 Node.js: https://nodejs.org/"
    fi
else
    echo "❌ 未找到 Node.js"
    echo "   请先安装 Node.js 18+: https://nodejs.org/"
    echo "   或使用包管理器安装:"
    echo "   macOS: brew install node@18"
    echo "   Linux: 使用 nvm 或系统包管理器"
    exit 1
fi

# 检查并安装 Rust
echo "🔍 检查 Rust..."
if rust_version=$(check_command rustc); then
    echo "✅ Rust 已安装: $rust_version"
    # 检查版本是否符合要求（Rust 1.70+）
    if ! check_version "$rust_version" "1.70"; then
        echo "⚠️  Rust 版本可能过低，建议 1.70+，当前: $rust_version"
        echo "   可以运行: rustup update"
    fi
else
    echo "⚠️  未找到 Rust，开始自动安装..."
    if ! install_rust; then
        echo "❌ Rust 安装失败，请手动安装:"
        echo "   官方源: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
        echo "   或使用镜像: export RUSTUP_DIST_SERVER=https://mirrors.tuna.tsinghua.edu.cn/rustup && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
        exit 1
    fi
fi

# 检测并选择包管理器（优先 pnpm，降级 npm）
detect_package_manager() {
    if command -v pnpm &> /dev/null; then
        echo "pnpm"
        return 0
    elif command -v npm &> /dev/null; then
        echo "npm"
        return 0
    else
        return 1
    fi
}

# 安装 Node.js 依赖
echo "🔍 检测包管理器..."
if package_manager=$(detect_package_manager); then
    if [ "$package_manager" = "pnpm" ]; then
        pnpm_version=$(check_command pnpm)
        echo "✅ 使用 pnpm: $pnpm_version"
        echo "📦 安装 Node.js 依赖..."
        pnpm install
    else
        npm_version=$(check_command npm)
        echo "✅ 使用 npm: $npm_version"
        echo "📦 安装 Node.js 依赖..."
        npm install
    fi
else
    echo "❌ 未找到包管理器（pnpm 或 npm）"
    echo "   推荐安装 pnpm: npm install -g pnpm"
    echo "   或使用 npm（通常随 Node.js 一起安装）"
    exit 1
fi

# 检查并安装 Tauri CLI
echo "🔍 检查 Tauri CLI..."
if tauri_version=$(check_command cargo-tauri); then
    echo "✅ Tauri CLI 已安装: $tauri_version"
else
    echo "📦 安装 Tauri CLI..."
    if cargo install tauri-cli; then
        echo "✅ Tauri CLI 安装成功"
    else
        echo "⚠️  Tauri CLI 安装失败，可能需要先配置 Rust 环境"
        echo "   可以稍后手动安装: cargo install tauri-cli"
    fi
fi

# 创建环境变量文件（如果不存在）
if [ ! -f .env.development ]; then
    echo "📝 创建 .env.development..."
    cat > .env.development << EOF
VITE_API_BASE_URL=http://127.0.0.1:8000
VITE_APP_ENV=development
EOF
fi

if [ ! -f .env.production ]; then
    echo "📝 创建 .env.production..."
    cat > .env.production << EOF
VITE_API_BASE_URL=https://api.example.com
VITE_APP_ENV=production
EOF
fi

echo "✅ 项目初始化完成！"
echo ""
echo "📖 使用方法："
echo "  开发模式: npm run tauri:dev"
echo "  构建: npm run tauri:build"
echo "  类型检查: npm run type-check"
