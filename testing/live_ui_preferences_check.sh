#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
SECOND_APP_ID="${SECOND_APP_ID:-377160}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live UI preferences check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "second_app_id=${SECOND_APP_ID}"
export BASE_URL APP_ID SECOND_APP_ID

python3 - <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.request

from dmm_test_auth import auth_headers

base_url = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ.get("APP_ID", "413150")
second_app_id = os.environ.get("SECOND_APP_ID", "377160")


def request(method, path, payload=None):
    data = None
    headers = {}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base_url + path, data=data, headers=auth_headers(headers), method=method)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path} failed: HTTP {exc.code}: {detail}") from exc
    if not body:
        return None
    return json.loads(body.decode("utf-8"))


status = request("GET", "/api/status")
original = status.get("ui") or {}
original_favorites = list(original.get("favorite_game_ids") or [])
original_recents = dict(original.get("recent_games") or {})
original_sort = original.get("game_sort") or "recent"

try:
    now = int(time.time() * 1000)
    request("PATCH", "/api/settings/ui", {
        "favorite_game_id": app_id,
        "favorite": True,
        "recent_game_id": app_id,
        "recent_at": now,
        "game_sort": "az",
    })
    request("PATCH", "/api/settings/ui", {
        "favorite_game_id": second_app_id,
        "favorite": True,
        "recent_game_id": second_app_id,
        "recent_at": now + 1,
        "game_sort": "za",
    })
    patched = request("GET", "/api/settings/ui")
    favorites = set(patched.get("favorite_game_ids") or [])
    recents = patched.get("recent_games") or {}
    if app_id not in favorites or second_app_id not in favorites:
        raise RuntimeError(f"favorites did not persist intent patches: {patched}")
    if int(recents.get(app_id, 0)) != now or int(recents.get(second_app_id, 0)) != now + 1:
        raise RuntimeError(f"recent-game values did not persist intent patches: {patched}")
    if patched.get("game_sort") != "za":
        raise RuntimeError(f"game_sort did not use the latest patch: {patched}")
    print("summary:")
    print("  favorites_include=", ",".join(sorted([app_id, second_app_id])), sep="")
    print("  game_sort=", patched.get("game_sort"), sep="")
    print("  recent_latest=", second_app_id, sep="")
finally:
    restored_favorites = set(original_favorites)
    for target in (app_id, second_app_id):
        request("PATCH", "/api/settings/ui", {
            "favorite_game_id": target,
            "favorite": target in restored_favorites,
        })
    request("PATCH", "/api/settings/ui", {"game_sort": original_sort})
    for target in (app_id, second_app_id):
        if target in original_recents:
            request("PATCH", "/api/settings/ui", {
                "recent_game_id": target,
                "recent": True,
                "recent_at": int(original_recents[target]),
            })
        else:
            request("PATCH", "/api/settings/ui", {
                "recent_game_id": target,
                "recent": False,
            })

print("UI preferences API source-of-truth check passed")
PY
