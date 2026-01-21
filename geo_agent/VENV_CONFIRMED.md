# ✅ 虚拟环境配置确认

## 虚拟环境状态

**状态**: ✅ 已配置本地虚拟环境  
**位置**: `geo_agent/.venv/`  
**类型**: 项目本地隔离环境  
**管理**: uv 自动管理

## 配置详情

### 虚拟环境信息

```
目录: geo_agent/.venv/
Python: 3.12.12
实现: CPython
管理器: uv 0.9.17
系统包: 隔离（include-system-site-packages = false）
提示符: (geo-agent)
```

### 目录结构

```
.venv/
├── bin/                    # 可执行文件
│   ├── python             # Python 3.12.12
│   ├── pip
│   ├── uvicorn
│   └── pytest
├── lib/                    # Python 库
│   └── python3.12/
│       └── site-packages/  # 已安装的包
├── pyvenv.cfg             # 虚拟环境配置
└── CACHEDIR.TAG           # 缓存标识
```

## 验证命令

### 检查虚拟环境

```bash
# 使用 Makefile（推荐）
make check-env

# 查看虚拟环境目录
ls -la .venv/

# 查看配置
cat .venv/pyvenv.cfg
```

### 检查 Python 位置

```bash
# 运行后应该显示 .venv 目录下的 python
uv run which python

# 输出示例:
# /Users/cow/Desktop/p_space/geo_marketing/geo_agent/.venv/bin/python
```

### 检查已安装的包

```bash
# 查看所有包
uv pip list

# 查看特定包的位置
uv run python -c "import fastapi; print(fastapi.__file__)"
# 应该输出 .venv/lib/python3.12/site-packages/fastapi/...
```

## 使用方式

### ✅ 推荐方式：使用 uv run（无需激活）

```bash
# 运行开发服务器
make dev
# 等同于: uv run python main.py

# 运行测试
make test
# 等同于: uv run pytest

# 运行任何 Python 脚本
uv run python your_script.py

# 运行任何命令
uv run uvicorn main:app
```

**优势**:
- ✅ 无需手动激活
- ✅ 自动使用正确的虚拟环境
- ✅ 跨平台一致
- ✅ 不会忘记激活

### ⚠️ 传统方式：手动激活（不推荐）

```bash
# Mac/Linux
source .venv/bin/activate
python main.py
deactivate

# Windows
.venv\Scripts\activate
python main.py
deactivate
```

## pyproject.toml 配置

```toml
[tool.uv]
# 使用项目本地虚拟环境
managed = true
```

这确保 uv 使用项目目录下的 `.venv/`。

## 环境隔离确认

### ✅ 已隔离

- ✅ 虚拟环境位于项目目录（`.venv/`）
- ✅ 不使用系统 Python 包
- ✅ 每个项目有独立环境
- ✅ 依赖版本独立管理

### ❌ 不会影响

- ❌ 系统 Python（`/usr/bin/python`）
- ❌ 全局 pip 包
- ❌ 其他项目的虚拟环境
- ❌ Homebrew Python

## Git 版本控制

### 不提交到 Git

```gitignore
.venv/          # ← 虚拟环境（忽略）
```

### 提交到 Git

```
uv.lock         # ← 依赖锁定（提交）
pyproject.toml  # ← 项目配置（提交）
.python-version # ← Python 版本（提交）
```

## 团队协作

### 新成员设置

```bash
git clone <repo>
cd geo_agent
make install    # uv 自动创建 .venv/
make dev        # 使用 .venv/ 运行
```

### 依赖一致性

通过 `uv.lock` 确保所有人使用相同的依赖版本：

```bash
git pull
make install    # 同步到 .venv/
```

## 常见场景

### 场景 1: 首次使用

```bash
cd geo_agent
make install    # 自动创建 .venv/ 并安装依赖
make check-env  # 验证环境
make dev        # 运行项目
```

### 场景 2: 依赖更新

```bash
uv sync --upgrade  # 更新 .venv/ 中的包
git add uv.lock
git commit -m "Update dependencies"
```

### 场景 3: 虚拟环境损坏

```bash
rm -rf .venv    # 删除虚拟环境
make install    # 重新创建
```

### 场景 4: 切换项目

```bash
cd project_a
make dev        # 使用 project_a/.venv/

cd ../project_b
make dev        # 使用 project_b/.venv/
```

完全隔离，互不干扰！

## 性能说明

### 虚拟环境大小

- 初始大小: ~100MB（基础 Python）
- 安装依赖后: ~200-300MB
- 包含所有项目依赖

### 安装速度

使用 uv：
- 首次创建 .venv/: ~2s
- 安装依赖: ~2s
- 总计: ~4s（比 pip 快 10-20 倍）

## 多 Python 版本

如果系统有多个 Python 版本，uv 会自动使用正确的版本：

```
.python-version  → 指定 3.12
uv              → 自动下载/使用 Python 3.12.12
.venv/          → 使用 Python 3.12.12
```

无需手动管理 Python 版本！

## 故障排查

### 问题 1: .venv/ 不存在

**解决**:
```bash
make install
```

### 问题 2: 使用了错误的 Python

**解决**:
```bash
# 检查
uv run which python

# 应该输出包含 .venv/ 的路径
# 如果不是，重建虚拟环境
rm -rf .venv
make install
```

### 问题 3: 包找不到

**解决**:
```bash
# 重新同步依赖
make install
```

### 问题 4: 虚拟环境损坏

**解决**:
```bash
# 完全重建
make clean
make install
```

## 总结

### ✅ 已确认

- ✅ 虚拟环境位置: `geo_agent/.venv/`
- ✅ Python 版本: 3.12.12
- ✅ 包隔离: 完全隔离
- ✅ 自动管理: uv 管理
- ✅ 团队一致: 通过 uv.lock

### 🎯 使用方法

```bash
# 开发
make dev

# 测试
make test

# 验证环境
make check-env

# 任何 Python 命令
uv run python -c "import sys; print(sys.executable)"
```

### 📖 更多信息

- [ENV_INFO.md](ENV_INFO.md) - 详细环境说明
- [UV_GUIDE.md](UV_GUIDE.md) - uv 使用指南
- [README.md](README.md) - 项目文档

---

**✅ 虚拟环境配置完成，可以安全使用！** 🎉

项目使用本地隔离的虚拟环境，不会影响系统或其他项目。
