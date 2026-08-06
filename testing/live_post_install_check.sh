#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
LOG_LINES="${LOG_LINES:-120}"

section() {
  printf '\n==> %s\n' "$1"
}

run_step() {
  local label="$1"
  shift
  section "${label}"
  "$@"
}

section "Decky Mod Manager post-install live check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "script_dir=${SCRIPT_DIR}"

section "Installed package matches staged package"
if ! "${SCRIPT_DIR}/live_installed_package_check.sh"; then
  cat >&2 <<TEXT

The installed Decky plugin does not match the package staged in ${SCRIPT_DIR}.
Run this on the Steam Deck, then start the server from Decky and rerun this check:

  ${SCRIPT_DIR}/install_decky_plugin_from_package.sh

TEXT
  exit 1
fi

section "Backend health"
if ! curl -fsS "${BASE_URL}/api/health"; then
  cat >&2 <<TEXT

Backend is not reachable at ${BASE_URL}.
Open Decky Mod Manager in Gaming Mode, start the server, then rerun this script.
TEXT
  exit 1
fi
printf '\n'

run_step "Web UI assets" \
  env BASE_URL="${BASE_URL}" "${SCRIPT_DIR}/live_web_asset_check.sh"

run_step "Live status snapshot" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" LOG_LINES="${LOG_LINES}" "${SCRIPT_DIR}/live_status.sh"

run_step "MVP live acceptance" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/mvp_live_check.sh"

run_step "Profile transfer" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/live_profile_transfer_check.sh"

run_step "Seeded profile" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/live_profile_seed_check.sh"

run_step "Shared UI preferences" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/live_ui_preferences_check.sh"

run_step "Rollback restore" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/live_rollback_check.sh"

run_step "Stardew file visibility" \
  env BASE_URL="${BASE_URL}" APP_ID="${APP_ID}" "${SCRIPT_DIR}/live_stardew_mod_files_check.sh"

section "Post-install live check passed"
echo "Run REQUIRE_SMAPI_ROOT=1 REQUIRE_RUNTIME=1 ${SCRIPT_DIR}/live_stardew_mod_files_check.sh after SMAPI is deployed/configured if you want SMAPI root files and runtime readiness to be strict."
