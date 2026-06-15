#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:6464}"
WEB_BASE="${WEB_BASE:-http://localhost:6465}"
SOURCE_ID="${SOURCE_ID:-6260}"
RSS_FEED_ID="${RSS_FEED_ID:-57}"
TAB="${TAB:-科幻}"

echo "[1/5] Health"
curl -fsS "$API_BASE/health"
echo

echo "[2/5] Web health"
curl -fsS "$WEB_BASE/health" >/dev/null || true
echo "web ok"

echo "[3/5] Explore"
ENCODED_TAB="$(TAB_VALUE="$TAB" python3 - <<'PY'
import os, urllib.parse
print(urllib.parse.quote(os.environ["TAB_VALUE"]))
PY
)"
curl -fsS "$API_BASE/api/explore?sourceId=$SOURCE_ID&tab=$ENCODED_TAB&pageSize=5" | head -c 600
echo

echo "[4/5] RSS fetch"
curl -fsS -X POST "$API_BASE/api/rss/feeds/$RSS_FEED_ID/fetch" | head -c 600
echo

echo "[5/5] Browser binary"
docker exec reader-go-api sh -lc 'command -v chromium-browser || command -v chromium'
