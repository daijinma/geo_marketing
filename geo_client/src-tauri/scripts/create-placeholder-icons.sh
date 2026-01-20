#!/bin/bash

# 创建占位图标脚本
# 注意：这只是占位图标，实际使用时应该使用真实的图标文件

ICONS_DIR="icons"

mkdir -p "$ICONS_DIR"

echo "📝 创建占位图标文件..."
echo "💡 提示: 这些是占位图标，实际使用时请运行: npm run tauri icon <your-icon.png>"

# 创建简单的占位说明文件
cat > "$ICONS_DIR/README.md" << 'EOF'
# 图标文件

这些是应用图标文件。当前使用的是占位图标。

## 生成真实图标

使用 Tauri CLI 生成图标：

```bash
# 从项目根目录运行
npm run tauri icon <path-to-your-icon.png>
# 或
pnpm tauri icon <path-to-your-icon.png>
```

图标要求：
- 推荐尺寸：1024x1024px
- 格式：PNG（支持透明）或 SVG
- 正方形

生成的图标会自动放置在 `src-tauri/icons/` 目录中。
EOF

echo "✅ 占位图标说明已创建"
echo "📖 查看 $ICONS_DIR/README.md 了解如何生成真实图标"
