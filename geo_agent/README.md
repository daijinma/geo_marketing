# geo_agent

OpenAI 兼容的 Agent 服务，内部使用阿里云 qwen3-max 模型。提供完整的日志监控系统。

## 功能特性

- ✅ **OpenAI 兼容接口**: 完全兼容 OpenAI API 规范，支持无缝迁移
- ✅ **qwen3-max 驱动**: 内部使用阿里云 DashScope API 调用 qwen-max 模型
- ✅ **完整日志系统**: 记录所有请求、响应、Token 使用量、延迟等信息
- ✅ **流式响应支持**: 支持 SSE 流式输出
- ✅ **生产就绪**: 包含错误处理、重试机制、CORS 配置等

## 技术栈

- **Python 3.12+**
- **FastAPI**: 高性能异步 Web 框架
- **DashScope SDK**: 阿里云官方 SDK
- **structlog**: 结构化日志
- **Pydantic**: 数据验证

## 快速开始

### 0. 安装 uv（首次使用）

本项目使用 [uv](https://github.com/astral-sh/uv) 作为包管理器：

```bash
# macOS / Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
```

### 1. 安装依赖

```bash
cd geo_agent
make install
# 或
uv sync
```

### 2. 配置环境变量

创建 `.env` 文件：

```bash
cp .env.example .env
```

编辑 `.env` 文件，配置必需的 API Key：

```env
DASHSCOPE_API_KEY=sk-your-api-key-here
PORT=8100
LOG_LEVEL=INFO
```

### 3. 启动服务

#### 开发模式（支持热重载）

```bash
make dev
```

#### 生产模式

```bash
make prod
```

服务启动后访问：

- **API 文档**: http://localhost:8100/docs
- **健康检查**: http://localhost:8100/health

## API 接口

### OpenAI 兼容端点

#### 1. 聊天补全（主要接口）

```bash
POST /v1/chat/completions
```

**请求示例**：

```bash
curl http://localhost:8100/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen3-max",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手"},
      {"role": "user", "content": "介绍一下北京"}
    ],
    "temperature": 0.7,
    "max_tokens": 2000
  }'
```

**响应示例**：

```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "qwen3-max",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "北京是中国的首都..."
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 150,
    "total_tokens": 170
  }
}
```

#### 2. 流式响应

```bash
curl http://localhost:8100/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen3-max",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

#### 3. 模型列表

```bash
GET /v1/models
```

### 使用 OpenAI SDK

```python
from openai import OpenAI

# 配置客户端
client = OpenAI(
    base_url="http://localhost:8100/v1",
    api_key="dummy"  # 如果不需要鉴权，随意填写
)

# 调用 API
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[
        {"role": "system", "content": "你是一个有帮助的助手"},
        {"role": "user", "content": "介绍一下人工智能"}
    ],
    temperature=0.7
)

print(response.choices[0].message.content)
```

### 添加/管理依赖

```bash
# 添加新的依赖
uv add package-name

# 添加开发依赖
uv add --dev package-name

# 更新依赖
uv sync --upgrade

# 移除依赖
uv remove package-name
```

> 📖 详细的 uv 使用方法请查看 [UV_GUIDE.md](UV_GUIDE.md)

### 流式调用

```python
stream = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "写一首诗"}],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

## 日志系统

服务提供完整的日志系统，包括控制台详细输出和文件记录。

### 控制台日志（详细打印）

服务运行时会在控制台详细打印所有中转过程的信息：

- 🔵 **HTTP 请求** - 接收到的原始 HTTP 请求信息
- 📨 **OpenAI 格式请求** - 解析后的 OpenAI 格式请求详情
- 📤 **中转请求** - 准备发送给 Qwen API 的请求
- 🚀 **DashScope API 调用** - 实际调用 DashScope API 的详细信息
- ✅ **DashScope API 响应** - 从 DashScope 收到的响应
- 📥 **中转响应** - 转换后的响应详情
- ✨ **OpenAI 格式响应** - 最终返回的 OpenAI 格式响应
- 🟢 **HTTP 响应** - HTTP 响应状态和耗时

每个阶段都会显示详细的参数、内容、Token 使用量和耗时信息，方便开发调试和问题排查。

> 📖 查看 [LOGGING_ENHANCEMENTS.md](LOGGING_ENHANCEMENTS.md) 了解详细的日志格式和使用说明。

### 日志文件

所有日志文件位于 `logs/` 目录，使用 JSON 格式存储：

### 1. 访问日志 (`access.log`)

记录所有 HTTP 请求：

```json
{
  "timestamp": "2026-01-21T10:30:45.123Z",
  "level": "INFO",
  "event": "http_access",
  "method": "POST",
  "path": "/v1/chat/completions",
  "status_code": 200,
  "latency_ms": 1234.56,
  "client_ip": "127.0.0.1",
  "request_id": "req_abc123"
}
```

### 2. Qwen API 调用日志 (`qwen_calls.log`)

记录所有 Qwen API 调用的详细信息：

```json
{
  "timestamp": "2026-01-21T10:30:45.123Z",
  "request_id": "req_abc123",
  "level": "INFO",
  "event": "qwen_api_call",
  "request": {
    "model": "qwen3-max",
    "messages": [...],
    "temperature": 0.7,
    "max_tokens": 2000
  },
  "response": {
    "content": "...",
    "finish_reason": "stop"
  },
  "usage": {
    "prompt_tokens": 20,
    "completion_tokens": 150,
    "total_tokens": 170
  },
  "latency_ms": 1234.56,
  "client_ip": "127.0.0.1"
}
```

### 3. 错误日志 (`error.log`)

记录所有错误和异常：

```json
{
  "timestamp": "2026-01-21T10:30:45.123Z",
  "level": "ERROR",
  "event": "error",
  "error_type": "ValueError",
  "error_message": "...",
  "request_id": "req_abc123"
}
```

### 查看日志

```bash
# 实时查看所有日志
make logs

# 查看 Qwen API 调用日志（JSON 格式化）
make logs-qwen

# 查看访问日志
make logs-access

# 查看错误日志
make logs-error

# 查看统计信息
make stats
```

## 测试

### 快速测试

```bash
# 使用 curl 测试
make test-curl

# 使用 OpenAI SDK 测试
make test-openai

# 测试日志增强功能
make test-logging
```

日志测试会展示完整的请求链路和所有详细打印信息。

### 完整测试脚本

创建测试脚本 `test_agent.py`：

```python
from openai import OpenAI
import time

client = OpenAI(
    base_url="http://localhost:8100/v1",
    api_key="test"
)

# 测试 1: 基本聊天
print("=== 测试 1: 基本聊天 ===")
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "1+1=?"}]
)
print(f"回答: {response.choices[0].message.content}")
print(f"Tokens: {response.usage.total_tokens}")

# 测试 2: 流式响应
print("\n=== 测试 2: 流式响应 ===")
stream = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "数到5"}],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
print()

# 测试 3: 系统提示词
print("\n=== 测试 3: 系统提示词 ===")
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[
        {"role": "system", "content": "你是一个诗人，总是用诗歌回答"},
        {"role": "user", "content": "描述春天"}
    ]
)
print(f"诗歌: {response.choices[0].message.content}")
```

运行测试：

```bash
python test_agent.py
```

## 配置说明

### config.yaml

```yaml
server:
  host: 0.0.0.0      # 监听地址
  port: 8100         # 监听端口
  workers: 4         # Worker 数量（生产模式）
  reload: false      # 热重载（开发模式设为 true）

qwen:
  api_key: ${DASHSCOPE_API_KEY}  # 从环境变量读取
  model: qwen-max                 # 使用的模型
  timeout: 60                     # API 超时（秒）
  max_retries: 3                  # 最大重试次数

logging:
  level: INFO        # 日志级别
  format: json       # 日志格式
  rotation: daily    # 日志轮转策略
  retention: 30d     # 日志保留时间
```

### 环境变量

| 变量名 | 说明 | 必需 | 默认值 |
|--------|------|------|--------|
| `DASHSCOPE_API_KEY` | 阿里云 DashScope API Key | ✅ | - |
| `PORT` | 服务端口 | ❌ | 8100 |
| `HOST` | 监听地址 | ❌ | 0.0.0.0 |
| `LOG_LEVEL` | 日志级别 | ❌ | INFO |
| `QWEN_MODEL` | 使用的模型 | ❌ | qwen-max |

## 项目结构

```
geo_agent/
├── app/
│   ├── api/              # API 路由
│   │   ├── v1/
│   │   │   ├── chat.py          # 聊天补全接口
│   │   │   ├── models.py        # 模型列表接口
│   │   │   └── completions.py   # 文本补全接口
│   │   └── health.py            # 健康检查
│   ├── core/             # 核心功能
│   │   ├── config.py            # 配置管理
│   │   ├── logger.py            # 日志系统
│   │   └── middleware.py        # 中间件
│   ├── models/           # 数据模型
│   │   ├── openai.py            # OpenAI 格式定义
│   │   └── dashscope.py         # DashScope 格式定义
│   ├── services/         # 业务逻辑
│   │   ├── qwen_client.py       # Qwen API 客户端
│   │   └── converter.py         # 格式转换器
│   └── utils/            # 工具函数
├── logs/                 # 日志文件
├── main.py              # 应用入口
├── config.yaml          # 配置文件
├── requirements.txt     # 依赖列表
├── Makefile            # 运行脚本
└── README.md           # 本文档
```

## 监控指标

查看统计信息：

```bash
make stats
```

输出示例：

```
Token usage statistics:
Total API calls: 150
Total tokens used: 45230
Average latency (ms): 1234.56
```

## 常见问题

### 1. API Key 配置错误

**错误**: `DASHSCOPE_API_KEY not configured`

**解决**: 确保在 `.env` 文件中正确配置了 `DASHSCOPE_API_KEY`

### 2. 端口被占用

**错误**: `Address already in use`

**解决**: 修改 `.env` 文件中的 `PORT` 变量，或者停止占用 8100 端口的进程

### 3. 日志文件过大

**解决**: 定期清理日志文件

```bash
make clean
```

## 开发指南

### 添加新功能

1. 在 `app/api/v1/` 中创建新的路由文件
2. 在 `main.py` 中注册路由
3. 更新 `README.md` 文档

### 调试模式

设置环境变量 `LOG_LEVEL=DEBUG` 以查看详细日志：

```bash
LOG_LEVEL=DEBUG python main.py
```

## License

MIT License

## 联系方式

如有问题，请联系项目维护者。
