# geo_agent 快速开始指南

## 5 分钟快速上手

### 0. 安装 uv（首次使用）

```bash
# macOS / Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# Windows
powershell -c "irm https://astral.sh/uv/install.ps1 | iex"

# 验证安装
uv --version
```

> **为什么使用 uv?**  
> uv 是 Rust 编写的现代 Python 包管理器，比 pip 快 10-100 倍，自动管理虚拟环境。

### 1. 安装依赖

```bash
cd geo_agent
make install
# 或
uv sync
```

### 2. 配置 API Key

创建 `.env` 文件：

```bash
# 复制示例文件
cp .env.example .env

# 编辑 .env，添加你的 API Key
# DASHSCOPE_API_KEY=sk-your-key-here
```

获取 DashScope API Key：
1. 访问 https://dashscope.console.aliyun.com/
2. 登录阿里云账号
3. 创建 API Key

### 3. 启动服务

```bash
make dev
```

看到以下输出表示启动成功：

```
INFO:     Uvicorn running on http://0.0.0.0:8100 (Press CTRL+C to quit)
```

### 4. 测试服务

#### 方法 1: 使用 curl

```bash
curl http://localhost:8100/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen3-max",
    "messages": [{"role": "user", "content": "你好"}]
  }'
```

#### 方法 2: 使用 Python

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

#### 方法 3: 使用测试脚本

```bash
python test_agent.py
```

### 5. 查看日志

```bash
# 实时查看 Qwen API 调用日志
make logs-qwen

# 查看所有日志
make logs
```

## 集成到你的项目

### JavaScript/TypeScript

```typescript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8100/v1',
  apiKey: 'dummy'
});

const response = await client.chat.completions.create({
  model: 'qwen3-max',
  messages: [{ role: 'user', content: 'Hello' }]
});

console.log(response.choices[0].message.content);
```

### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8100/v1",
    api_key="dummy"
)

response = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "Hello"}]
)

print(response.choices[0].message.content)
```

### Go

```go
package main

import (
    "context"
    "fmt"
    openai "github.com/sashabaranov/go-openai"
)

func main() {
    config := openai.DefaultConfig("dummy")
    config.BaseURL = "http://localhost:8100/v1"
    client := openai.NewClientWithConfig(config)

    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: "qwen3-max",
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: "Hello"},
            },
        },
    )

    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## 常见用例

### 1. 简单问答

```python
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[
        {"role": "user", "content": "Python 是什么？"}
    ]
)
```

### 2. 带系统提示词

```python
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[
        {"role": "system", "content": "你是一个 Python 专家"},
        {"role": "user", "content": "解释装饰器"}
    ]
)
```

### 3. 多轮对话

```python
messages = [
    {"role": "user", "content": "我叫小明"},
    {"role": "assistant", "content": "你好小明！"},
    {"role": "user", "content": "我叫什么名字？"}
]

response = client.chat.completions.create(
    model="qwen3-max",
    messages=messages
)
```

### 4. 流式响应

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

### 5. 调整参数

```python
response = client.chat.completions.create(
    model="qwen3-max",
    messages=[{"role": "user", "content": "生成一个故事"}],
    temperature=0.9,      # 更随机
    max_tokens=500,       # 限制长度
    top_p=0.95
)
```

## 监控和调试

### 查看实时日志

```bash
# 终端 1: 启动服务
make dev

# 终端 2: 查看日志
make logs-qwen
```

### 统计信息

```bash
make stats
```

输出示例：
```
Token usage statistics:
Total API calls: 42
Total tokens used: 12500
Average latency (ms): 1234.56
```

## 故障排查

### 问题 1: 连接被拒绝

**检查**: 服务是否启动？

```bash
curl http://localhost:8100/health
```

### 问题 2: API Key 错误

**检查**: `.env` 文件是否配置正确？

```bash
cat .env | grep DASHSCOPE_API_KEY
```

### 问题 3: 响应慢

**检查**: 查看日志中的 `latency_ms`

```bash
make logs-qwen
```

## 下一步

- 📖 阅读完整文档: [README.md](README.md)
- 🧪 运行测试套件: `python test_agent.py`
- 📊 查看 API 文档: http://localhost:8100/docs
- 📝 查看日志: `make logs`

## 需要帮助？

查看完整文档或检查日志文件获取更多信息。
