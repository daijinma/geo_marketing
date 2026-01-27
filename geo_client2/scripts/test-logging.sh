#!/bin/bash

# 日志系统测试脚本

echo "🧪 开始测试日志系统..."
echo ""

# 1. 检查前端文件
echo "1️⃣ 检查前端文件..."
if [ -f "frontend/src/pages/Logs.tsx" ]; then
    echo "   ✅ Logs.tsx 存在"
else
    echo "   ❌ Logs.tsx 不存在"
fi

if [ -f "frontend/src/utils/logger.ts" ]; then
    echo "   ✅ logger.ts 存在"
else
    echo "   ❌ logger.ts 不存在"
fi

if [ -f "frontend/src/components/ErrorBoundary.tsx" ]; then
    echo "   ✅ ErrorBoundary.tsx 存在"
else
    echo "   ❌ ErrorBoundary.tsx 不存在"
fi

echo ""

# 2. 检查后端文件
echo "2️⃣ 检查后端文件..."
if [ -f "backend/logger/logger.go" ]; then
    echo "   ✅ logger.go 存在"
else
    echo "   ❌ logger.go 不存在"
fi

if [ -f "backend/database/repositories/log.go" ]; then
    echo "   ✅ log.go 存在"
else
    echo "   ❌ log.go 不存在"
fi

echo ""

# 3. 检查文档
echo "3️⃣ 检查文档..."
if [ -f "LOGGING.md" ]; then
    echo "   ✅ LOGGING.md 存在"
else
    echo "   ❌ LOGGING.md 不存在"
fi

if [ -f "docs/LOG_VIEWER_UI.md" ]; then
    echo "   ✅ LOG_VIEWER_UI.md 存在"
else
    echo "   ❌ LOG_VIEWER_UI.md 不存在"
fi

echo ""

# 4. 检查数据库迁移
echo "4️⃣ 检查数据库架构..."
if grep -q "session_id TEXT" backend/database/schema.go; then
    echo "   ✅ session_id 字段已添加"
else
    echo "   ❌ session_id 字段未添加"
fi

if grep -q "correlation_id TEXT" backend/database/schema.go; then
    echo "   ✅ correlation_id 字段已添加"
else
    echo "   ❌ correlation_id 字段未添加"
fi

if grep -q "performance_ms INTEGER" backend/database/schema.go; then
    echo "   ✅ performance_ms 字段已添加"
else
    echo "   ❌ performance_ms 字段未添加"
fi

echo ""

# 5. 检查 API 绑定
echo "5️⃣ 检查 Wails API 绑定..."
if grep -q "GetLogs" backend/app.go; then
    echo "   ✅ GetLogs 方法存在"
else
    echo "   ❌ GetLogs 方法不存在"
fi

if grep -q "AddLog" backend/app.go; then
    echo "   ✅ AddLog 方法存在"
else
    echo "   ❌ AddLog 方法不存在"
fi

echo ""

# 6. 检查路由
echo "6️⃣ 检查前端路由..."
if grep -q "path=\"logs\"" frontend/src/App.tsx; then
    echo "   ✅ /logs 路由已配置"
else
    echo "   ❌ /logs 路由未配置"
fi

echo ""
echo "========================================="
echo "✨ 日志系统测试完成！"
echo ""
echo "📝 访问方式："
echo "   1. 启动应用: make dev"
echo "   2. 在侧边栏点击「日志列表」"
echo "   3. 或访问: http://localhost:34115/logs"
echo ""
echo "📚 查看文档："
echo "   - 开发者文档: cat LOGGING.md"
echo "   - UI使用说明: cat docs/LOG_VIEWER_UI.md"
echo "========================================="
