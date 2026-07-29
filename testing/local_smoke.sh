#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PORT:-17959}"
TMP_ROOT="${TMP_ROOT:-$(mktemp -d)}"
CONFIG_HOME="${TMP_ROOT}/config"
DATA_DIR="${TMP_ROOT}/data"
LOG_FILE="${TMP_ROOT}/server.log"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  if [[ -z "${KEEP_TMP:-}" ]]; then
    rm -rf "${TMP_ROOT}"
  else
    echo "Kept smoke directory: ${TMP_ROOT}"
  fi
}
trap cleanup EXIT

require_contains() {
  local body="$1"
  local needle="$2"
  local label="$3"
  if [[ "${body}" != *"${needle}"* ]]; then
    echo "Smoke check failed for ${label}: expected ${needle}" >&2
    echo "${body}" >&2
    exit 1
  fi
}

assert_web_assets_served() {
  local index_file="$1"
  python3 - "${index_file}" <<'PY'
import re
import sys

body = open(sys.argv[1], encoding="utf-8").read()
assets = sorted(set(re.findall(r'/(assets/[^"\'<> ]+)', body)))
if not assets:
    print("web index did not reference built assets", file=sys.stderr)
    sys.exit(1)
for asset in assets:
    print(asset)
PY
}

mkdir -p "${CONFIG_HOME}/decky-mod-manager" "${DATA_DIR}"

cat > "${CONFIG_HOME}/decky-mod-manager/config.json" <<JSON
{
  "listen_addr": "127.0.0.1:${PORT}",
  "lan_only": true,
  "data_dir": "${DATA_DIR}",
  "nexus": {},
  "install": {}
}
JSON

echo "==> Building native backend"
(
  cd "${ROOT_DIR}"
  go build -o bin/dmm-server ./cmd/dmm-server
)

echo "==> Starting backend on 127.0.0.1:${PORT}"
XDG_CONFIG_HOME="${CONFIG_HOME}" "${ROOT_DIR}/bin/dmm-server" >"${LOG_FILE}" 2>&1 &
SERVER_PID="$!"

for _ in {1..50}; do
  if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

health="$(curl -fsS "http://127.0.0.1:${PORT}/api/health")"
require_contains "${health}" '"ok":true' "health"

status="$(curl -fsS "http://127.0.0.1:${PORT}/api/status")"
require_contains "${status}" '"lan_only":true' "status lan_only"
require_contains "${status}" '"auto_deploy":false' "status auto_deploy default"

jobs="$(curl -fsS "http://127.0.0.1:${PORT}/api/jobs")"
require_contains "${jobs}" '[]' "empty jobs"

deps="$(curl -fsS "http://127.0.0.1:${PORT}/api/dependencies")"
require_contains "${deps}" '"name"' "dependencies"

index="$(curl -fsS "http://127.0.0.1:${PORT}/")"
require_contains "${index}" '<div id="app"></div>' "web index"
printf '%s' "${index}" >"${TMP_ROOT}/index.html"
while IFS= read -r asset; do
  body="$(curl -fsS "http://127.0.0.1:${PORT}/${asset}")"
  if [[ -z "${body}" ]]; then
    echo "Smoke check failed for web asset ${asset}: empty response" >&2
    exit 1
  fi
done < <(assert_web_assets_served "${TMP_ROOT}/index.html")

echo "==> Local smoke passed"
