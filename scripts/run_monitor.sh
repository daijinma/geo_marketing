#!/bin/bash
# 使用 uv 运行监测任务
echo "正在启动 GEO 监测任务 (via uv)..."
cd geo_server
uv run main.py
