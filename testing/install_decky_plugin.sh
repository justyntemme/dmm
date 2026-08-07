#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DECK_HOST="${DECK_HOST:-192.168.8.102}"
DECK_USER="${DECK_USER:-deck}"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
DECK_STAGE="${DECK_STAGE:-/home/deck/.local/share/decky-mod-manager-dev/plugin-test}"
DECK_PLUGIN_DIR="${DECK_PLUGIN_DIR:-/home/deck/homebrew/plugins/${PLUGIN_NAME}}"
DECK_TARGET="${DECK_USER}@${DECK_HOST}"

LOCAL_PACKAGE="${ROOT_DIR}/dist/${PLUGIN_NAME}"
LOCAL_TARBALL="${ROOT_DIR}/dist/${PLUGIN_NAME}.tar.gz"
LOCAL_ZIP="${ROOT_DIR}/dist/${PLUGIN_NAME}.zip"
TESTING_SCRIPTS=(
  "${ROOT_DIR}/testing/install_decky_plugin_from_package.sh"
  "${ROOT_DIR}/testing/install_decky_privileged_wrapper.sh"
  "${ROOT_DIR}/testing/install_decky_testing_sudoers.sh"
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
  "${ROOT_DIR}/testing/live_extension_coverage_check.sh"
  "${ROOT_DIR}/testing/live_extension_targets_check.sh"
  "${ROOT_DIR}/testing/live_provider_resolve_check.sh"
  "${ROOT_DIR}/testing/live_installed_package_check.sh"
  "${ROOT_DIR}/testing/live_post_install_check.sh"
  "${ROOT_DIR}/testing/mvp_audit.sh"
)

SSH_OPTS=()
if [[ -n "${DECK_SSH_OPTS:-}" ]]; then
  # Intentionally simple splitting: pass options without embedded spaces.
  read -r -a SSH_OPTS <<<"${DECK_SSH_OPTS}"
fi

deck_ssh() {
  ssh "${SSH_OPTS[@]}" "${DECK_TARGET}" "$@"
}

deck_scp() {
  local args=("${SSH_OPTS[@]}")
  scp "${args[@]}" "$@"
}

echo "==> Building Go backend for Steam Deck"
cd "${ROOT_DIR}"
GOOS=linux GOARCH=amd64 go build -o bin/dmm-server-linux-amd64 ./cmd/dmm-server
GOOS=linux GOARCH=amd64 go build -o bin/dmm-nxm-handler-linux-amd64 ./cmd/dmm-nxm-handler

echo "==> Building Svelte web UI"
cd "${ROOT_DIR}/web"
npm install
npm run build

echo "==> Building Decky sidebar plugin"
cd "${ROOT_DIR}/decky"
npm install
npm run build

echo "==> Creating local plugin package at ${LOCAL_PACKAGE}"
cd "${ROOT_DIR}"
mkdir -p "${LOCAL_PACKAGE}/bin" "${LOCAL_PACKAGE}/web/dist" "${LOCAL_PACKAGE}/dist"
find "${LOCAL_PACKAGE}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
mkdir -p "${LOCAL_PACKAGE}/bin" "${LOCAL_PACKAGE}/web" "${LOCAL_PACKAGE}/dist"
cp bin/dmm-server-linux-amd64 "${LOCAL_PACKAGE}/bin/dmm-server"
cp bin/dmm-nxm-handler-linux-amd64 "${LOCAL_PACKAGE}/bin/dmm-nxm-handler"
cp decky/plugin.json decky/package.json decky/main.py "${LOCAL_PACKAGE}/"
cp -R decky/dist/. "${LOCAL_PACKAGE}/dist/"
cp -R web/dist "${LOCAL_PACKAGE}/web/"
chmod +x "${LOCAL_PACKAGE}/bin/dmm-server" "${LOCAL_PACKAGE}/bin/dmm-nxm-handler"

echo "==> Creating tarball ${LOCAL_TARBALL}"
COPYFILE_DISABLE=1 tar --no-xattrs -C "${ROOT_DIR}/dist" -czf "${LOCAL_TARBALL}" "${PLUGIN_NAME}" 2>/dev/null || \
  COPYFILE_DISABLE=1 tar -C "${ROOT_DIR}/dist" -czf "${LOCAL_TARBALL}" "${PLUGIN_NAME}"

echo "==> Creating Decky install ZIP ${LOCAL_ZIP}"
rm -f "${LOCAL_ZIP}"
(cd "${ROOT_DIR}/dist" && COPYFILE_DISABLE=1 zip -qr "${LOCAL_ZIP}" "${PLUGIN_NAME}")

echo "==> Copying package to ${DECK_USER}@${DECK_HOST}:${DECK_STAGE}"
deck_ssh "mkdir -p '${DECK_STAGE}' /home/deck/.testing"
deck_scp "${LOCAL_TARBALL}" "${LOCAL_ZIP}" "${DECK_TARGET}:${DECK_STAGE}/"
deck_scp "${LOCAL_TARBALL}" "${LOCAL_ZIP}" "${DECK_TARGET}:/home/deck/.testing/"
deck_scp "${TESTING_SCRIPTS[@]}" "${DECK_TARGET}:/home/deck/.testing/"
deck_ssh "chmod +x /home/deck/.testing/*.sh"

echo "==> Installing into Decky plugin directory"
echo "    You may be prompted for the Steam Deck sudo/root password."
ssh "${SSH_OPTS[@]}" -tt "${DECK_TARGET}" "
set -e
tmp_dir='${DECK_STAGE}/extract'
plugin_parent=\$(dirname '${DECK_PLUGIN_DIR}')
stamp=\$(date +%Y%m%d-%H%M%S)
backup_root='/home/deck/.local/share/${PLUGIN_NAME}/backups/plugin-installs'
backup_dir=\"\$backup_root/${PLUGIN_NAME}-\$stamp\"
install_dir=\"\$plugin_parent/.${PLUGIN_NAME}.install-\$stamp\"
rm -rf \"\$tmp_dir\"
mkdir -p \"\$tmp_dir\"
tar -xzf '${DECK_STAGE}/${PLUGIN_NAME}.tar.gz' -C \"\$tmp_dir\"
if [ ! -x \"\$tmp_dir/${PLUGIN_NAME}/bin/dmm-server\" ] || [ ! -x \"\$tmp_dir/${PLUGIN_NAME}/bin/dmm-nxm-handler\" ] || [ ! -f \"\$tmp_dir/${PLUGIN_NAME}/main.py\" ]; then
  echo 'Package is missing required Decky Mod Manager files.' >&2
  exit 1
fi
echo 'Stopping existing ${PLUGIN_NAME} backend/plugin processes'
sudo pkill -f '^${DECK_PLUGIN_DIR}/bin/dmm-server$' 2>/dev/null || true
sudo pkill -f '^Decky Mod Manager \(${DECK_PLUGIN_DIR}/main.py\)$' 2>/dev/null || true
if [ -d '${DECK_PLUGIN_DIR}' ]; then
  echo \"Backing up current plugin to \$backup_dir\"
  mkdir -p \"\$backup_root\"
  sudo cp -a '${DECK_PLUGIN_DIR}' \"\$backup_dir\"
fi
sudo rm -rf \"\$install_dir\"
sudo mkdir -p \"\$install_dir\"
sudo cp -R \"\$tmp_dir/${PLUGIN_NAME}/.\" \"\$install_dir/\"
sudo find \"\$install_dir\" -name '._*' -delete
sudo chown -R root:root \"\$install_dir\"
sudo chmod +x \"\$install_dir/bin/dmm-server\" \"\$install_dir/bin/dmm-nxm-handler\"
sudo rm -rf '${DECK_PLUGIN_DIR}'
sudo mv \"\$install_dir\" '${DECK_PLUGIN_DIR}'
if sudo systemctl restart plugin_loader.service 2>/dev/null; then
  echo 'Restarted plugin_loader.service'
elif sudo systemctl restart plugin_loader-release.service 2>/dev/null; then
  echo 'Restarted plugin_loader-release.service'
else
  sudo pkill -f '/home/deck/homebrew/services/PluginLoader' || true
  nohup sudo /home/deck/homebrew/services/PluginLoader >/tmp/decky-pluginloader.log 2>&1 &
  echo 'Restarted PluginLoader process directly'
fi
echo 'Installed ${PLUGIN_NAME} to ${DECK_PLUGIN_DIR}'
"

echo "==> Verifying installed package"
deck_ssh "PACKAGE='/home/deck/.testing/${PLUGIN_NAME}.tar.gz' PLUGIN_DIR='${DECK_PLUGIN_DIR}' /home/deck/.testing/live_installed_package_check.sh"

echo "==> Done"
echo "Open the Decky sidebar in Gaming Mode and look for Decky Mod Manager."
