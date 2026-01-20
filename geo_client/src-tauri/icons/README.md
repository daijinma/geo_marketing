# 图标文件说明

## 问题说明

### 错误 1：`icon /path/to/icon.png is not RGBA`

**原因**：Tauri 要求图标文件必须是 **RGBA 格式**（32位，包含 Alpha 通道），而当前的图标文件是 RGB 格式（24位，不包含 Alpha 通道）。

### 错误 2：`Invalid PNG signature`

**原因**：图标文件不是有效的 PNG 格式。可能是：
- WebP 格式文件被错误命名为 `.png`
- 文件损坏
- 文件格式不正确

**解决方案**：删除无效的图标文件，或使用 Tauri CLI 生成正确的图标。

## 解决方案

### 方案 1：使用 Tauri CLI 生成图标（推荐）

这是最简单和推荐的方法，会自动生成所有需要的格式：

```bash
# 从项目根目录运行
npm run tauri icon <path-to-your-source-icon.png>
# 或
pnpm tauri icon <path-to-your-source-icon.png>
```

**图标要求**：
- 推荐尺寸：**1024x1024px**
- 格式：PNG（支持透明）或 SVG
- 正方形

生成的图标会自动放置在 `src-tauri/icons/` 目录中，并更新 `tauri.conf.json` 配置。

### 方案 2：手动转换图标为 RGBA

如果已有图标文件，需要将其转换为 RGBA 格式：

#### 使用 Python (需要安装 Pillow)

```bash
pip install Pillow
python3 -c "from PIL import Image; img = Image.open('icons/icon.png'); img = img.convert('RGBA'); img.save('icons/icon.png')"
```

#### 使用在线工具

1. 访问在线转换工具（如 https://convertio.co/png-rgba/）
2. 上传图标文件
3. 选择转换为 RGBA PNG 格式
4. 下载并替换原文件

#### 使用图像编辑软件

使用 Photoshop、GIMP 或其他图像编辑软件：
1. 打开图标文件
2. 确保图像模式为 RGBA（32位）
3. 导出为 PNG 格式，确保包含 Alpha 通道

### 方案 3：暂时不使用图标

如果暂时不需要图标，可以：
1. 保持 `tauri.conf.json` 中的 `icon` 数组为空：`"icon": []`
2. 或移除/重命名图标文件

## 当前状态

- 图标文件已暂时禁用（重命名为 `icon.png.disabled`）
- 配置文件中的图标数组为空：`"icon": []`
- 构建时不会使用图标，会使用 Tauri 默认图标

## 恢复图标

当准备好 RGBA 格式的图标后：

1. **使用 Tauri CLI**（推荐）：
   ```bash
   npm run tauri icon <your-icon.png>
   ```

2. **手动添加**：
   - 将 RGBA 格式的图标文件放在 `icons/` 目录
   - 更新 `tauri.conf.json`：
     ```json
     "icon": [
       "icons/32x32.png",
       "icons/128x128.png",
       "icons/128x128@2x.png",
       "icons/icon.icns",
       "icons/icon.ico"
     ]
     ```

## 验证图标格式

使用以下命令检查图标格式：

```bash
# macOS/Linux
file icons/icon.png

# 应该显示包含 "RGBA" 或 "32-bit"
```
