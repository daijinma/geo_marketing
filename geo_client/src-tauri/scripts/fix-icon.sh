#!/bin/bash

# 修复图标格式脚本
# 将 RGB 格式的图标转换为 RGBA 格式

ICON_DIR="icons"
ICON_FILE="$ICON_DIR/icon.png"
BACKUP_FILE="$ICON_DIR/icon.png.backup"

cd "$(dirname "$0")/.."

if [ ! -f "$ICON_FILE" ]; then
    echo "❌ 未找到图标文件: $ICON_FILE"
    exit 1
fi

echo "🔧 修复图标格式..."

# 备份原文件
cp "$ICON_FILE" "$BACKUP_FILE"
echo "✅ 已备份原图标: $BACKUP_FILE"

# 检查是否有 sips（macOS 自带）
if command -v sips &> /dev/null; then
    echo "📝 使用 sips 转换图标为 RGBA 格式..."
    # 使用 sips 转换为 RGBA
    sips -s format png -s hasAlpha 1 "$ICON_FILE" --out "$ICON_FILE.tmp" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        mv "$ICON_FILE.tmp" "$ICON_FILE"
        echo "✅ 图标已转换为 RGBA 格式"
    else
        echo "⚠️  sips 转换失败，尝试其他方法..."
        rm -f "$ICON_FILE.tmp"
    fi
fi

# 如果 sips 不可用或失败，提供手动转换说明
if [ ! -f "$ICON_FILE" ] || ! file "$ICON_FILE" | grep -q "RGBA"; then
    echo ""
    echo "⚠️  自动转换失败，请手动转换图标："
    echo ""
    echo "方法 1: 使用在线工具"
    echo "  1. 访问 https://convertio.co/png-rgba/ 或类似工具"
    echo "  2. 上传 $ICON_FILE"
    echo "  3. 选择转换为 RGBA PNG 格式"
    echo "  4. 下载并替换原文件"
    echo ""
    echo "方法 2: 使用 Python (如果已安装 Pillow)"
    echo "  python3 -c \"from PIL import Image; img = Image.open('$ICON_FILE'); img = img.convert('RGBA'); img.save('$ICON_FILE')\""
    echo ""
    echo "方法 3: 使用 Tauri CLI 生成图标（推荐）"
    echo "  从项目根目录运行:"
    echo "  npm run tauri icon <path-to-your-source-icon.png>"
    echo "  或"
    echo "  pnpm tauri icon <path-to-your-source-icon.png>"
    echo ""
    echo "方法 4: 暂时移除图标文件"
    echo "  mv $ICON_FILE $ICON_FILE.disabled"
    echo "  然后更新 tauri.conf.json，将 icon 数组设置为空: \"icon\": []"
    echo ""
    exit 1
fi

# 验证转换结果
if file "$ICON_FILE" | grep -q "RGBA"; then
    echo "✅ 图标格式验证通过：RGBA"
    exit 0
else
    echo "❌ 图标格式验证失败，请检查文件"
    exit 1
fi
