#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "[1/4] Build API image"
docker build --network=host -t reader-go-api:latest .

echo "[2/4] Recreate containers"
docker compose up -d --force-recreate reader-go-api reader-go-web

echo "[3/4] Wait for API health"
for _ in $(seq 1 60); do
  status="$(docker inspect -f '{{.State.Health.Status}}' reader-go-api 2>/dev/null || true)"
  if [ "$status" = "healthy" ]; then
    break
  fi
  sleep 2
done

status="$(docker inspect -f '{{.State.Health.Status}}' reader-go-api 2>/dev/null || true)"
if [ "$status" != "healthy" ]; then
  echo "reader-go-api is not healthy"
  docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
  exit 1
fi

echo "[4/4] Current containers"
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | grep reader-go || true
