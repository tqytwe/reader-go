#!/bin/bash
# 安装 Reader Go systemd 服务（非 Docker 部署）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SERVICE_FILE="$ROOT/deploy/reader-go-api.service"

if [[ $EUID -ne 0 ]]; then
  echo "请使用 sudo 运行: sudo bash $0"
  exit 1
fi

# 确保数据目录存在
mkdir -p "$ROOT/data/cache" "$ROOT/data/localbooks"
chmod 755 "$ROOT/data"

# 安装 systemd unit
cp "$SERVICE_FILE" /etc/systemd/system/reader-go-api.service
systemctl daemon-reload
systemctl enable reader-go-api
systemctl restart reader-go-api

echo "服务已安装。状态："
systemctl status reader-go-api --no-pager || true
echo ""
echo "API: http://127.0.0.1:6464/health"
echo "如需前端，请配置 nginx 指向 $ROOT/web/dist 并代理 /api 到 6464"
