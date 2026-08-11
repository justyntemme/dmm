#!/usr/bin/env bash
set -euo pipefail

WRAPPER_VERSION="3"
PLUGIN_NAME="${PLUGIN_NAME:-decky-mod-manager}"
TESTING_DIR="${TESTING_DIR:-/home/deck/.testing}"
PRIVILEGED_BIN="${PRIVILEGED_BIN:-/opt/decky-mod-manager-testing/bin}"
INSTALLER="${INSTALLER:-${PRIVILEGED_BIN}/install_decky_plugin_from_package.sh}"
PACKAGE="${PACKAGE:-${TESTING_DIR}/${PLUGIN_NAME}.tar.gz}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "This wrapper must run as root through sudo." >&2
  exit 1
fi

case "${PACKAGE}" in
  "${TESTING_DIR}/${PLUGIN_NAME}.tar.gz"|"${TESTING_DIR}/${PLUGIN_NAME}.zip")
    ;;
  *)
    echo "Refusing package outside ${TESTING_DIR}: ${PACKAGE}" >&2
    exit 1
    ;;
esac

if [[ ! -f "${INSTALLER}" ]]; then
  echo "Installer not found: ${INSTALLER}" >&2
  exit 1
fi
if [[ ! -f "${PACKAGE}" ]]; then
  echo "Package not found: ${PACKAGE}" >&2
  exit 1
fi

owner="$(stat -c '%U:%G' "${INSTALLER}")"
mode="$(stat -c '%a' "${INSTALLER}")"
if [[ "${owner}" != "root:root" || "${mode}" != "755" ]]; then
  echo "Unsafe installer ownership/mode: ${owner} ${mode}; expected root:root 755." >&2
  exit 1
fi

package_owner="$(stat -c '%U:%G' "${PACKAGE}")"
package_mode="$(stat -c '%a' "${PACKAGE}")"
if [[ "${package_owner}" != "deck:deck" && "${package_owner}" != "root:root" ]]; then
  echo "Unsafe package ownership: ${package_owner}; expected deck:deck or root:root." >&2
  exit 1
fi
if [[ "${package_mode}" != "644" ]]; then
  echo "Unsafe package mode: ${package_mode}; expected 644." >&2
  exit 1
fi

echo "Running privileged Decky Mod Manager installer wrapper v${WRAPPER_VERSION}"
exec env PACKAGE="${PACKAGE}" "${INSTALLER}"
