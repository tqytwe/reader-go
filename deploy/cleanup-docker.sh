#!/usr/bin/env bash
set -euo pipefail

echo "[1/2] Remove dangling images"
docker image prune -f

echo "[2/2] Remove build cache"
docker builder prune -f
