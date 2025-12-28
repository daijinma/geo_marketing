#!/bin/bash
# 使用 uv 运行监控任务
echo "正在启动 GEO 监测任务 (via uv)..."
cd llm_sentry_monitor
uv run main.py
