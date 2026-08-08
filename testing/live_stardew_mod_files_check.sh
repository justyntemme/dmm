#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"
STATE_DIR="${STATE_DIR:-${HOME}/.local/state/decky-mod-manager}"
TOKEN_FILE="${DMM_TOKEN_FILE:-${STATE_DIR}/api-token}"
API_TOKEN="${DMM_AUTH_TOKEN:-}"
GAME_PATH="${GAME_PATH:-}"
REQUIRE_RUNTIME="${REQUIRE_RUNTIME:-0}"
REQUIRE_SMAPI_ROOT="${REQUIRE_SMAPI_ROOT:-0}"
SSH_TARGET="${SSH_TARGET:-}"
SSH_KEY="${SSH_KEY:-${HOME}/.ssh/decky_mod_manager_test}"
REMOTE_DATA_DIR="${REMOTE_DATA_DIR:-/home/deck/.local/share/decky-mod-manager}"

shell_quote() {
  printf "%q" "$1"
}

if [[ -n "${SSH_TARGET}" && "${DMM_REMOTE_CHECK:-0}" != "1" ]]; then
  ssh_args=(
    -o IdentityAgent=none
    -o BatchMode=yes
    -o IdentitiesOnly=yes
    -o ConnectTimeout=6
  )
  if [[ -f "${SSH_KEY}" ]]; then
    ssh_args=(-i "${SSH_KEY}" "${ssh_args[@]}")
  fi
  remote_base_url="${REMOTE_BASE_URL:-http://127.0.0.1:${PORT}}"
  remote_command=$(
    printf 'DMM_REMOTE_CHECK=1 PORT=%s HOST=%s BASE_URL=%s APP_ID=%s DATA_DIR=%s GAME_PATH=%s REQUIRE_RUNTIME=%s REQUIRE_SMAPI_ROOT=%s bash -s' \
      "$(shell_quote "${PORT}")" \
      "$(shell_quote "127.0.0.1")" \
      "$(shell_quote "${remote_base_url}")" \
      "$(shell_quote "${APP_ID}")" \
      "$(shell_quote "${REMOTE_DATA_DIR}")" \
      "$(shell_quote "${GAME_PATH}")" \
      "$(shell_quote "${REQUIRE_RUNTIME}")" \
      "$(shell_quote "${REQUIRE_SMAPI_ROOT}")"
  )
  exec ssh "${ssh_args[@]}" "${SSH_TARGET}" "${remote_command}" < "$0"
fi

section() {
  printf '\n==> %s\n' "$1"
}

section "Stardew deployed mod file visibility check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "data_dir=${DATA_DIR}"
echo "api_auth=$([[ -n "${API_TOKEN}" ]] && printf env || ([[ -f "${TOKEN_FILE}" ]] && printf token-file || printf none))"

if [[ -z "${API_TOKEN}" && -f "${TOKEN_FILE}" ]]; then
  API_TOKEN="$(<"${TOKEN_FILE}")"
fi
curl_args=(-fsS)
if [[ -n "${API_TOKEN}" ]]; then
  curl_args+=(-H "X-DMM-Token: ${API_TOKEN}")
fi

diagnostics_json="$(curl "${curl_args[@]}" "${BASE_URL}/api/games/${APP_ID}/diagnostics")"
deployment_json="$(curl "${curl_args[@]}" "${BASE_URL}/api/games/${APP_ID}/deploy/status")"

export DIAGNOSTICS_JSON="${diagnostics_json}"
export DEPLOYMENT_JSON="${deployment_json}"
export DATA_DIR
export GAME_PATH
export REQUIRE_RUNTIME
export REQUIRE_SMAPI_ROOT

python3 - <<'PY'
import json
import os
import pathlib
import sys


def load_env_json(name):
    try:
        return json.loads(os.environ[name])
    except KeyError:
        print(f"missing {name}", file=sys.stderr)
        sys.exit(2)
    except json.JSONDecodeError as exc:
        print(f"{name} is not valid JSON: {exc}", file=sys.stderr)
        sys.exit(2)


diag = load_env_json("DIAGNOSTICS_JSON")
deployment = load_env_json("DEPLOYMENT_JSON")
data_dir = pathlib.Path(os.environ["DATA_DIR"]).resolve()
require_runtime = os.environ.get("REQUIRE_RUNTIME", "0") != "0"
require_smapi_root = os.environ.get("REQUIRE_SMAPI_ROOT", "0") != "0"
game_path_env = os.environ.get("GAME_PATH", "").strip()
game_path = pathlib.Path(game_path_env).expanduser() if game_path_env else pathlib.Path(diag.get("game", {}).get("game_path", ""))
game_path = game_path.resolve()
mods_dir = game_path / "Mods"
runtime_requirements = list(diag.get("runtime_requirements") or [])

failures = []
warnings = []

if not deployment.get("deployed"):
    failures.append("DMM has no active deployment manifest")
if int(deployment.get("file_count") or 0) < 1:
    failures.append("DMM active deployment has no files")
if not mods_dir.is_dir():
    failures.append(f"Stardew Mods folder does not exist: {mods_dir}")

manifest_links = []
dmm_links = []
regular_or_external = []
if mods_dir.is_dir():
    for manifest in sorted(mods_dir.glob("*/manifest.json")):
        manifest_links.append(manifest)
        if not manifest.is_symlink():
            regular_or_external.append((manifest, "manifest is not a symlink"))
            continue
        try:
            target = manifest.resolve(strict=True)
        except FileNotFoundError:
            regular_or_external.append((manifest, "manifest symlink target is missing"))
            continue
        try:
            target.relative_to(data_dir)
        except ValueError:
            regular_or_external.append((manifest, f"manifest target is outside DMM data: {target}"))
            continue
        dmm_links.append((manifest, target))

enabled = int(diag.get("enabled_mods") or 0)
if enabled > 0 and not dmm_links:
    failures.append("no DMM-managed SMAPI manifest symlinks are visible in the Stardew Mods folder")
if len(dmm_links) < enabled:
    warnings.append(f"{len(dmm_links)} DMM-managed manifest link(s) visible for {enabled} enabled profile mod(s)")

sample_files = list(deployment.get("sample_files") or [])
missing_samples = []
external_samples = []
for sample in sample_files:
    path = pathlib.Path(sample)
    if not path.exists() and not path.is_symlink():
        missing_samples.append(sample)
        continue
    if path.is_symlink():
        try:
            target = path.resolve(strict=True)
            target.relative_to(data_dir)
        except (FileNotFoundError, ValueError):
            external_samples.append(sample)

if missing_samples:
    failures.append("deployment sample file(s) are missing: " + ", ".join(missing_samples))
if external_samples:
    failures.append("deployment sample symlink(s) do not point into DMM data: " + ", ".join(external_samples))

smapi_root_markers = [
    game_path / "StardewModdingAPI",
    game_path / "StardewModdingAPI.dll",
    game_path / "StardewModdingAPI.deps.json",
    game_path / "smapi-internal" / "SMAPI.Toolkit.CoreInterfaces.dll",
]
smapi_root_links = []
smapi_root_missing = []
smapi_root_unmanaged = []
for marker in smapi_root_markers:
    if not marker.exists() and not marker.is_symlink():
        smapi_root_missing.append(marker)
        continue
    if not marker.is_symlink():
        smapi_root_unmanaged.append((marker, "marker is not a symlink"))
        continue
    try:
        target = marker.resolve(strict=True)
        target.relative_to(data_dir)
    except FileNotFoundError:
        smapi_root_unmanaged.append((marker, "marker symlink target is missing"))
        continue
    except ValueError:
        smapi_root_unmanaged.append((marker, f"marker target is outside DMM data: {target}"))
        continue
    smapi_root_links.append((marker, target))

if require_smapi_root:
    if smapi_root_missing:
        failures.append(
            "required DMM-managed SMAPI root marker(s) are missing: "
            + ", ".join(str(path) for path in smapi_root_missing)
        )
    if smapi_root_unmanaged:
        failures.append(
            "required SMAPI root marker(s) are unmanaged: "
            + ", ".join(f"{path} ({reason})" for path, reason in smapi_root_unmanaged)
        )

if runtime_requirements:
    for requirement in runtime_requirements:
        if requirement.get("required") and requirement.get("status") != "ok":
            message = (
                f"{requirement.get('name', requirement.get('id', 'runtime'))} runtime requirement "
                f"is {requirement.get('status', 'unknown')}: {requirement.get('message', '')}"
            )
            if require_runtime:
                failures.append(message)
            else:
                warnings.append(message)
else:
    message = "runtime status is unavailable from diagnostics"
    if require_runtime:
        failures.append(message)
    else:
        warnings.append(message)

print("summary:")
print(f"  game_path={game_path}")
print(f"  mods_dir={mods_dir}")
print(f"  enabled_profile_mods={enabled}")
print(f"  deployment_files={deployment.get('file_count')}")
print(f"  manifest_links_seen={len(manifest_links)}")
print(f"  dmm_manifest_links={len(dmm_links)}")
print(f"  dmm_smapi_root_links={len(smapi_root_links)}")
if runtime_requirements:
    print("  runtime_requirements=")
    for requirement in runtime_requirements:
        print(f"    - {requirement.get('name', requirement.get('id', 'runtime'))}: {requirement.get('status', 'unknown')}")
for manifest, target in dmm_links[:8]:
    print(f"  dmm_mod={manifest.parent.name} -> {target}")
for marker, target in smapi_root_links:
    print(f"  dmm_smapi_root={marker.name} -> {target}")

if regular_or_external:
    print("\nnon_dmm_or_unmanaged_manifests:")
    for manifest, reason in regular_or_external[:10]:
        print(f"  - {manifest}: {reason}")

if warnings:
    print("\nwarnings:")
    for warning in warnings:
        print(f"  - {warning}")

if failures:
    print("\nfailures:")
    for failure in failures:
        print(f"  - {failure}")
    sys.exit(1)

print("\nStardew deployed mod files are visible as DMM-managed links")
PY
