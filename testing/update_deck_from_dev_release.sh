#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
TESTING_DIR="${TESTING_DIR:-${HOME}/.testing}"
REPO="justyntemme/dmm"
RELEASE="${DMM_RELEASE:-}"
PACKAGE_NAME="${PLUGIN_NAME}.tar.gz"
CHECKSUM_NAME="SHA256SUMS"
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

resolve_release_assets() {
  if [[ -n "${RELEASE}" ]]; then
    if [[ ! "${RELEASE}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "DMM_RELEASE must be a version tag such as v0.0.2" >&2
      return 1
    fi
    printf 'https://github.com/%s/releases/download/%s/%s\n' "${REPO}" "${RELEASE}" "${PACKAGE_NAME}"
    printf 'https://github.com/%s/releases/download/%s/%s\n' "${REPO}" "${RELEASE}" "${CHECKSUM_NAME}"
    printf '%s\n' "${RELEASE}"
    return
  fi
  python3 - "$REPO" "$PACKAGE_NAME" "$CHECKSUM_NAME" <<'PY'
import json
import re
import sys
import urllib.request

repo = sys.argv[1]
package_name = sys.argv[2]
checksum_name = sys.argv[3]
request = urllib.request.Request(
    f"https://api.github.com/repos/{repo}/releases/latest",
    headers={"User-Agent": "Decky-Mod-Manager-Updater"},
)
with urllib.request.urlopen(request, timeout=30) as response:
    if response.geturl() != request.full_url:
        raise SystemExit("GitHub latest release lookup left the pinned API origin")
    release = json.loads(response.read().decode("utf-8"))
tag = str(release.get("tag_name") or "")
if not re.fullmatch(r"v?[0-9]+\.[0-9]+\.[0-9]+", tag):
    raise SystemExit("latest GitHub release does not use a supported version tag")
expected = {
    package_name: f"https://github.com/{repo}/releases/download/{tag}/{package_name}",
    checksum_name: f"https://github.com/{repo}/releases/download/{tag}/{checksum_name}",
}
found = {}
for asset in release.get("assets") or []:
    name = str(asset.get("name") or "")
    url = str(asset.get("browser_download_url") or "")
    if name in expected:
        if url != expected[name]:
            raise SystemExit(f"release asset URL is not pinned for {name}")
        found[name] = url
missing = [name for name in expected if name not in found]
if missing:
    raise SystemExit(f"latest release {tag} does not include {', '.join(missing)}")
print(found[package_name])
print(found[checksum_name])
print(tag)
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
  echo "curl is required to download DMM release assets" >&2
  exit 1
fi
if ! command -v sha256sum >/dev/null 2>&1; then
  echo "sha256sum is required to authenticate DMM release packages" >&2
  exit 1
fi
if [[ -z "${RELEASE}" ]] && ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to resolve the latest GitHub release. Set DMM_RELEASE to select a release directly." >&2
  exit 1
fi
mapfile -t RELEASE_ASSETS < <(resolve_release_assets)
if [[ "${#RELEASE_ASSETS[@]}" -ne 3 ]]; then
  echo "Unable to resolve the pinned DMM package, checksum, and release tag." >&2
  exit 1
fi
PACKAGE_URL="${RELEASE_ASSETS[0]}"
CHECKSUM_URL="${RELEASE_ASSETS[1]}"
RELEASE="${RELEASE_ASSETS[2]}"

section "Staging Decky Mod Manager test helpers"
mkdir -p "${TESTING_DIR}"
for helper in "${HELPERS[@]}"; do
  install -m 0755 "${ROOT_DIR}/testing/${helper}" "${TESTING_DIR}/${helper}"
done

section "Downloading ${RELEASE} package and checksum"
download "${CHECKSUM_URL}" "${CHECKSUM_PATH}"
download "${PACKAGE_URL}" "${PACKAGE_PATH}"
chmod 0644 "${PACKAGE_PATH}"

expected_digest="$(awk -v name="${PACKAGE_NAME}" '$2 == name || $2 == "*" name { print tolower($1) }' "${CHECKSUM_PATH}")"
if [[ ! "${expected_digest}" =~ ^[0-9a-f]{64}$ ]]; then
  echo "${CHECKSUM_NAME} does not contain exactly one valid digest for ${PACKAGE_NAME}." >&2
  exit 1
fi
actual_digest="$(sha256sum "${PACKAGE_PATH}" | awk '{print tolower($1)}')"
if [[ "${actual_digest}" != "${expected_digest}" ]]; then
  echo "Release package SHA-256 mismatch." >&2
  exit 1
fi
echo "Verified ${PACKAGE_NAME} SHA-256: ${actual_digest}"

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
