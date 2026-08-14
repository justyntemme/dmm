#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DECKY_APP="${ROOT_DIR}/decky/src/index.tsx"

require_text() {
  local pattern="$1"
  local message="$2"
  if ! rg -q --fixed-strings "$pattern" "$DECKY_APP"; then
    echo "$message" >&2
    exit 1
  fi
}

require_text "function keepDeckyScrollFocusVisible(" "Decky controller scroll helper is missing"
require_text 'main-${tab}-${selectedGameID ? "game" : "list"}' "main Decky pages must use controller scroll restoration"

for surface in installer-choices nexus-browser deployment-recovery pair-phone profile-picker archive-browser; do
  require_text "label: \"${surface}\"" "Decky controller scroll restoration is missing for ${surface}"
done

echo "Decky controller scroll audit passed"
