#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_APP="${ROOT_DIR}/web/src/App.svelte"
DECKY_APP="${ROOT_DIR}/decky/src/index.tsx"

section() {
  printf '\n==> %s\n' "$1"
}

require_text() {
  local file="$1"
  local pattern="$2"
  local message="$3"
  if ! rg -q --fixed-strings "$pattern" "$file"; then
    echo "${message}" >&2
    exit 1
  fi
}

reject_text() {
  local file="$1"
  local pattern="$2"
  local message="$3"
  if rg -q --fixed-strings "$pattern" "$file"; then
    echo "${message}" >&2
    exit 1
  fi
}

section "MVP UI product audit"

require_text "$WEB_APP" "type SettingsPage = \"overview\" | \"jobs\" | \"install\" | \"nexus\";" \
  "mobile settings pages must keep install settings separate from server settings"
require_text "$WEB_APP" "Selected Profile" \
  "game Plugins view must lead with selected profile state"
require_text "$WEB_APP" "Profile Mods" \
  "game Plugins view must include the profile mod list"
require_text "$WEB_APP" "Add From Nexus" \
  "Nexus import must live inside the selected game workspace"
require_text "$WEB_APP" "Advanced Deployment Tools" \
  "file-level deployment controls must remain an advanced disclosure"
require_text "$WEB_APP" "Auto install captured downloads" \
  "mobile install settings must show Deck-managed install behavior"
reject_text "$WEB_APP" "Approve downloads automatically" \
  "mobile install settings must not expose the retired download-approval model"
reject_text "$WEB_APP" ">Server</button>" \
  "mobile UI must not expose a Server settings tab; server access belongs in Decky"
reject_text "$WEB_APP" ">Dependencies</button>" \
  "mobile UI must not expose a Dependencies tab; dependency status belongs in Decky"

require_text "$DECKY_APP" "Server Access" \
  "Decky plugin must retain server access controls/status"
require_text "$DECKY_APP" "Dependencies" \
  "Decky plugin must retain dependency status"
require_text "$DECKY_APP" "Auto-install captured downloads" \
  "Decky plugin must expose automatic install approval"
require_text "$DECKY_APP" "Auto-enable installed mods" \
  "Decky plugin must expose automatic enable/deploy behavior"

echo "MVP UI product audit passed"
