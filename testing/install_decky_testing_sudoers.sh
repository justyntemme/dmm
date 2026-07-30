#!/usr/bin/env bash
set -euo pipefail

PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
TESTING_DIR="${TESTING_DIR:-/home/deck/.testing}"
SUDOERS_PATH="${SUDOERS_PATH:-/etc/sudoers.d/zz-decky-mod-manager-testing}"
LEGACY_SUDOERS_PATH="${LEGACY_SUDOERS_PATH:-/etc/sudoers.d/decky-mod-manager-testing}"
PRIVILEGED_ROOT="${PRIVILEGED_ROOT:-/opt/decky-mod-manager-testing}"
PRIVILEGED_DIR="${PRIVILEGED_DIR:-${PRIVILEGED_ROOT}/bin}"
WRAPPER_SOURCE="${TESTING_DIR}/install_decky_privileged_wrapper.sh"
WRAPPER_TARGET="${PRIVILEGED_DIR}/decky-mod-manager-test-install"
INSTALLER_TARGET="${PRIVILEGED_DIR}/install_decky_plugin_from_package.sh"
VERIFIER_SOURCE="${TESTING_DIR}/live_installed_package_check.sh"
VERIFIER_TARGET="${PRIVILEGED_DIR}/live_installed_package_check.sh"
SETUP_SCRIPT="${TESTING_DIR}/install_decky_testing_sudoers.sh"
INSTALLER="${TESTING_DIR}/install_decky_plugin_from_package.sh"
TARBALL="${TESTING_DIR}/${PLUGIN_NAME}.tar.gz"
ZIP="${TESTING_DIR}/${PLUGIN_NAME}.zip"

usage() {
  cat <<USAGE
Usage: ${0##*/} [--remove]

Installs a narrow sudoers rule for overnight Decky Mod Manager testing.

The rule does not allow passwordless sudo of writable ~/.testing scripts.
Instead, it installs a root-owned wrapper at:
  ${WRAPPER_TARGET}

The wrapper calls a root-owned copy of the package installer at:
  ${INSTALLER_TARGET}

Allowed passwordless commands:
  ${WRAPPER_TARGET}
  /usr/bin/systemctl restart plugin_loader.service
  /usr/bin/systemctl status plugin_loader.service
  /usr/bin/journalctl -u plugin_loader.service
  /usr/bin/shutdown -r now

After installation, run package installs with:
  sudo ${WRAPPER_TARGET}

To remove the sudoers rule and wrapper:
  sudo ${0##*/} --remove
USAGE
}

case "${1:-}" in
  -h|--help)
    usage
    exit 0
    ;;
  --remove)
    sudo rm -f "${SUDOERS_PATH}" "${LEGACY_SUDOERS_PATH}" "${WRAPPER_TARGET}" "${INSTALLER_TARGET}" "${VERIFIER_TARGET}"
    sudo rmdir "${PRIVILEGED_DIR}" 2>/dev/null || true
    sudo rmdir "${PRIVILEGED_ROOT}" 2>/dev/null || true
    echo "Removed ${SUDOERS_PATH}, ${LEGACY_SUDOERS_PATH}, and ${PRIVILEGED_ROOT}"
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

for path in "${WRAPPER_SOURCE}" "${INSTALLER}" "${VERIFIER_SOURCE}" "${TARBALL}"; do
  if [[ ! -f "${path}" ]]; then
    echo "Required file not found: ${path}" >&2
    exit 1
  fi
done

tmp_wrapper="$(mktemp)"
tmp_sudoers="$(mktemp)"
cleanup() {
  rm -f "${tmp_wrapper}" "${tmp_sudoers}"
}
trap cleanup EXIT

cp "${WRAPPER_SOURCE}" "${tmp_wrapper}"
chmod 0755 "${tmp_wrapper}"

cat >"${tmp_sudoers}" <<EOF
# Decky Mod Manager temporary test permissions.
# Remove with: sudo ${TESTING_DIR}/install_decky_testing_sudoers.sh --remove
deck ALL=(root) NOPASSWD: ${WRAPPER_TARGET}
deck ALL=(root) NOPASSWD: /usr/bin/systemctl restart plugin_loader.service
deck ALL=(root) NOPASSWD: /usr/bin/systemctl status plugin_loader.service
deck ALL=(root) NOPASSWD: /usr/bin/journalctl -u plugin_loader.service
deck ALL=(root) NOPASSWD: /usr/bin/shutdown -r now
EOF

sudo install -d -o root -g root -m 0755 "${PRIVILEGED_DIR}"
sudo install -o root -g root -m 0755 "${tmp_wrapper}" "${WRAPPER_TARGET}"
sudo install -o root -g root -m 0755 "${INSTALLER}" "${INSTALLER_TARGET}"
sudo install -o root -g root -m 0755 "${VERIFIER_SOURCE}" "${VERIFIER_TARGET}"
sudo chown deck:deck "${TARBALL}"
sudo chmod 0644 "${TARBALL}"
if [[ -f "${ZIP}" ]]; then
  sudo chown deck:deck "${ZIP}"
  sudo chmod 0644 "${ZIP}"
fi
if [[ "${LEGACY_SUDOERS_PATH}" != "${SUDOERS_PATH}" ]]; then
  sudo rm -f "${LEGACY_SUDOERS_PATH}"
fi
sudo install -o root -g root -m 0440 "${tmp_sudoers}" "${SUDOERS_PATH}"
sudo visudo -cf "${SUDOERS_PATH}"

echo "Installed ${SUDOERS_PATH}"
echo "Installed ${WRAPPER_TARGET}"
echo
echo "Passwordless test install command:"
echo "  sudo ${WRAPPER_TARGET}"
echo
echo "Remove when done:"
echo "  sudo ${TESTING_DIR}/install_decky_testing_sudoers.sh --remove"
