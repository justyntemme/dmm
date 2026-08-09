#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
TESTING_DIR="${TESTING_DIR:-${HOME}/.testing}"
REPO="${DMM_REPO:-justyntemme/dmm}"
RELEASE="${DMM_RELEASE:-}"
PACKAGE_NAME="${PLUGIN_NAME}.tar.gz"
PACKAGE_URL="${DMM_PACKAGE_URL:-}"
PACKAGE_PATH="${TESTING_DIR}/${PACKAGE_NAME}"
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

resolve_package_url() {
  if [[ -n "${PACKAGE_URL}" ]]; then
    printf '%s\n' "${PACKAGE_URL}"
    return
  fi
  if [[ -n "${RELEASE}" ]]; then
    printf 'https://github.com/%s/releases/download/%s/%s\n' "${REPO}" "${RELEASE}" "${PACKAGE_NAME}"
    return
  fi
  python3 - "$REPO" "$PACKAGE_NAME" <<'PY'
import json
import sys
import urllib.request

repo = sys.argv[1]
package_name = sys.argv[2]
request = urllib.request.Request(
    f"https://api.github.com/repos/{repo}/releases/latest",
    headers={"User-Agent": "Decky-Mod-Manager-Updater"},
)
with urllib.request.urlopen(request, timeout=30) as response:
    release = json.loads(response.read().decode("utf-8"))
tag = str(release.get("tag_name") or "")
for asset in release.get("assets") or []:
    if asset.get("name") == package_name and asset.get("browser_download_url"):
        print(asset["browser_download_url"])
        sys.exit(0)
raise SystemExit(f"latest release {tag or '<unknown>'} does not include {package_name}")
PY
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
if [[ -z "${PACKAGE_URL}" && -z "${RELEASE}" ]] && ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to resolve the latest GitHub release. Set DMM_RELEASE or DMM_PACKAGE_URL to skip lookup." >&2
  exit 1
fi
PACKAGE_URL="$(resolve_package_url)"

section "Staging Decky Mod Manager test helpers"
mkdir -p "${TESTING_DIR}"
for helper in "${HELPERS[@]}"; do
  install -m 0755 "${ROOT_DIR}/testing/${helper}" "${TESTING_DIR}/${helper}"
done

section "Downloading ${RELEASE:-latest release} package"
download "${PACKAGE_URL}" "${PACKAGE_PATH}"
chmod 0644 "${PACKAGE_PATH}"

if [[ ! -x "${WRAPPER}" ]]; then
  section "Installing privileged test wrapper"
  echo "This step may prompt for your Steam Deck sudo password."
  (
    cd "${TESTING_DIR}"
    sudo ./install_decky_testing_sudoers.sh
  )
fi

section "Installing latest Decky Mod Manager package"
echo "The installer updates the Decky plugin package without rebooting by default."
sudo "${WRAPPER}"
