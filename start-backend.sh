#!/usr/bin/env bash
set -e

echo "🚀 启动后端 API 服务"
echo "==================="

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# 设置环境变量
export PORT=${PORT:-6464}
export DATABASE_URL=${DATABASE_URL:-./data/reader.db}
export CGO_ENABLED=1

echo "端口: $PORT"
echo "数据库: $DATABASE_URL"
echo ""

# 创建数据目录
mkdir -p data

echo "启动中..."
go run cmd/server/main.go
