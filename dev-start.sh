#!/usr/bin/env bash
set -e

echo "🚀 启动 Reader Go 本地开发环境"
echo "================================"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "❌ 未找到 Go，请先安装 Go"
    exit 1
fi

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "❌ 未找到 Node.js，请先安装 Node.js"
    exit 1
fi

echo -e "${BLUE}✓ 开发工具检查通过${NC}"
echo "  Go: $(go version)"
echo "  Node: $(node --version)"
echo ""

# 创建数据目录
mkdir -p data

# 启动后端
echo -e "${GREEN}[1/2] 启动后端 API 服务 (端口 6464)${NC}"
CGO_ENABLED=1 go run cmd/server/main.go &
BACKEND_PID=$!

# 等待后端启动
sleep 2

# 检查后端是否启动成功
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo "❌ 后端启动失败"
    exit 1
fi

echo -e "${GREEN}✓ 后端已启动 (PID: $BACKEND_PID)${NC}"

# 启动前端
echo -e "${GREEN}[2/2] 启动前端开发服务器 (端口 5173)${NC}"
cd web
npm run dev &
FRONTEND_PID=$!

echo -e "${GREEN}✓ 前端已启动 (PID: $FRONTEND_PID)${NC}"
echo ""
echo "================================"
echo -e "${BLUE}🎉 开发环境启动完成！${NC}"
echo ""
echo "  前端: http://localhost:5173"
echo "  后端: http://localhost:6464"
echo "  API 文档: http://localhost:6464/health"
echo ""
echo "按 Ctrl+C 停止所有服务"
echo "================================"

# 捕获退出信号
cleanup() {
    echo ""
    echo "🛑 正在停止服务..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    echo "✓ 服务已停止"
    exit 0
}

trap cleanup SIGINT SIGTERM

# 等待
wait
