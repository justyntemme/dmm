#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

DECK_HOST="${DECK_HOST:-192.168.8.102}"
DECK_USER="${DECK_USER:-deck}"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
DECK_STAGE="${DECK_STAGE:-/home/deck/.local/share/decky-mod-manager-dev/plugin-test}"
DECK_PLUGIN_DIR="${DECK_PLUGIN_DIR:-/home/deck/homebrew/plugins/${PLUGIN_NAME}}"

LOCAL_PACKAGE="${ROOT_DIR}/dist/${PLUGIN_NAME}"
LOCAL_TARBALL="${ROOT_DIR}/dist/${PLUGIN_NAME}.tar.gz"
LOCAL_ZIP="${ROOT_DIR}/dist/${PLUGIN_NAME}.zip"

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
ssh "${DECK_USER}@${DECK_HOST}" "mkdir -p '${DECK_STAGE}'"
scp "${LOCAL_TARBALL}" "${LOCAL_ZIP}" "${DECK_USER}@${DECK_HOST}:${DECK_STAGE}/"

echo "==> Installing into Decky plugin directory"
echo "    You may be prompted for the Steam Deck sudo/root password."
ssh -tt "${DECK_USER}@${DECK_HOST}" "
set -e
tmp_dir='${DECK_STAGE}/extract'
rm -rf \"\$tmp_dir\"
mkdir -p \"\$tmp_dir\"
tar -xzf '${DECK_STAGE}/${PLUGIN_NAME}.tar.gz' -C \"\$tmp_dir\"
sudo mkdir -p '${DECK_PLUGIN_DIR}'
sudo find '${DECK_PLUGIN_DIR}' -mindepth 1 -maxdepth 1 -exec rm -rf {} +
sudo cp -R \"\$tmp_dir/${PLUGIN_NAME}/.\" '${DECK_PLUGIN_DIR}/'
sudo find '${DECK_PLUGIN_DIR}' -name '._*' -delete
sudo chown -R root:root '${DECK_PLUGIN_DIR}'
sudo chmod +x '${DECK_PLUGIN_DIR}/bin/dmm-server' '${DECK_PLUGIN_DIR}/bin/dmm-nxm-handler'
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

echo "==> Done"
echo "Open the Decky sidebar in Gaming Mode and look for Decky Mod Manager."
