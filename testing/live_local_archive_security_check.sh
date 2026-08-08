#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
APP_ID="${APP_ID:-413150}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

python3 - "${BASE_URL}" "${APP_ID}" <<'PY'
import json
import sys
import urllib.error
import urllib.parse
import urllib.request

from dmm_test_auth import auth_headers

base_url = sys.argv[1].rstrip("/")
app_id = sys.argv[2]


def request(path, token=True):
    headers = auth_headers() if token else {}
    req = urllib.request.Request(base_url + path, headers=headers, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=20) as response:
            payload = response.read()
            return response.status, payload.decode("utf-8", "replace")
    except urllib.error.HTTPError as err:
        return err.code, err.read().decode("utf-8", "replace")


def expect_status(path, status, token=True):
    got_status, body = request(path, token=token)
    if got_status != status:
        raise RuntimeError(f"{path} returned {got_status}, want {status}: {body}")
    return body


expect_status("/api/games", 401, token=False)

body = expect_status(f"/api/games/{app_id}/local-archives/browse", 200)
payload = json.loads(body)
roots = payload.get("roots") or []
if not roots:
    raise RuntimeError("local archive browse returned no approved roots")
if not any(root.endswith("/Downloads") or "/Downloads/" in root for root in roots):
    raise RuntimeError(f"approved roots do not include Deck Downloads: {roots}")

for bad_path in ["/", "/tmp", ".."]:
    encoded = urllib.parse.quote(bad_path, safe="")
    body = expect_status(f"/api/games/{app_id}/local-archives/browse?path={encoded}", 400)
    if "outside the allowed Deck download folders" not in body and bad_path != "/":
        raise RuntimeError(f"unexpected rejection body for {bad_path!r}: {body}")

print("local archive security check passed")
print(f"roots={len(roots)} current={payload.get('current_path')}")
PY
