#!/bin/bash

# 检查 Rust 依赖是否已构建

check_cargo_deps() {
    local tauri_dir="src-tauri"
    
    if [ ! -d "$tauri_dir" ]; then
        echo "❌ 未找到 src-tauri 目录"
        return 1
    fi
    
    cd "$tauri_dir" || return 1
    
    # 检查 Cargo.lock 是否存在
    if [ ! -f "Cargo.lock" ]; then
        echo "⚠️  Cargo.lock 不存在，需要构建依赖"
        cd ..
        return 1
    fi
    
    # 检查 target 目录是否存在且有内容
    if [ -d "target" ] && [ -n "$(ls -A target 2>/dev/null)" ]; then
        # 检查是否有编译产物（至少检查 debug 目录）
        if [ -d "target/debug" ] || [ -d "target/release" ]; then
            echo "✅ Rust 依赖已构建，跳过安装"
            cd ..
            return 0
        fi
    fi
    
    echo "⚠️  Rust 依赖未构建或需要更新"
    cd ..
    return 1
}

# 预构建依赖（如果未构建）
prebuild_deps() {
    local tauri_dir="src-tauri"
    
    if [ ! -d "$tauri_dir" ]; then
        return 1
    fi
    
    cd "$tauri_dir" || return 1
    
    echo "📦 预构建 Rust 依赖..."
    
    # 使用 cargo fetch 只下载依赖，不编译
    if cargo fetch --quiet 2>/dev/null; then
        echo "✅ 依赖已下载"
        cd ..
        return 0
    else
        echo "⚠️  依赖下载失败，将在运行时自动处理"
        cd ..
        return 1
    fi
}

# 主函数
main() {
    if check_cargo_deps; then
        exit 0
    else
        # 可选：预构建依赖
        # prebuild_deps
        exit 1
    fi
}

main "$@"
