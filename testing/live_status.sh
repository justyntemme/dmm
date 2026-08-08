#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
STATE_DIR="${STATE_DIR:-${HOME}/.local/state/decky-mod-manager}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"
TOKEN_FILE="${DMM_TOKEN_FILE:-${STATE_DIR}/api-token}"
API_TOKEN="${DMM_AUTH_TOKEN:-}"
if [[ -z "${API_TOKEN}" && -f "${TOKEN_FILE}" ]]; then
  API_TOKEN="$(<"${TOKEN_FILE}")"
fi

section() {
  printf '\n==> %s\n' "$1"
}

request() {
  local label="$1"
  local url="$2"
  section "$label"
  local curl_args=(-fsS)
  if [[ -n "${API_TOKEN}" && "${url}" == */api/* && "${url}" != */api/health ]]; then
    curl_args+=(-H "X-DMM-Token: ${API_TOKEN}")
  fi
  if ! curl "${curl_args[@]}" "$url"; then
    printf '\nrequest failed: %s\n' "$url" >&2
  fi
  printf '\n'
}

tail_log() {
  local label="$1"
  local path="$2"
  section "$label"
  if [[ -f "$path" ]]; then
    tail -n "${LOG_LINES:-80}" "$path" | redact_log
  else
    echo "missing: $path"
  fi
}

redact_log() {
  python3 - <<'PY'
import re
import sys

query_pattern = re.compile(r"(?i)((?:key|expires|md5|token|api_key)=)[^&\"'\s\\]+")
json_field_pattern = re.compile(r"(?i)(\"(?:nxm_)?(?:key|expires|token|api_key)\"\s*:\s*\")[^\"]*(\")")

for line in sys.stdin:
    line = query_pattern.sub(r"\1[redacted]", line)
    line = json_field_pattern.sub(r"\1[redacted]\2", line)
    sys.stdout.write(line)
PY
}

section "Decky Mod Manager live status"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "data_dir=${DATA_DIR}"
echo "state_dir=${STATE_DIR}"
echo "api_auth=$([[ -n "${API_TOKEN}" ]] && printf token || printf none)"

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
preview_args=(-fsS)
if [[ -n "${API_TOKEN}" ]]; then
  preview_args+=(-H "X-DMM-Token: ${API_TOKEN}")
fi
if ! curl "${preview_args[@]}" "${BASE_URL}/api/games/${APP_ID}/deploy/preview"; then
  printf '\npreview failed; this is expected if no clean supported game/profile is available yet\n'
fi
printf '\n'

tail_log "Backend Log" "${STATE_DIR}/backend.log"
tail_log "Plugin Log" "${STATE_DIR}/plugin.log"
tail_log "NXM Handler Log" "${STATE_DIR}/nxm-handler.log"
