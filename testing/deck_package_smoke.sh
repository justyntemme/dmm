#!/usr/bin/env bash
set -euo pipefail

APP_ID="${APP_ID:-413150}"
PORT="${PORT:-17955}"
PACKAGE="${PACKAGE:-${HOME}/.testing/decky-mod-manager.tar.gz}"
TMP_ROOT="${TMP_ROOT:-/tmp/dmm-package-smoke}"
SHAPE_ONLY="${SHAPE_ONLY:-}"
SERVER_PID=""

CONFIG_HOME="${TMP_ROOT}/config"
DATA_DIR="${TMP_ROOT}/data"
PACKAGE_DIR="${TMP_ROOT}/package"
LOG_FILE="${TMP_ROOT}/server.log"
BASE_URL="http://127.0.0.1:${PORT}"

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  if [[ -z "${KEEP_TMP:-}" ]]; then
    rm -rf "${TMP_ROOT}"
  else
    echo "Kept package smoke directory: ${TMP_ROOT}"
  fi
}
trap cleanup EXIT

section() {
  printf '\n==> %s\n' "$1"
}

wait_for_server() {
  for _ in $(seq 1 60); do
    if curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "server did not become healthy" >&2
  tail -n 80 "${LOG_FILE}" >&2 || true
  exit 1
}

require_contains() {
  local body="$1"
  local needle="$2"
  local label="$3"
  if [[ "${body}" != *"${needle}"* ]]; then
    echo "package smoke failed for ${label}: expected ${needle}" >&2
    echo "${body}" >&2
    exit 1
  fi
}

require_file_contains() {
  local file="$1"
  local needle="$2"
  local label="$3"
  if ! grep -qF "${needle}" "${file}"; then
    echo "package smoke failed for ${label}: expected ${needle} in ${file}" >&2
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

if [[ ! -f "${PACKAGE}" ]]; then
  echo "package not found: ${PACKAGE}" >&2
  exit 1
fi

section "Preparing package smoke"
rm -rf "${TMP_ROOT}"
mkdir -p "${CONFIG_HOME}/decky-mod-manager" "${DATA_DIR}" "${PACKAGE_DIR}"
tar -xzf "${PACKAGE}" -C "${PACKAGE_DIR}"

if [[ ! -x "${PACKAGE_DIR}/decky-mod-manager/bin/dmm-server" ]]; then
  echo "package is missing bin/dmm-server" >&2
  exit 1
fi
if [[ ! -x "${PACKAGE_DIR}/decky-mod-manager/bin/dmm-nxm-handler" ]]; then
  echo "package is missing bin/dmm-nxm-handler" >&2
  exit 1
fi
if [[ "${REQUIRE_LOOT_SORTER:-0}" == "1" && ! -x "${PACKAGE_DIR}/decky-mod-manager/bin/dmm-loot-sorter" ]]; then
  echo "package is missing required bin/dmm-loot-sorter" >&2
  exit 1
fi
if [[ ! -f "${PACKAGE_DIR}/decky-mod-manager/web/dist/index.html" ]]; then
  echo "package is missing bundled web UI" >&2
  exit 1
fi
if [[ ! -f "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" ]]; then
  echo "package is missing bundled Decky UI" >&2
  exit 1
fi
if [[ ! -f "${PACKAGE_DIR}/decky-mod-manager/build-info.json" ]]; then
  echo "package is missing build-info.json" >&2
  exit 1
fi

section "Checking packaged UI content"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Games" "Decky games tab"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Actions" "Decky actions tab"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Settings" "Decky settings tab"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Pair Phone" "Decky phone pairing UI"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Auto-install downloaded mods" "Decky auto-install setting"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Auto-enable installed mods" "Decky auto-enable setting"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Build:" "Decky build metadata UI"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Refresh Debug" "Decky debug refresh action"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/dist/index.js" "Debug Tools" "Decky debug tools"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/build-info.json" "short_commit" "build metadata"
require_file_contains "${PACKAGE_DIR}/decky-mod-manager/web/dist/index.html" "Decky Mod Manager" "web index title"
for asset_file in "${PACKAGE_DIR}/decky-mod-manager"/web/dist/assets/*.js; do
  require_file_contains "${asset_file}" "Selected Profile" "web profile-first UI"
  require_file_contains "${asset_file}" "Installed, disabled in this profile" "web profile-scoped mod state"
  require_file_contains "${asset_file}" "All Installed" "web supported game filter"
  require_file_contains "${asset_file}" "Explore Mods" "web in-game source browsing"
  require_file_contains "${asset_file}" "Open a result on the Deck" "web in-game Nexus source explanation"
  require_file_contains "${asset_file}" "Open on Deck" "web browser-first Nexus result action"
  require_file_contains "${asset_file}" "Advanced Profile Tools" "web advanced profile disclosure"
  require_file_contains "${asset_file}" "These Deck behavior switches are managed from the Decky sidebar settings." "web install settings Decky ownership note"
done

if [[ -n "${SHAPE_ONLY}" ]]; then
  section "Package shape passed"
  echo "server=${PACKAGE_DIR}/decky-mod-manager/bin/dmm-server"
  echo "nxm_handler=${PACKAGE_DIR}/decky-mod-manager/bin/dmm-nxm-handler"
  echo "web=${PACKAGE_DIR}/decky-mod-manager/web/dist/index.html"
  exit 0
fi

cat > "${CONFIG_HOME}/decky-mod-manager/config.json" <<JSON
{
  "listen_addr": "127.0.0.1:${PORT}",
  "lan_only": false,
  "data_dir": "${DATA_DIR}",
  "install": {
    "auto_install_captured_downloads": true,
    "auto_enable_installed_mods": false
  }
}
JSON

section "Starting packaged backend"
XDG_CONFIG_HOME="${CONFIG_HOME}" XDG_DATA_HOME="${TMP_ROOT}/xdg-data" \
  "${PACKAGE_DIR}/decky-mod-manager/bin/dmm-server" >"${LOG_FILE}" 2>&1 &
SERVER_PID="$!"
wait_for_server

section "Checking API and web assets"
health="$(curl -fsS "${BASE_URL}/api/health")"
echo "${health}"
require_contains "${health}" '"ok":true' "health"

status="$(curl -fsS "${BASE_URL}/api/status")"
require_contains "${status}" '"lan_only":false' "status lan_only"
require_contains "${status}" '"auto_install_captured_downloads":true' "status auto_install_captured_downloads"
require_contains "${status}" '"auto_enable_installed_mods":false' "status auto_enable_installed_mods"

index="$(curl -fsS "${BASE_URL}/")"
require_contains "${index}" "Decky Mod Manager" "web index title"
require_contains "${index}" '<div id="app"></div>' "web app mount"
printf '%s' "${index}" >"${TMP_ROOT}/index.html"
while IFS= read -r asset; do
  body="$(curl -fsS "${BASE_URL}/${asset}")"
  if [[ -z "${body}" ]]; then
    echo "package smoke failed for web asset ${asset}: empty response" >&2
    exit 1
  fi
done < <(assert_web_assets_served "${TMP_ROOT}/index.html")

diag_code="$(curl -sS -o "${TMP_ROOT}/diagnostics.json" -w "%{http_code}" "${BASE_URL}/api/games/${APP_ID}/diagnostics")"
if [[ "${diag_code}" != "200" && "${diag_code}" != "404" ]]; then
  echo "unexpected diagnostics status: ${diag_code}" >&2
  cat "${TMP_ROOT}/diagnostics.json" >&2
  exit 1
fi
echo "diagnostics_status=${diag_code}"

section "Package smoke passed"
tail -n 20 "${LOG_FILE}"
