#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
TESTING_DIR="${TESTING_DIR:-${HOME}/.testing}"
REPO="${DMM_REPO:-justyntemme/dmm}"
RELEASE="${DMM_RELEASE:-dev-latest}"
PACKAGE_NAME="${PLUGIN_NAME}.tar.gz"
CHECKSUM_NAME="${PLUGIN_NAME}-checksums.txt"
PACKAGE_URL="${DMM_PACKAGE_URL:-https://github.com/${REPO}/releases/download/${RELEASE}/${PACKAGE_NAME}}"
CHECKSUM_URL="${DMM_CHECKSUM_URL:-https://github.com/${REPO}/releases/download/${RELEASE}/${CHECKSUM_NAME}}"
PACKAGE_PATH="${TESTING_DIR}/${PACKAGE_NAME}"
CHECKSUM_PATH="${TESTING_DIR}/${CHECKSUM_NAME}"
WRAPPER="${DMM_UPDATE_WRAPPER:-/opt/decky-mod-manager-testing/bin/decky-mod-manager-test-install}"

section() {
  printf '\n==> %s\n' "$1"
}

require_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    echo "Required file not found: ${path}" >&2
    exit 1
  fi
}

download() {
  local url="$1"
  local target="$2"
  local tmp="${target}.download"
  curl -fL --retry 3 --connect-timeout 20 --max-time 600 -o "${tmp}" "${url}"
  mv "${tmp}" "${target}"
}

verify_checksum() {
  local expected
  local actual

  expected="$(awk -v file="${PACKAGE_NAME}" '$2 == file {print $1}' "${CHECKSUM_PATH}")"
  if [[ -z "${expected}" ]]; then
    echo "Checksum file did not include ${PACKAGE_NAME}" >&2
    exit 1
  fi
  actual="$(sha256sum "${PACKAGE_PATH}" | awk '{print $1}')"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "Checksum mismatch for ${PACKAGE_PATH}" >&2
    echo "expected=${expected}" >&2
    echo "actual=${actual}" >&2
    exit 1
  fi
}

HELPERS=(
  "install_decky_plugin_from_package.sh"
  "install_decky_privileged_wrapper.sh"
  "install_decky_testing_sudoers.sh"
  "live_installed_package_check.sh"
)

for helper in "${HELPERS[@]}"; do
  require_file "${ROOT_DIR}/testing/${helper}"
done

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required to download ${PACKAGE_URL}" >&2
  exit 1
fi
if ! command -v sha256sum >/dev/null 2>&1; then
  echo "sha256sum is required to verify ${CHECKSUM_URL}" >&2
  exit 1
fi

section "Staging Decky Mod Manager test helpers"
mkdir -p "${TESTING_DIR}"
for helper in "${HELPERS[@]}"; do
  install -m 0755 "${ROOT_DIR}/testing/${helper}" "${TESTING_DIR}/${helper}"
done

section "Downloading ${RELEASE} package"
download "${PACKAGE_URL}" "${PACKAGE_PATH}"
download "${CHECKSUM_URL}" "${CHECKSUM_PATH}"
chmod 0644 "${PACKAGE_PATH}" "${CHECKSUM_PATH}"

section "Verifying package checksum"
verify_checksum

if [[ ! -x "${WRAPPER}" ]]; then
  section "Installing privileged test wrapper"
  echo "This step may prompt for your Steam Deck sudo password."
  (
    cd "${TESTING_DIR}"
    sudo ./install_decky_testing_sudoers.sh
  )
fi

section "Installing latest Decky Mod Manager package"
echo "The installer reboots the Steam Deck after a successful install."
sudo "${WRAPPER}"
