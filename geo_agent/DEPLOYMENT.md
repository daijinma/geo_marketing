# geo_agent 部署指南

## 生产环境部署

### 方案 1: 直接部署

#### 1. 环境准备

```bash
# 安装 Python 3.12+
python --version

# 克隆项目
cd /path/to/geo_agent

# 创建虚拟环境（推荐）
python -m venv venv
source venv/bin/activate  # Linux/Mac
# 或
venv\Scripts\activate  # Windows

# 安装依赖
pip install -r requirements.txt
```

#### 2. 配置环境变量

创建生产环境 `.env` 文件：

```env
# 必需配置
DASHSCOPE_API_KEY=sk-your-production-key

# 服务配置
PORT=8100
HOST=0.0.0.0
LOG_LEVEL=INFO

# 可选：API 访问控制
AGENT_API_KEYS=prod-key-1,prod-key-2
```

#### 3. 启动服务

```bash
# 生产模式启动（多 worker）
make prod

# 或使用 systemd（推荐）
sudo systemctl start geo_agent
```

#### 4. 配置 systemd（Linux）

创建 `/etc/systemd/system/geo_agent.service`：

```ini
[Unit]
Description=geo_agent OpenAI Compatible Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/geo_agent
Environment="PATH=/path/to/geo_agent/venv/bin"
EnvironmentFile=/path/to/geo_agent/.env
ExecStart=/path/to/geo_agent/venv/bin/python main.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable geo_agent
sudo systemctl start geo_agent
sudo systemctl status geo_agent
```

### 方案 2: Docker 部署

#### 1. 构建镜像

```bash
cd geo_agent
docker build -t geo_agent:latest .
```

#### 2. 运行容器

```bash
# 使用 .env 文件
docker run -d \
  --name geo_agent \
  -p 8100:8100 \
  --env-file .env \
  --restart unless-stopped \
  geo_agent:latest

# 或直接传递环境变量
docker run -d \
  --name geo_agent \
  -p 8100:8100 \
  -e DASHSCOPE_API_KEY=sk-xxx \
  -e PORT=8100 \
  -e LOG_LEVEL=INFO \
  --restart unless-stopped \
  geo_agent:latest
```

#### 3. 查看日志

```bash
docker logs -f geo_agent
```

#### 4. Docker Compose

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  geo_agent:
    build: .
    ports:
      - "8100:8100"
    env_file:
      - .env
    volumes:
      - ./logs:/app/logs
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8100/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

启动：

```bash
docker-compose up -d
```

### 方案 3: Nginx 反向代理

#### 1. 安装 Nginx

```bash
sudo apt install nginx  # Ubuntu/Debian
# 或
sudo yum install nginx  # CentOS/RHEL
```

#### 2. 配置 Nginx

创建 `/etc/nginx/sites-available/geo_agent`：

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    # SSL 配置（推荐）
    # listen 443 ssl http2;
    # ssl_certificate /path/to/cert.pem;
    # ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8100;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # SSE 支持（流式响应）
        proxy_buffering off;
        proxy_cache off;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding on;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 日志
    access_log /var/log/nginx/geo_agent_access.log;
    error_log /var/log/nginx/geo_agent_error.log;
}
```

启用配置：

```bash
sudo ln -s /etc/nginx/sites-available/geo_agent /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 性能优化

### 1. Worker 数量

调整 `config.yaml` 中的 worker 数量：

```yaml
server:
  workers: 4  # 建议设置为 CPU 核心数
```

或环境变量：

```bash
WORKERS=4 make prod
```

### 2. 连接池

DashScope SDK 会自动管理连接池，无需额外配置。

### 3. 日志轮转

使用 logrotate 管理日志：

创建 `/etc/logrotate.d/geo_agent`：

```
/path/to/geo_agent/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0640 www-data www-data
    sharedscripts
    postrotate
        systemctl reload geo_agent > /dev/null 2>&1 || true
    endscript
}
```

## 监控

### 1. 健康检查

```bash
curl http://localhost:8100/health
```

响应：

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "qwen_api_configured": true
}
```

### 2. 日志监控

```bash
# 实时监控
tail -f logs/qwen_calls.log | jq .

# 统计
make stats
```

### 3. Prometheus 监控（可选）

安装 prometheus-fastapi-instrumentator：

```bash
pip install prometheus-fastapi-instrumentator
```

在 `main.py` 中添加：

```python
from prometheus_fastapi_instrumentator import Instrumentator

Instrumentator().instrument(app).expose(app)
```

访问指标：http://localhost:8100/metrics

## 安全建议

### 1. API 访问控制

启用 API Key 验证：

```env
AGENT_API_KEYS=key1,key2,key3
```

在请求中添加：

```bash
curl -H "Authorization: Bearer key1" \
  http://localhost:8100/v1/chat/completions \
  ...
```

### 2. HTTPS

生产环境务必使用 HTTPS：

```bash
# 使用 Let's Encrypt
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d api.yourdomain.com
```

### 3. 防火墙

```bash
# 只允许必要的端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 4. 限流

在 Nginx 中配置限流：

```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

server {
    location / {
        limit_req zone=api_limit burst=20 nodelay;
        ...
    }
}
```

## 故障恢复

### 1. 自动重启

systemd 配置已包含：

```ini
Restart=always
RestartSec=10
```

Docker 配置：

```bash
docker run --restart unless-stopped ...
```

### 2. 备份

备份关键配置：

```bash
# 备份脚本
#!/bin/bash
BACKUP_DIR=/backup/geo_agent/$(date +%Y%m%d)
mkdir -p $BACKUP_DIR
cp .env $BACKUP_DIR/
cp config.yaml $BACKUP_DIR/
tar -czf $BACKUP_DIR/logs.tar.gz logs/
```

### 3. 日志归档

定期归档日志：

```bash
# 归档 30 天前的日志
find logs/ -name "*.log" -mtime +30 -exec gzip {} \;
```

## 性能基准

预期性能指标：

- **QPS**: 100-500（取决于服务器配置）
- **延迟**: 500-2000ms（取决于模型和请求复杂度）
- **并发**: 支持 100+ 并发连接

### 压力测试

使用 Apache Bench：

```bash
ab -n 1000 -c 10 -p request.json -T application/json \
  http://localhost:8100/v1/chat/completions
```

## 扩展部署

### 1. 负载均衡

使用 Nginx 负载均衡：

```nginx
upstream geo_agent_cluster {
    server 127.0.0.1:8100;
    server 127.0.0.1:8101;
    server 127.0.0.1:8102;
}

server {
    location / {
        proxy_pass http://geo_agent_cluster;
        ...
    }
}
```

### 2. Kubernetes 部署

创建 `k8s-deployment.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: geo-agent
spec:
  replicas: 3
  selector:
    matchLabels:
      app: geo-agent
  template:
    metadata:
      labels:
        app: geo-agent
    spec:
      containers:
      - name: geo-agent
        image: geo_agent:latest
        ports:
        - containerPort: 8100
        env:
        - name: DASHSCOPE_API_KEY
          valueFrom:
            secretKeyRef:
              name: geo-agent-secrets
              key: api-key
---
apiVersion: v1
kind: Service
metadata:
  name: geo-agent-service
spec:
  selector:
    app: geo-agent
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8100
  type: LoadBalancer
```

部署：

```bash
kubectl apply -f k8s-deployment.yaml
```

## 问题排查

### 查看日志

```bash
# systemd
sudo journalctl -u geo_agent -f

# Docker
docker logs -f geo_agent

# 直接运行
tail -f logs/*.log
```

### 常见问题

1. **服务启动失败**: 检查 `.env` 配置
2. **API 调用失败**: 验证 DASHSCOPE_API_KEY
3. **响应慢**: 检查网络和 DashScope API 状态
4. **内存占用高**: 减少 worker 数量或增加服务器内存

## 维护计划

- **每日**: 检查日志和错误率
- **每周**: 查看 Token 使用统计
- **每月**: 清理旧日志，更新依赖
- **每季度**: 性能测试和优化

## 联系支持

遇到问题请查看：
- 日志文件: `logs/`
- API 文档: http://localhost:8100/docs
- 健康状态: http://localhost:8100/health
