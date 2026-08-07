#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

section() {
  printf '\n==> %s\n' "$1"
}

cleanup_python_cache() {
  find "${ROOT_DIR}/decky/__pycache__" -type f -delete 2>/dev/null || true
  rmdir "${ROOT_DIR}/decky/__pycache__" 2>/dev/null || true
}

section "Go tests"
(
  cd "${ROOT_DIR}"
  go test ./...
)

section "Web build"
(
  cd "${ROOT_DIR}"
  npm --prefix web run build
)

section "Decky build"
(
  cd "${ROOT_DIR}"
  npm --prefix decky run build
)

section "Decky Python syntax"
(
  cd "${ROOT_DIR}"
  python3 -m py_compile decky/main.py
  cleanup_python_cache
)

section "Testing script syntax"
(
  cd "${ROOT_DIR}"
  bash -n \
    testing/ui_mvp_audit.sh \
    testing/local_smoke.sh \
    testing/deck_package_smoke.sh \
    testing/deck_rehearsal.sh \
    testing/create_deck_transfer_bundle.sh \
    testing/live_status.sh \
    testing/mvp_live_check.sh \
    testing/live_auto_install_check.sh \
    testing/live_profile_toggle_check.sh \
    testing/live_profile_transfer_check.sh \
    testing/live_profile_seed_check.sh \
    testing/live_rollback_check.sh \
    testing/live_stardew_mod_files_check.sh \
    testing/live_web_asset_check.sh \
    testing/live_nexus_browser_handoff_check.sh \
    testing/live_ui_preferences_check.sh \
    testing/live_extension_coverage_check.sh \
    testing/live_provider_resolve_check.sh \
    testing/live_installed_package_check.sh \
    testing/live_post_install_check.sh \
    testing/install_decky_plugin.sh \
    testing/install_decky_plugin_from_package.sh \
    testing/install_decky_privileged_wrapper.sh \
    testing/install_decky_testing_sudoers.sh
)

section "MVP UI product audit"
(
  cd "${ROOT_DIR}"
  ./testing/ui_mvp_audit.sh
)

section "Deck testing artifact coverage"
(
  cd "${ROOT_DIR}"
  for script in \
    install_decky_plugin_from_package.sh \
    install_decky_privileged_wrapper.sh \
    install_decky_testing_sudoers.sh \
    deck_package_smoke.sh \
    deck_rehearsal.sh \
    live_status.sh \
    mvp_live_check.sh \
    live_auto_install_check.sh \
    live_profile_toggle_check.sh \
    live_profile_transfer_check.sh \
    live_profile_seed_check.sh \
    live_rollback_check.sh \
    live_stardew_mod_files_check.sh \
    live_web_asset_check.sh \
    live_nexus_browser_handoff_check.sh \
    live_ui_preferences_check.sh \
    live_extension_coverage_check.sh \
    live_provider_resolve_check.sh \
    live_installed_package_check.sh \
    live_post_install_check.sh \
    mvp_audit.sh
  do
    rg -q "${script}" testing/create_deck_transfer_bundle.sh
    rg -q "${script}" testing/install_decky_plugin.sh
  done
)

section "Local backend smoke"
(
  cd "${ROOT_DIR}"
  ./testing/local_smoke.sh
)

section "Package"
(
  cd "${ROOT_DIR}"
  make package
)

section "Package shape"
(
  cd "${ROOT_DIR}"
  PACKAGE=dist/decky-mod-manager.tar.gz SHAPE_ONLY=1 TMP_ROOT="${TMPDIR:-/tmp}/dmm-package-shape-audit" ./testing/deck_package_smoke.sh
)

section "Diff whitespace"
(
  cd "${ROOT_DIR}"
  git diff --check
)

section "MVP repo audit passed"
