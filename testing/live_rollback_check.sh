#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
REQUIRE_RESTORE_AVAILABLE="${REQUIRE_RESTORE_AVAILABLE:-1}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

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

from dmm_test_auth import auth_headers


base = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ["APP_ID"]
require_restore_available = os.environ.get("REQUIRE_RESTORE_AVAILABLE", "1") != "0"


def request(method, path):
    req = urllib.request.Request(base + path, headers=auth_headers(), method=method)
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

history = request("GET", f"/api/games/{app_id}/deploy/history?limit=10")
active = next(
    (deployment for deployment in history.get("deployments", []) if deployment.get("active")),
    None,
)
if active is None:
    raise RuntimeError(f"active deployment point is missing: {history}")

deployment_id = active.get("id")
preview = request(
    "GET",
    f"/api/games/{app_id}/deploy/history/{deployment_id}/preview",
)
if preview.get("deployment_id") != deployment_id:
    raise RuntimeError(f"restore preview identity mismatch: {preview}")
if preview.get("current_file_count") != preview.get("target_file_count"):
    raise RuntimeError(f"active deployment preview changes file count: {preview}")

result = request(
    "POST",
    f"/api/games/{app_id}/deploy/history/{deployment_id}/restore",
)
job = result.get("job") or {}
if job.get("type") != "rollback":
    raise RuntimeError(f"restore returned non-rollback job: {job}")
if job.get("status") != "completed":
    raise RuntimeError(f"rollback job did not complete: {job}")

after = request("GET", f"/api/games/{app_id}/deploy/status")
if not after.get("deployed"):
    raise RuntimeError(f"deployment missing after rollback: {after}")
if after.get("file_count") != status.get("file_count"):
    raise RuntimeError(f"rollback changed the active deployment file count: before={status} after={after}")

print("summary:")
print(f"  strategy={after.get('strategy')}")
print(f"  files={after.get('file_count')}")
print(f"  restored_point={deployment_id}")
print(f"  changes={len((preview.get('plan') or {}).get('actions') or [])}")
print(f"  job={job.get('id')}")
print("\nRollback check passed")
PY
