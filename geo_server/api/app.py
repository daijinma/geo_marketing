"""
FastAPI application initialization
"""
import os
from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from api.routes import tasks, auth, export, health

# 创建FastAPI应用
app = FastAPI(
    title="LLM Sentry Monitor API",
    description="GEO 品牌曝光监测系统 - 任务管理 API",
    version="1.0.0"
)

# 注册路由
app.include_router(tasks.router)
app.include_router(auth.router)
app.include_router(export.router)
app.include_router(health.router)

# 静态文件服务
static_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), "static")
if os.path.exists(static_dir):
    app.mount("/static", StaticFiles(directory=static_dir), name="static")
