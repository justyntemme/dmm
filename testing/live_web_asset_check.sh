#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"

section() {
  printf '\n==> %s\n' "$1"
}

section "Web UI asset check"
echo "base_url=${BASE_URL}"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

if ! curl -fsS "${BASE_URL}/" >"${tmp_dir}/index.html"; then
  echo "web UI index is not reachable at ${BASE_URL}/" >&2
  exit 1
fi

if ! grep -q '<div id="app"></div>' "${tmp_dir}/index.html"; then
  echo "web UI index does not contain the Svelte mount point" >&2
  exit 1
fi

mapfile -t assets < <(python3 - "${tmp_dir}/index.html" <<'PY'
import re
import sys

body = open(sys.argv[1], encoding="utf-8").read()
for asset in sorted(set(re.findall(r'/(assets/[^"\'<> ]+)', body))):
    print(asset)
PY
)

if [[ "${#assets[@]}" -eq 0 ]]; then
  echo "web UI index does not reference built assets" >&2
  exit 1
fi

for asset in "${assets[@]}"; do
  target="${tmp_dir}/$(basename "${asset}")"
  if ! curl -fsS "${BASE_URL}/${asset}" >"${target}"; then
    echo "web UI asset is not reachable: /${asset}" >&2
    exit 1
  fi
  if [[ ! -s "${target}" ]]; then
    echo "web UI asset is empty: /${asset}" >&2
    exit 1
  fi
  echo "asset_ok=/${asset}"
done

js_checked=0
for target in "${tmp_dir}"/*.js; do
  [[ -f "${target}" ]] || continue
  if grep -qF "Selected Profile" "${target}"; then
    js_checked=1
    for needle in \
      "Installed, disabled in this profile" \
      "Add Mod" \
      "Install to Profile" \
      "mods in the same on/off state" \
      "Explore Mods" \
      "Open on Deck" \
      "Advanced Profile Tools" \
      "These Deck behavior switches are managed from the Decky sidebar settings."
    do
      if ! grep -qF "${needle}" "${target}"; then
        echo "web UI asset is missing MVP UI text: ${needle}" >&2
        exit 1
      fi
    done
  fi
done

if [[ "${js_checked}" -eq 0 ]]; then
  echo "web UI assets did not contain the expected profile-first MVP UI" >&2
  exit 1
fi

echo "Web UI assets are reachable"
