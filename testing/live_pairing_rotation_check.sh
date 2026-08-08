#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
STATE_DIR="${STATE_DIR:-${HOME}/.local/state/decky-mod-manager}"
TOKEN_FILE="${DMM_TOKEN_FILE:-${STATE_DIR}/api-token}"
PLUGIN_MAIN="${PLUGIN_MAIN:-/home/deck/homebrew/plugins/decky-mod-manager/main.py}"

backend_binary() {
  local plugin_dir
  plugin_dir="$(cd "$(dirname "${PLUGIN_MAIN}")" && pwd)"
  printf '%s\n' "${plugin_dir}/bin/dmm-server"
}

start_backend_for_test_harness() {
  local plugin_dir
  plugin_dir="$(cd "$(dirname "${PLUGIN_MAIN}")" && pwd)"
  local binary="${plugin_dir}/bin/dmm-server"
  if [[ ! -x "${binary}" || ! -f "${TOKEN_FILE}" ]]; then
    return 1
  fi
  local token
  token="$(<"${TOKEN_FILE}")"
  if ! command -v systemd-run >/dev/null 2>&1; then
    echo "backend is down and systemd-run is not available for test repair" >&2
    return 1
  fi
  systemctl --user stop dmm-test-server.service >/dev/null 2>&1 || true
  systemctl --user reset-failed dmm-test-server.service >/dev/null 2>&1 || true
  systemd-run --user \
    --unit=dmm-test-server \
    --collect \
    --property="WorkingDirectory=${plugin_dir}" \
    --setenv="DMM_AUTH_TOKEN=${token}" \
    --setenv="DMM_AUTH_TOKEN_FILE=${TOKEN_FILE}" \
    --setenv="DMM_DECKY_PLUGIN_DIR=${plugin_dir}" \
    "${binary}" >/dev/null
  for _ in $(seq 1 20); do
    if curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
      echo "standalone rotation harness restored backend through user systemd"
      return 0
    fi
    sleep 0.5
  done
  return 1
}

repair_backend_if_needed() {
  if curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
    return 0
  fi
  start_backend_for_test_harness
}

restart_backend_for_test_harness() {
  local binary
  binary="$(backend_binary)"
  pkill -TERM -f "^${binary}$" >/dev/null 2>&1 || true
  for _ in $(seq 1 20); do
    if ! curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
      break
    fi
    sleep 0.25
  done
  start_backend_for_test_harness
}

repair_backend_if_needed

python3 - "${BASE_URL}" "${TOKEN_FILE}" "${PLUGIN_MAIN}" <<'PY'
import asyncio
import importlib.util
import json
import os
import stat
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

base_url = sys.argv[1].rstrip("/")
token_file = Path(sys.argv[2]).expanduser()
plugin_main = Path(sys.argv[3]).expanduser()


def read_token() -> str:
    return token_file.read_text(encoding="utf-8").strip()


def request_status(token: str) -> tuple[int, str]:
    req = urllib.request.Request(
        base_url + "/api/status",
        headers={"X-DMM-Token": token} if token else {},
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as response:
            return response.status, response.read().decode("utf-8", "replace")
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode("utf-8", "replace")


def wait_for_health() -> None:
    for _ in range(60):
        try:
            with urllib.request.urlopen(base_url + "/api/health", timeout=2) as response:
                if response.status == 200:
                    return
        except Exception:
            pass
        time.sleep(0.5)
    raise RuntimeError("backend did not become healthy after pairing rotation")


old_token = read_token()
if len(old_token) < 24:
    raise RuntimeError("current token is missing or too short")

status, body = request_status(old_token)
if status != 200:
    raise RuntimeError(f"old token did not work before rotation: HTTP {status}: {body[:500]}")

if not plugin_main.exists():
    raise RuntimeError(f"Decky plugin bridge is missing: {plugin_main}")

spec = importlib.util.spec_from_file_location("dmm_decky_plugin", plugin_main)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)

result = asyncio.run(module.Plugin().reset_api_token())
if not isinstance(result, dict):
    raise RuntimeError(f"reset_api_token returned non-dict result: {result!r}")
if not result.get("running"):
    raise RuntimeError(f"backend was not running after token reset: {json.dumps(result, default=str)[:1000]}")

wait_for_health()
new_token = read_token()
if len(new_token) < 24:
    raise RuntimeError("new token is missing or too short")
if new_token == old_token:
    raise RuntimeError("pairing token did not rotate")

old_status, _ = request_status(old_token)
if old_status != 401:
    raise RuntimeError(f"old token returned HTTP {old_status}, want 401 after rotation")

new_status, new_body = request_status(new_token)
if new_status != 200:
    raise RuntimeError(f"new token returned HTTP {new_status}, want 200: {new_body[:500]}")

mode = stat.S_IMODE(token_file.stat().st_mode)
if mode != 0o600:
    raise RuntimeError(f"token file mode is {mode:o}, want 600")

print("pairing rotation check passed")
print(f"token_file={token_file}")
print("old_token=rejected new_token=accepted mode=600")
PY

# The real Decky UI method keeps ownership of the restarted backend. This script
# imports the bridge in a one-off process, so hand backend ownership to a
# transient user service after verification instead of leaving a child process
# tied to the short-lived test interpreter.
restart_backend_for_test_harness
