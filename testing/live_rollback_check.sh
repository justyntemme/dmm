#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
REQUIRE_RESTORE_AVAILABLE="${REQUIRE_RESTORE_AVAILABLE:-1}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live rollback check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "require_restore_available=${REQUIRE_RESTORE_AVAILABLE}"

export BASE_URL
export APP_ID
export REQUIRE_RESTORE_AVAILABLE

python3 - <<'PY'
import json
import os
import urllib.error
import urllib.request


base = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ["APP_ID"]
require_restore_available = os.environ.get("REQUIRE_RESTORE_AVAILABLE", "1") != "0"


def request(method, path):
    req = urllib.request.Request(base + path, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            if not raw:
                return {}
            return json.loads(raw)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path} failed: {exc.code} {detail}") from exc


status = request("GET", f"/api/games/{app_id}/deploy/status")
if require_restore_available and not status.get("restore_available"):
    raise RuntimeError(f"restore is not available: {status}")

result = request("POST", f"/api/games/{app_id}/deploy/restore")
job = result.get("job") or {}
restore = result.get("result") or {}
issues = restore.get("issues") or []
repaired = restore.get("repaired") or []
if job.get("type") != "rollback":
    raise RuntimeError(f"restore returned non-rollback job: {job}")
if job.get("status") != "completed":
    raise RuntimeError(f"rollback job did not complete: {job}")
if issues:
    raise RuntimeError(f"rollback reported issues: {issues}")

after = request("GET", f"/api/games/{app_id}/deploy/status")
if not after.get("deployed"):
    raise RuntimeError(f"deployment missing after rollback: {after}")

print("summary:")
print(f"  strategy={after.get('strategy')}")
print(f"  files={after.get('file_count')}")
print(f"  repaired={len(repaired)}")
print(f"  job={job.get('id')}")
print("\nRollback check passed")
PY
