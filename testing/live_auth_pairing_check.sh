#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

python3 - "${BASE_URL}" <<'PY'
import json
import sys
import urllib.error
import urllib.parse
import urllib.request

from dmm_test_auth import token

base_url = sys.argv[1].rstrip("/")
api_token = token()
if not api_token:
    raise RuntimeError("DMM_AUTH_TOKEN or DMM_TOKEN_FILE is required for the live auth check")


def request(path, headers=None):
    req = urllib.request.Request(base_url + path, headers=headers or {}, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status, response.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as err:
        return err.code, err.read().decode("utf-8", "replace")


def expect(path, status, headers=None):
    got, body = request(path, headers=headers)
    if got != status:
        raise RuntimeError(f"{path} returned {got}, want {status}: {body[:500]}")
    return body


expect("/api/health", 200)
expect("/", 200)
expect("/api/status", 401)
expect("/api/status", 401, {"X-DMM-Token": "wrong-token"})

encoded = urllib.parse.quote(api_token, safe="")
expect(f"/api/status?token={encoded}", 401)
status_body = expect("/api/status", 200, {"X-DMM-Token": api_token})
status = json.loads(status_body)
if not status.get("auth", {}).get("enabled"):
    raise RuntimeError(f"/api/status did not report auth enabled: {status.get('auth')}")
if "token" in status.get("auth", {}):
    raise RuntimeError("/api/status must not expose the runtime token")

ws_status, ws_body = request(f"/api/events/ws?token={encoded}")
if ws_status not in (400, 426):
    raise RuntimeError(f"websocket token route returned {ws_status}, want handshake boundary status: {ws_body[:500]}")
bad_ws_status, _ = request("/api/events/ws?token=wrong-token")
if bad_ws_status != 401:
    raise RuntimeError(f"websocket wrong-token route returned {bad_ws_status}, want 401")

print("auth pairing check passed")
print("rest_query_token=rejected websocket_query_token=accepted_for_upgrade")
PY
