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

require_text "$WEB_APP" "type SettingsPage = \"overview\" | \"jobs\" | \"install\" | \"sources\" | \"nexus\";" \
  "mobile settings pages must keep install/provider settings separate from server settings"
require_text "$WEB_APP" "Selected Profile" \
  "game Plugins view must lead with selected profile state"
require_text "$WEB_APP" "Installed, disabled in this profile" \
  "game Mods view must include profile-scoped mod state"
require_text "$WEB_APP" "Add Mod" \
  "generic mod import must live inside the selected game workspace"
require_text "$WEB_APP" "Explore Mods" \
  "phone/tablet UI must expose game-scoped source browsing"
require_text "$WEB_APP" "Open on Deck" \
  "phone/tablet Nexus browsing must use the Deck browser capture flow"
require_text "$WEB_APP" "Steam Workshop" \
  "phone/tablet Mods view must show Steam Workshop platform mods when present"
require_text "$WEB_APP" "source-pill" \
  "mod and action rows must carry visible source tags"
require_text "$WEB_APP" "Archive upload" \
  "source settings must show Local Archive as an active upload provider"
require_text "$WEB_APP" "plugin.catalog ?? plugin.source" \
  "load-order rows must display provider/native source tags"
require_text "$WEB_APP" "source-native" \
  "native plugin rows must have a visible source-pill style"
require_text "$WEB_APP" "Advanced Profile Tools" \
  "file-level deployment controls must remain an advanced disclosure"
require_text "$WEB_APP" "These Deck behavior switches are managed from the Decky sidebar settings." \
  "mobile install settings must point users to Decky-owned behavior switches"
reject_text "$WEB_APP" "Auto-install captured downloads" \
  "mobile UI must not expose the Decky-owned auto-install toggle"
reject_text "$WEB_APP" "Auto-enable installed mods" \
  "mobile UI must not expose the Decky-owned auto-enable toggle"
reject_text "$WEB_APP" "Approve downloads automatically" \
  "mobile install settings must not expose the retired download-approval model"
reject_text "$WEB_APP" ">Server</button>" \
  "mobile UI must not expose a Server settings tab; server access belongs in Decky"
reject_text "$WEB_APP" ">Dependencies</button>" \
  "mobile UI must not expose a Dependencies tab; dependency status belongs in Decky"

require_text "$DECKY_APP" "Server Access" \
  "Decky plugin must retain server access controls/status"
require_text "$DECKY_APP" "games: \"Games\"" \
  "Decky route tab must be labeled Games for the game-selection workflow"
require_text "$DECKY_APP" "Dependencies" \
  "Decky plugin must retain dependency status"
require_text "$DECKY_APP" "Auto-install captured downloads" \
  "Decky plugin must expose automatic captured-install behavior"
require_text "$DECKY_APP" "Auto-enable installed mods" \
  "Decky plugin must expose automatic enable/deploy behavior"
require_text "$DECKY_APP" "plugin.catalog || plugin.source" \
  "Decky load-order rows must display provider/native source tags"
require_text "$DECKY_APP" "native: { border" \
  "Decky source-pill palette must include native plugin rows"
reject_text "$DECKY_APP" "Open Nexus Mods" \
  "Decky Manage tab must not expose the removed global external Nexus opener"
reject_text "$DECKY_APP" "Open Nexus Page" \
  "Decky selected-game view must use the in-app Nexus browser instead of a separate external page opener"
reject_text "$DECKY_APP" "\"open_nexus\"" \
  "Decky plugin must not retain the removed external Nexus bridge call"

echo "MVP UI product audit passed"
