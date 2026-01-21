# 🚀 geo_agent - 开始使用

## 这是什么？

**geo_agent** 是一个 OpenAI 兼容的 API 服务，内部使用阿里云 qwen3-max 模型。

✨ 无需修改代码，直接替换 OpenAI API！

## 快速开始（3 步）

### 0️⃣ 安装 uv（首次）

```bash
# macOS / Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
```

> 💡 uv 是现代化的 Python 包管理器，比 pip 快 10-100 倍！

### 1️⃣ 安装依赖

```bash
cd geo_agent
make install
# 或直接使用 uv
uv sync
```

### 2️⃣ 配置 API Key

```bash
# 复制配置文件
cp .env.example .env

# 编辑 .env，添加你的 DashScope API Key
# DASHSCOPE_API_KEY=sk-your-key-here
```

> 获取 API Key: https://dashscope.console.aliyun.com/

### 3️⃣ 启动服务

```bash
make dev
```

✅ 看到 `Uvicorn running on http://0.0.0.0:8100` 表示成功！

## 测试服务

### 方式 1: curl

```bash
curl http://localhost:8100/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen3-max",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

### 方式 2: Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8100/v1",
    api_key="test"
)

response = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "你好"}]
)

print(response.choices[0].message.content)
```

### 方式 3: 测试脚本

```bash
python test_agent.py
```

## 查看日志

```bash
# 查看 Qwen API 调用日志（JSON 格式）
make logs-qwen

# 查看所有日志
make logs

# 查看统计信息
make stats
```

## 核心功能

✅ **OpenAI 兼容**: 完全兼容 OpenAI API  
✅ **qwen3-max 驱动**: 使用阿里云最新模型  
✅ **完整日志**: 记录所有请求、响应、Token 使用  
✅ **流式响应**: 支持 SSE 流式输出  
✅ **生产就绪**: 错误处理、重试、监控等

## 接口说明

| 接口 | 说明 |
|------|------|
| `POST /v1/chat/completions` | 聊天补全（主接口） |
| `GET /v1/models` | 模型列表 |
| `GET /health` | 健康检查 |
| `GET /docs` | API 文档 (Swagger) |

## 日志文件

所有日志位于 `logs/` 目录：

- **qwen_calls.log**: Qwen API 调用日志（最重要）
  - 完整的请求和响应
  - Token 使用统计
  - 延迟监控

- **access.log**: HTTP 访问日志
- **error.log**: 错误日志

## 更多文档

| 文档 | 说明 |
|------|------|
| [README.md](README.md) | 📖 完整文档（9KB） |
| [QUICKSTART.md](QUICKSTART.md) | ⚡ 快速开始（5KB） |
| [DEPLOYMENT.md](DEPLOYMENT.md) | 🚀 部署指南（8KB） |
| [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) | 📊 项目总结（7KB） |
| [IMPLEMENTATION_REPORT.md](IMPLEMENTATION_REPORT.md) | 📝 实施报告 |

## 常用命令

```bash
make help          # 查看所有命令
make install       # 安装依赖 (uv sync)
make dev           # 开发模式 (uv run)
make prod          # 生产模式 (uv run)
make test          # 运行测试 (uv run pytest)
make logs-qwen     # 查看 Qwen 日志
make stats         # 统计信息
make clean         # 清理日志
```

### uv 直接命令

```bash
uv sync            # 同步依赖
uv add package     # 添加包
uv run python main.py  # 运行脚本
uv run pytest      # 运行测试
```

> 📖 查看 [UV_GUIDE.md](UV_GUIDE.md) 了解更多 uv 使用方法

## 使用场景

✅ 需要使用 qwen-max 但希望保持 OpenAI API 接口  
✅ 需要详细的 API 调用日志和监控  
✅ 需要在国内部署的 AI 服务  
✅ 需要自建 AI API 服务的企业

## 兼容性

✅ OpenAI Python SDK  
✅ OpenAI Node.js SDK  
✅ LangChain  
✅ LlamaIndex  
✅ 任何支持 OpenAI API 的工具

## 需要帮助？

1. 📖 查看 [README.md](README.md)
2. ⚡ 查看 [QUICKSTART.md](QUICKSTART.md)
3. 📝 查看日志文件: `logs/`
4. 🧪 运行测试: `python test_agent.py`
5. 📊 访问 API 文档: http://localhost:8100/docs

## 项目结构

```
geo_agent/
├── app/              # 应用代码
│   ├── api/         # API 路由
│   ├── core/        # 核心功能（配置、日志、中间件）
│   ├── models/      # 数据模型
│   ├── services/    # 业务逻辑（Qwen 客户端、转换器）
│   └── utils/       # 工具函数
├── logs/            # 日志文件
├── main.py          # 应用入口
├── config.yaml      # 配置文件
├── requirements.txt # 依赖
├── Makefile         # 运行脚本
└── test_agent.py    # 测试脚本
```

---

**开始使用吧！** 🎉

如有问题，查看完整文档或运行测试脚本。
