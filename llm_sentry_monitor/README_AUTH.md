# 用户认证系统使用说明

本文档说明如何使用服务端的用户认证功能进行客户端联调测试。

## 功能概述

服务端已实现完整的用户认证系统，包括：
- 用户表（users）：存储用户信息和加密密码
- 认证Token表（auth_tokens）：管理token信息（可选）
- 登录API接口：`POST /client/auth/login`
- JWT Token生成和验证
- 密码加密（bcrypt）

## 部署步骤

### 1. 执行数据库迁移

首先需要执行数据库迁移脚本，创建用户表和认证相关表：

```bash
cd geo_db
docker exec -i <容器名> psql -U geo_admin -d geo_monitor < migrations/006_add_users_and_auth_tables.sql
```

或者在数据库客户端中直接执行 `migrations/006_add_users_and_auth_tables.sql` 文件。

### 2. 安装依赖

安装新增的Python依赖：

```bash
cd llm_sentry_monitor
pip install -r requirements.txt
```

新增的依赖包括：
- `bcrypt`：密码加密
- `PyJWT`：JWT token生成和验证
- `fastapi`：Web框架（如果还没有安装）
- `uvicorn[standard]`：ASGI服务器（如果还没有安装）

### 3. 配置JWT密钥（可选）

在生产环境中，建议设置环境变量 `JWT_SECRET_KEY`：

```bash
export JWT_SECRET_KEY="your-secure-random-secret-key"
```

或者在 `.env` 文件中添加：
```
JWT_SECRET_KEY=your-secure-random-secret-key
```

如果不设置，默认使用 `"your-secret-key-change-in-production"`（仅用于开发测试）。

### 4. 创建测试用户

使用提供的脚本创建测试用户：

```bash
cd llm_sentry_monitor
python scripts/create_test_user.py
```

脚本会交互式询问用户信息：
- 用户名（默认：admin）
- 密码（默认：admin123）
- 邮箱（可选）
- 全名（可选）
- 是否管理员（默认：否）

或者直接在Python中创建用户：

```python
from core.auth import create_user

# 创建普通用户
user_id = create_user(
    username="admin",
    password="admin123",
    email="admin@example.com",
    full_name="管理员",
    is_admin=True
)
```

## API使用

### 登录接口

**端点**: `POST /client/auth/login`

**请求体**:
```json
{
    "username": "admin",
    "password": "admin123"
}
```

**响应** (成功):
```json
{
    "success": true,
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-01-17T12:00:00"
}
```

**响应** (失败):
```json
{
    "detail": "用户名或密码错误"
}
```
状态码: 401

### 使用Token

客户端登录成功后，需要在后续请求的Header中携带token：

```
Authorization: Bearer <token>
```

示例（使用curl）:
```bash
curl -X POST http://127.0.0.1:8000/client/tasks/pending \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json"
```

## Token说明

- **有效期**: 24小时
- **算法**: HS256
- **包含信息**:
  - `sub`: 用户ID
  - `username`: 用户名
  - `is_admin`: 是否管理员
  - `exp`: 过期时间（UTC）
  - `iat`: 签发时间（UTC）

## 数据库表结构

### users表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL | 主键 |
| username | VARCHAR(100) | 用户名（唯一） |
| password_hash | TEXT | 加密后的密码 |
| email | VARCHAR(255) | 邮箱 |
| full_name | VARCHAR(255) | 全名 |
| is_active | BOOLEAN | 是否激活 |
| is_admin | BOOLEAN | 是否管理员 |
| last_login_at | TIMESTAMP | 最后登录时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### auth_tokens表（可选）

用于存储token信息，支持token撤销和黑名单管理。

## 安全注意事项

1. **密码加密**: 所有密码使用bcrypt加密存储，不会明文保存
2. **Token过期**: Token默认24小时过期，过期后需要重新登录
3. **JWT密钥**: 生产环境务必设置强随机密钥
4. **HTTPS**: 生产环境建议使用HTTPS传输，避免token被窃取
5. **用户激活**: 只有 `is_active=True` 的用户可以登录

## 测试流程

1. 执行数据库迁移脚本
2. 创建测试用户（如：admin/admin123）
3. 启动API服务器：`python main.py api` 或 `uvicorn api:app`
4. 客户端调用登录接口获取token
5. 使用token访问其他需要认证的接口（如果实现了认证中间件）

## 故障排查

### 登录失败

1. 检查数据库迁移是否成功执行
2. 确认用户是否存在且 `is_active=True`
3. 检查密码是否正确
4. 查看服务端日志获取详细错误信息

### Token验证失败

1. 检查token是否过期（24小时）
2. 确认JWT_SECRET_KEY是否一致
3. 检查token格式是否正确（Bearer token）

### 数据库连接错误

1. 检查数据库服务是否正常运行
2. 确认环境变量配置正确（DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD）
3. 检查数据库用户权限

## 后续扩展

如果需要为其他API添加认证保护，可以实现FastAPI的依赖项：

```python
from fastapi import Depends, HTTPException
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from core.auth import decode_access_token

security = HTTPBearer()

async def get_current_user(credentials: HTTPAuthorizationCredentials = Depends(security)):
    token = credentials.credentials
    payload = decode_access_token(token)
    if payload is None:
        raise HTTPException(status_code=401, detail="无效的token")
    return payload

# 在需要认证的路由中使用
@app.get("/protected")
async def protected_route(current_user: dict = Depends(get_current_user)):
    return {"message": f"Hello, {current_user['username']}!"}
```

## 联系方式

如有问题，请查看服务端日志或联系开发团队。
