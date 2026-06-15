#!/usr/bin/env bash
set -e

echo "🚀 启动前端开发服务器"
echo "==================="

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT/web"

echo "工作目录: $(pwd)"
echo ""

# 检查依赖
if [ ! -d "node_modules" ]; then
    echo "安装依赖..."
    npm install
fi

echo "启动中..."
npm run dev
