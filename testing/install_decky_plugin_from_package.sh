#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
PACKAGE="${PACKAGE:-${SCRIPT_DIR}/${PLUGIN_NAME}.tar.gz}"
DECK_PLUGIN_DIR="${DECK_PLUGIN_DIR:-/home/deck/homebrew/plugins/${PLUGIN_NAME}}"

if [[ ! -f "${PACKAGE}" ]]; then
  echo "Package not found: ${PACKAGE}" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

echo "==> Extracting ${PACKAGE}"
case "${PACKAGE}" in
  *.zip)
    unzip -q "${PACKAGE}" -d "${tmp_dir}"
    ;;
  *.tar.gz|*.tgz)
    tar -xzf "${PACKAGE}" -C "${tmp_dir}"
    ;;
  *)
    echo "Unsupported package type: ${PACKAGE}" >&2
    exit 1
    ;;
esac

if [[ ! -d "${tmp_dir}/${PLUGIN_NAME}" ]]; then
  echo "Package did not contain ${PLUGIN_NAME}/" >&2
  exit 1
fi

echo "==> Installing to ${DECK_PLUGIN_DIR}"
echo "    You may be prompted for the Steam Deck sudo/root password."
sudo mkdir -p "${DECK_PLUGIN_DIR}"
sudo find "${DECK_PLUGIN_DIR}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
sudo cp -R "${tmp_dir}/${PLUGIN_NAME}/." "${DECK_PLUGIN_DIR}/"
sudo find "${DECK_PLUGIN_DIR}" -name '._*' -delete
sudo chown -R root:root "${DECK_PLUGIN_DIR}"
sudo chmod +x "${DECK_PLUGIN_DIR}/bin/dmm-server" "${DECK_PLUGIN_DIR}/bin/dmm-nxm-handler"

echo "==> Restarting Decky plugin loader"
if sudo systemctl restart plugin_loader.service 2>/dev/null; then
  echo "Restarted plugin_loader.service"
elif sudo systemctl restart plugin_loader-release.service 2>/dev/null; then
  echo "Restarted plugin_loader-release.service"
else
  sudo pkill -f '/home/deck/homebrew/services/PluginLoader' || true
  nohup sudo /home/deck/homebrew/services/PluginLoader >/tmp/decky-pluginloader.log 2>&1 &
  echo "Restarted PluginLoader process directly"
fi

echo "==> Installed ${PLUGIN_NAME}"
echo "Open the Decky sidebar in Gaming Mode and look for Decky Mod Manager."
