#!/bin/bash

# ============================================
# LLM Sentry Monitor API - cURL 示例
# ============================================

API_BASE_URL="http://localhost:8000"

echo "============================================"
echo "示例 1: 创建新任务 (POST /mock)"
echo "============================================"
echo ""

# 基本示例：只指定关键词和平台
curl -X POST "http://localhost:8000/mock" \
  -H "Content-Type: application/json" \
  -d '{
    "keywords": ["土巴兔装修靠谱嘛", "装修公司推荐"],
    "platforms": ["deepseek"]
  }'

echo ""
echo ""
echo "============================================"
echo "示例 2: 创建任务（带自定义设置）"
echo "============================================"
echo ""

# 完整示例：包含自定义设置
curl -X POST "http://localhost:8000/mock" \
  -H "Content-Type: application/json" \
  -d '{
    "keywords": ["北京本地装修公司推荐"],
    "platforms": ["deepseek", "doubao"]
  }'

echo ""
echo ""
echo "============================================"
echo "示例 3: 查询任务状态 (GET /status)"
echo "============================================"
echo ""

# 查询任务状态（将 TASK_ID 替换为实际的任务ID）
TASK_ID=1
curl -X GET "http://localhost:8000/status?id=${TASK_ID}"

echo ""
echo ""
echo "============================================"
echo "示例 4: 格式化输出（使用 jq）"
echo "============================================"
echo ""

# 如果安装了 jq，可以使用它来格式化 JSON 输出
# curl -X GET "${API_BASE_URL}/status?id=${TASK_ID}" | jq '.'

echo ""
echo "============================================"
echo "使用说明："
echo "============================================"
echo "1. 确保 API 服务器已启动: make dev"
echo "2. 运行 POST /mock 创建任务，会返回 task_id"
echo "3. 使用返回的 task_id 查询任务状态"
echo "4. 任务状态: none (不存在), pending (执行中), done (已完成)"
echo ""

