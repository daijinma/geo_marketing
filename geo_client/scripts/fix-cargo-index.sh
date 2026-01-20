#!/bin/bash

# 快速修复 Cargo 索引更新卡住的问题

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_DIR/src-tauri/.cargo/config.toml"

cd "$PROJECT_DIR"

echo "🔧 修复 Cargo 索引更新问题..."
echo ""
echo "请选择解决方案："
echo "1. 临时使用官方源 + sparse protocol（推荐，如果网络允许）"
echo "2. 使用离线模式（如果依赖已下载）"
echo "3. 清除索引缓存后重试"
echo ""
read -p "请选择 (1/2/3): " choice

case $choice in
  1)
    echo "📝 切换到官方源 + sparse protocol..."
    # 备份原配置
    cp "$CONFIG_FILE" "$CONFIG_FILE.backup"
    
    # 临时注释掉镜像源，启用 sparse protocol
    sed -i.bak 's/^\[source.crates-io\]/# [source.crates-io]/' "$CONFIG_FILE"
    sed -i.bak 's/^replace-with = /# replace-with = /' "$CONFIG_FILE"
    sed -i.bak 's/^\[source.ustc\]/# [source.ustc]/' "$CONFIG_FILE"
    sed -i.bak 's/^registry = /# registry = /' "$CONFIG_FILE"
    
    # 添加 sparse protocol 配置
    if ! grep -q "\[registries.crates-io\]" "$CONFIG_FILE"; then
      cat >> "$CONFIG_FILE" << 'EOF'

# 使用官方源 + sparse protocol（更快）
[registries.crates-io]
protocol = "sparse"
EOF
    fi
    
    echo "✅ 已切换到官方源 + sparse protocol"
    echo "💡 如果网络较慢，可以运行: ./scripts/fix-cargo-index.sh 恢复镜像源"
    ;;
  2)
    echo "📦 使用离线模式..."
    cd "$PROJECT_DIR/src-tauri"
    if cargo build --offline --release 2>/dev/null; then
      echo "✅ 离线构建成功"
    else
      echo "❌ 离线构建失败，请先运行: cd src-tauri && cargo fetch"
    fi
    ;;
  3)
    echo "🧹 清除索引缓存..."
    rm -rf ~/.cargo/registry/index
    rm -rf ~/.cargo/.package-cache
    echo "✅ 缓存已清除，请重新运行构建命令"
    ;;
  *)
    echo "❌ 无效选择"
    exit 1
    ;;
esac
