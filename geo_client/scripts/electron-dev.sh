#!/bin/bash

# Electron 开发脚本
# 同时启动 Vite 开发服务器和 Electron

# 检查是否安装了依赖
if [ ! -d "node_modules" ]; then
  echo "未找到 node_modules，正在安装依赖..."
  npm install
fi

# 启动 Vite 开发服务器（后台运行）
echo "启动 Vite 开发服务器..."
npm run dev &
VITE_PID=$!

# 等待 Vite 服务器启动
echo "等待 Vite 服务器启动..."
sleep 3

# 启动 Electron
echo "启动 Electron..."
npm run electron:dev

# 当 Electron 退出时，也退出 Vite
kill $VITE_PID 2>/dev/null
