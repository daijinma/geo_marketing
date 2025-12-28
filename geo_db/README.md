# LLM Sentry Database Service

本仓库是 `LLM Sentry` 项目的数据库服务模块，采用独立容器化部署。

## 1. 目录结构
```text
geo_db/
├── docker-compose.yml   # 容器编排配置
├── init.sql             # 数据库初始化脚本 (表结构定义)
└── README.md            # 本说明文件
```

## 2. 快速启动
确保本地已安装 Docker 和 Docker Compose。

```bash
cd geo_db
docker-compose up -d
```

## 3. 数据库配置
- **类型**: PostgreSQL 15
- **端口**: 5432
- **用户名**: `geo_admin`
- **密码**: `geo_password123`
- **数据库名**: `geo_monitor`

## 4. 数据持久化
数据存储在本地目录 `./postgres_data` 中，即使容器销毁，数据也会保留。
