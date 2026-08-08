#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/dist/deck-transfer}"
ARCHIVE="${ARCHIVE:-${ROOT_DIR}/dist/decky-mod-manager-deck-transfer.tar.gz}"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"

section() {
  printf '\n==> %s\n' "$1"
}

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "missing required file: $path" >&2
    exit 1
  fi
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@"
  fi
}

TARBALL="${ROOT_DIR}/dist/${PLUGIN_NAME}.tar.gz"
ZIP="${ROOT_DIR}/dist/${PLUGIN_NAME}.zip"
FILES=(
  "${TARBALL}"
  "${ZIP}"
  "${ROOT_DIR}/testing/install_decky_plugin_from_package.sh"
  "${ROOT_DIR}/testing/install_decky_privileged_wrapper.sh"
  "${ROOT_DIR}/testing/install_decky_testing_sudoers.sh"
  "${ROOT_DIR}/testing/dmm_test_auth.py"
  "${ROOT_DIR}/testing/deck_package_smoke.sh"
  "${ROOT_DIR}/testing/deck_rehearsal.sh"
  "${ROOT_DIR}/testing/live_status.sh"
  "${ROOT_DIR}/testing/mvp_live_check.sh"
  "${ROOT_DIR}/testing/live_auto_install_check.sh"
  "${ROOT_DIR}/testing/live_profile_toggle_check.sh"
  "${ROOT_DIR}/testing/live_profile_transfer_check.sh"
  "${ROOT_DIR}/testing/live_profile_seed_check.sh"
  "${ROOT_DIR}/testing/live_rollback_check.sh"
  "${ROOT_DIR}/testing/live_stardew_mod_files_check.sh"
  "${ROOT_DIR}/testing/live_web_asset_check.sh"
  "${ROOT_DIR}/testing/live_nexus_browser_handoff_check.sh"
  "${ROOT_DIR}/testing/live_ui_preferences_check.sh"
  "${ROOT_DIR}/testing/live_auth_pairing_check.sh"
  "${ROOT_DIR}/testing/live_local_archive_security_check.sh"
  "${ROOT_DIR}/testing/live_extension_coverage_check.sh"
  "${ROOT_DIR}/testing/live_extension_targets_check.sh"
  "${ROOT_DIR}/testing/live_provider_resolve_check.sh"
  "${ROOT_DIR}/testing/live_installed_package_check.sh"
  "${ROOT_DIR}/testing/live_post_install_check.sh"
  "${ROOT_DIR}/testing/mvp_audit.sh"
)

for file in "${FILES[@]}"; do
  require_file "$file"
done

section "Creating Deck transfer bundle"
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"
for file in "${FILES[@]}"; do
  cp "$file" "${OUT_DIR}/"
done
chmod +x "${OUT_DIR}"/*.sh

(
  cd "${OUT_DIR}"
  sha256_file * > SHA256SUMS
)

cat > "${OUT_DIR}/README.txt" <<'TEXT'
Decky Mod Manager Deck transfer bundle

Copy either this folder or decky-mod-manager-deck-transfer.tar.gz to the Steam Deck.

If you copied the folder, run from the Deck:

  mkdir -p ~/.testing
  cp -a /path/to/deck-transfer/. ~/.testing/

If you copied decky-mod-manager-deck-transfer.tar.gz, run from the Deck:

  mkdir -p ~/.testing
  tar -xzf /path/to/decky-mod-manager-deck-transfer.tar.gz -C /tmp
  cp -a /tmp/deck-transfer/. ~/.testing/

Then run:

  chmod +x ~/.testing/*.sh
  cd ~/.testing
  sha256sum -c SHA256SUMS
  ./deck_package_smoke.sh
  ./deck_rehearsal.sh
  ./install_decky_plugin_from_package.sh

For unattended overnight testing, install the temporary narrow sudoers rule:

  ./install_decky_testing_sudoers.sh

Then package installs can run without a sudo prompt through the root-owned wrapper:

  sudo /opt/decky-mod-manager-testing/bin/decky-mod-manager-test-install

Remove the temporary rule when done:

  sudo ~/.testing/install_decky_testing_sudoers.sh --remove

After installing, start the server from Decky and run:

  ~/.testing/live_post_install_check.sh

Or run the individual checks:

  ~/.testing/live_status.sh
  ~/.testing/live_installed_package_check.sh
  ~/.testing/mvp_live_check.sh
  ~/.testing/live_profile_toggle_check.sh
  ~/.testing/live_profile_transfer_check.sh
  ~/.testing/live_profile_seed_check.sh
  ~/.testing/live_rollback_check.sh
  ~/.testing/live_stardew_mod_files_check.sh
  ~/.testing/live_nexus_browser_handoff_check.sh
  ~/.testing/live_extension_coverage_check.sh
  ~/.testing/live_extension_targets_check.sh
  ~/.testing/live_provider_resolve_check.sh
  ~/.testing/live_auth_pairing_check.sh
  ~/.testing/live_local_archive_security_check.sh

For auto-install validation, start the server from Decky, run:

  ~/.testing/live_auto_install_check.sh

Then click a fresh Nexus Mod Manager Download link from the Deck browser while the script is waiting.
TEXT

section "Creating transfer archive"
rm -f "${ARCHIVE}"
COPYFILE_DISABLE=1 tar --no-xattrs -C "$(dirname "${OUT_DIR}")" -czf "${ARCHIVE}" "$(basename "${OUT_DIR}")" 2>/dev/null || \
  COPYFILE_DISABLE=1 tar -C "$(dirname "${OUT_DIR}")" -czf "${ARCHIVE}" "$(basename "${OUT_DIR}")"

section "Bundle ready"
echo "folder=${OUT_DIR}"
echo "archive=${ARCHIVE}"
