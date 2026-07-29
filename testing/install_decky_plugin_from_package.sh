#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
PACKAGE="${PACKAGE:-${SCRIPT_DIR}/${PLUGIN_NAME}.tar.gz}"
DECK_PLUGIN_DIR="${DECK_PLUGIN_DIR:-/home/deck/homebrew/plugins/${PLUGIN_NAME}}"
PLUGIN_PARENT="$(dirname "${DECK_PLUGIN_DIR}")"
BACKUP_ROOT="${BACKUP_ROOT:-/home/deck/.local/share/${PLUGIN_NAME}/backups/plugin-installs}"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="${BACKUP_ROOT}/${PLUGIN_NAME}-${STAMP}"
INSTALL_DIR="${PLUGIN_PARENT}/.${PLUGIN_NAME}.install-${STAMP}"

usage() {
  cat <<USAGE
Usage: ${0##*/} [--help]

Installs ${PLUGIN_NAME} from:
  ${PACKAGE}

Environment overrides:
  PACKAGE=/path/to/decky-mod-manager.tar.gz
  DECK_PLUGIN_DIR=/home/deck/homebrew/plugins/decky-mod-manager
  BACKUP_ROOT=/home/deck/.local/share/decky-mod-manager/backups/plugin-installs

The install requires sudo because Decky plugin files are root-owned and the
Decky plugin loader must be restarted.
USAGE
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
  "")
    ;;
  *)
    echo "Unknown argument: $1" >&2
    usage >&2
    exit 2
    ;;
esac

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
if [[ ! -x "${tmp_dir}/${PLUGIN_NAME}/bin/dmm-server" || ! -x "${tmp_dir}/${PLUGIN_NAME}/bin/dmm-nxm-handler" || ! -f "${tmp_dir}/${PLUGIN_NAME}/main.py" ]]; then
  echo "Package is missing required Decky Mod Manager files." >&2
  exit 1
fi

echo "==> Installing to ${DECK_PLUGIN_DIR}"
echo "    You may be prompted for the Steam Deck sudo/root password."
echo "==> Stopping existing ${PLUGIN_NAME} backend/plugin processes"
sudo pkill -f "^${DECK_PLUGIN_DIR}/bin/dmm-server$" 2>/dev/null || true
sudo pkill -f "^Decky Mod Manager \\(${DECK_PLUGIN_DIR}/main.py\\)$" 2>/dev/null || true

if [[ -d "${DECK_PLUGIN_DIR}" ]]; then
  echo "==> Backing up current plugin to ${BACKUP_DIR}"
  mkdir -p "${BACKUP_ROOT}"
  sudo cp -a "${DECK_PLUGIN_DIR}" "${BACKUP_DIR}"
fi

echo "==> Preparing replacement directory"
sudo rm -rf "${INSTALL_DIR}"
sudo mkdir -p "${INSTALL_DIR}"
sudo cp -R "${tmp_dir}/${PLUGIN_NAME}/." "${INSTALL_DIR}/"
sudo find "${INSTALL_DIR}" -name '._*' -delete
sudo chown -R root:root "${INSTALL_DIR}"
sudo chmod +x "${INSTALL_DIR}/bin/dmm-server" "${INSTALL_DIR}/bin/dmm-nxm-handler"

echo "==> Replacing plugin directory"
sudo rm -rf "${DECK_PLUGIN_DIR}"
sudo mv "${INSTALL_DIR}" "${DECK_PLUGIN_DIR}"

echo "==> Verifying installed package"
if [[ -x "${SCRIPT_DIR}/live_installed_package_check.sh" ]]; then
  if ! PACKAGE="${PACKAGE}" PLUGIN_DIR="${DECK_PLUGIN_DIR}" "${SCRIPT_DIR}/live_installed_package_check.sh"; then
    echo "Installed plugin does not match package." >&2
    if [[ -d "${BACKUP_DIR}" ]]; then
      echo "==> Restoring previous plugin from ${BACKUP_DIR}" >&2
      sudo rm -rf "${DECK_PLUGIN_DIR}"
      sudo cp -a "${BACKUP_DIR}" "${DECK_PLUGIN_DIR}"
    fi
    exit 1
  fi
else
  echo "Skipping installed package verification; ${SCRIPT_DIR}/live_installed_package_check.sh is missing." >&2
fi

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
