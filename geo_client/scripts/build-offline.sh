#!/bin/bash

# 离线构建脚本（跳过索引更新）

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TAURI_DIR="$PROJECT_DIR/src-tauri"

cd "$PROJECT_DIR"

echo "🔧 使用离线模式构建（跳过索引更新）..."
echo "💡 提示: 如果依赖未下载，请先运行: cd src-tauri && cargo fetch"

cd "$TAURI_DIR"

# 使用离线模式构建
cargo build --offline --release

echo "✅ 离线构建完成"
