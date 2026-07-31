#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
STATE_DIR="${STATE_DIR:-${HOME}/.local/state/decky-mod-manager}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"

section() {
  printf '\n==> %s\n' "$1"
}

request() {
  local label="$1"
  local url="$2"
  section "$label"
  if ! curl -fsS "$url"; then
    printf '\nrequest failed: %s\n' "$url" >&2
  fi
  printf '\n'
}

tail_log() {
  local label="$1"
  local path="$2"
  section "$label"
  if [[ -f "$path" ]]; then
    tail -n "${LOG_LINES:-80}" "$path"
  else
    echo "missing: $path"
  fi
}

section "Decky Mod Manager live status"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "data_dir=${DATA_DIR}"
echo "state_dir=${STATE_DIR}"

section "Process Snapshot"
pgrep -af 'dmm-server|dmm-nxm-handler|PluginLoader' || true

section "Installed Package Check"
if [[ -x "${HOME}/.testing/live_installed_package_check.sh" ]]; then
  PACKAGE="${HOME}/.testing/decky-mod-manager.tar.gz" "${HOME}/.testing/live_installed_package_check.sh" || true
else
  echo "missing: ${HOME}/.testing/live_installed_package_check.sh"
fi

request "Health" "${BASE_URL}/api/health"
request "Status" "${BASE_URL}/api/status"
request "Games" "${BASE_URL}/api/games"
request "Game Diagnostics (${APP_ID})" "${BASE_URL}/api/games/${APP_ID}/diagnostics"
request "Jobs" "${BASE_URL}/api/jobs"
request "Profile Mods (${APP_ID})" "${BASE_URL}/api/games/${APP_ID}/mods"
request "Install Candidates (${APP_ID})" "${BASE_URL}/api/games/${APP_ID}/install-candidates"
request "Deployment Status (${APP_ID})" "${BASE_URL}/api/games/${APP_ID}/deploy/status"

section "Deployment Preview (${APP_ID})"
if ! curl -fsS "${BASE_URL}/api/games/${APP_ID}/deploy/preview"; then
  printf '\npreview failed; this is expected if no clean supported game/profile is available yet\n'
fi
printf '\n'

tail_log "Backend Log" "${STATE_DIR}/backend.log"
tail_log "Plugin Log" "${STATE_DIR}/plugin.log"
tail_log "NXM Handler Log" "${STATE_DIR}/nxm-handler.log"
