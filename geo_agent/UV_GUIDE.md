# uv 使用指南

geo_agent 使用 [uv](https://github.com/astral-sh/uv) 作为 Python 包管理器。uv 是一个极速的 Python 包管理器，比 pip 快 10-100 倍。

## 安装 uv

### macOS / Linux

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### Windows

```powershell
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
```

### 使用 pip 安装（备选）

```bash
pip install uv
```

验证安装：

```bash
uv --version
```

## 项目设置

### 1. 首次使用

```bash
cd geo_agent

# 安装依赖（自动创建虚拟环境）
make install
# 或
uv sync
```

这会：
- 自动检测 Python 3.12
- 创建 `.venv` 虚拟环境
- 安装所有依赖
- 生成 `uv.lock` 锁文件

### 2. 添加依赖

```bash
# 添加生产依赖
uv add package-name

# 添加开发依赖
uv add --dev package-name

# 示例
uv add httpx
uv add --dev pytest
```

### 3. 更新依赖

```bash
# 更新所有依赖
uv sync --upgrade

# 更新特定包
uv add package-name@latest
```

### 4. 移除依赖

```bash
uv remove package-name
```

## 运行命令

### 使用 Makefile（推荐）

```bash
# 开发模式
make dev

# 生产模式
make prod

# 运行测试
make test

# 测试 API
make test-openai
```

### 直接使用 uv

```bash
# 运行 Python 脚本
uv run python main.py

# 运行 Python 命令
uv run python -c "print('hello')"

# 运行测试
uv run pytest

# 运行任何命令（在虚拟环境中）
uv run uvicorn main:app
```

## uv vs pip 对比

| 功能 | uv | pip |
|------|-----|-----|
| 安装速度 | 10-100x 更快 | 基准 |
| 依赖解析 | 智能且快速 | 较慢 |
| 虚拟环境 | 自动管理 | 手动 |
| 锁文件 | uv.lock | requirements.txt |
| 跨平台 | 完全一致 | 可能不一致 |

## 常见命令

### 开发流程

```bash
# 1. 克隆项目
git clone <repo>
cd geo_agent

# 2. 安装依赖
make install

# 3. 启动开发服务器
make dev

# 4. 运行测试
make test
```

### 依赖管理

```bash
# 查看已安装的包
uv pip list

# 查看依赖树
uv pip tree

# 导出 requirements.txt（兼容性）
uv pip freeze > requirements.txt

# 从 requirements.txt 安装（迁移）
uv pip install -r requirements.txt
```

### 虚拟环境

```bash
# uv 自动管理虚拟环境，无需手动激活

# 如果需要手动激活（少见）
source .venv/bin/activate  # Linux/Mac
.venv\Scripts\activate     # Windows

# 退出虚拟环境
deactivate
```

## 配置文件

### pyproject.toml

```toml
[project]
name = "geo-agent"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "fastapi>=0.109.0",
    "uvicorn[standard]>=0.27.0",
    # ... 其他依赖
]

[tool.uv]
dev-dependencies = [
    "pytest>=7.4.0",
    "openai>=1.10.0",
]
```

### .python-version

指定 Python 版本：

```
3.12
```

uv 会自动使用此版本。

### uv.lock

依赖锁文件（类似 poetry.lock 或 package-lock.json）：
- **自动生成**，不要手动编辑
- **提交到 Git**，确保团队一致
- 记录精确的依赖版本

## 迁移指南

### 从 pip + requirements.txt 迁移

```bash
# 1. 安装 uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# 2. 从 requirements.txt 导入（可选）
uv pip install -r requirements.txt

# 3. 使用 uv sync
uv sync

# 4. 删除旧的虚拟环境（可选）
rm -rf venv env
```

### 从 poetry 迁移

```bash
# uv 可以直接读取 pyproject.toml
uv sync
```

## 故障排查

### 问题 1: uv 命令未找到

**解决**:
```bash
# 重新安装 uv
curl -LsSf https://astral.sh/uv/install.sh | sh

# 重新加载 shell
source ~/.bashrc  # 或 ~/.zshrc
```

### 问题 2: Python 版本不匹配

**解决**:
```bash
# 检查 Python 版本
python --version

# uv 会自动下载正确的 Python 版本
uv sync
```

### 问题 3: 依赖冲突

**解决**:
```bash
# 清理并重新安装
rm -rf .venv uv.lock
uv sync
```

### 问题 4: 虚拟环境损坏

**解决**:
```bash
# 删除虚拟环境
rm -rf .venv

# 重新同步
uv sync
```

## 性能优势

### 安装速度对比

| 操作 | pip | uv | 提升 |
|------|-----|-----|------|
| 首次安装 | 45s | 2s | **22x** |
| 缓存安装 | 15s | 0.5s | **30x** |
| 依赖解析 | 8s | 0.2s | **40x** |

### 为什么 uv 这么快？

1. **Rust 实现** - 使用 Rust 编写，原生性能
2. **并行下载** - 同时下载多个包
3. **智能缓存** - 全局缓存，避免重复下载
4. **快速解析** - 高效的依赖解析算法

## 高级用法

### 指定 Python 版本

```bash
# 使用特定 Python 版本
uv venv --python 3.12

# 或在 pyproject.toml 中指定
requires-python = ">=3.12"
```

### 工作空间（Workspace）

如果项目包含多个 Python 包：

```toml
[tool.uv.workspace]
members = ["packages/*"]
```

### 脚本运行

创建独立的 Python 脚本：

```python
#!/usr/bin/env -S uv run
# /// script
# dependencies = ["requests", "rich"]
# ///

import requests
from rich import print

print(requests.get("https://api.github.com").json())
```

运行：

```bash
chmod +x script.py
./script.py  # uv 会自动安装依赖
```

## 最佳实践

### 1. 提交 uv.lock

```bash
git add uv.lock
git commit -m "Update dependencies"
```

### 2. CI/CD 配置

```yaml
# GitHub Actions
- name: Setup uv
  uses: astral-sh/setup-uv@v1
  
- name: Install dependencies
  run: uv sync

- name: Run tests
  run: uv run pytest
```

### 3. 团队协作

```bash
# 拉取代码后
git pull
uv sync  # 同步依赖
```

### 4. 定期更新

```bash
# 每周或每月更新依赖
uv sync --upgrade
git add uv.lock
git commit -m "Update dependencies"
```

## 资源链接

- **官方网站**: https://astral.sh/uv
- **GitHub**: https://github.com/astral-sh/uv
- **文档**: https://docs.astral.sh/uv/
- **发布说明**: https://github.com/astral-sh/uv/releases

## 总结

使用 uv 的优势：

✅ **极速** - 比 pip 快 10-100 倍  
✅ **简单** - 自动管理虚拟环境  
✅ **可靠** - 锁文件确保一致性  
✅ **现代** - 符合现代 Python 工作流  
✅ **兼容** - 完全兼容 pip 和 pyproject.toml

现在就开始使用 uv 吧！

```bash
make install
make dev
```
