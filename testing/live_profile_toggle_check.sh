#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live profile toggle deployment check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "data_dir=${DATA_DIR}"

export BASE_URL
export APP_ID
export DATA_DIR

python3 - <<'PY'
import json
import os
import pathlib
import sys
import time
import urllib.error
import urllib.request


base = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ["APP_ID"]
data_dir = pathlib.Path(os.environ["DATA_DIR"]).expanduser().resolve()


def request(method, path, body=None):
    payload = None
    headers = {}
    if body is not None:
        payload = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base + path, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read()
            if not raw:
                return {}
            return json.loads(raw)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path} failed: {exc.code} {detail}") from exc


def manifest_targets(mod):
    parsed = mod.get("manifest")
    if not isinstance(parsed, dict):
        return []
    files = parsed.get("files")
    targets = []
    for item in files or []:
        target = item.get("target_relative")
        if not target:
            continue
        target_path = pathlib.PurePosixPath(str(target).replace("\\", "/"))
        if target_path.is_absolute() or ".." in target_path.parts:
            continue
        targets.append(str(target_path))
    return targets


def fetch_state():
    diag = request("GET", f"/api/games/{app_id}/diagnostics")
    mods = request("GET", f"/api/games/{app_id}/mods")
    deployment = request("GET", f"/api/games/{app_id}/deploy/status")
    return diag, mods, deployment


def deploy_profile():
    result = request("POST", f"/api/games/{app_id}/deploy")
    job = result.get("job") or {}
    if job.get("status") != "completed":
        raise RuntimeError(f"deploy job did not complete: {job}")
    return result


def set_enabled(profile_id, mod_id, enabled):
    return request("PUT", f"/api/profiles/{profile_id}/mods/{mod_id}", {"enabled": enabled})


def is_runtime_mod(mod):
    mod_type = str(mod.get("mod_type") or "").lower()
    name = str(mod.get("name") or "").lower()
    return mod_type in {"smapi", "launch-tool", "script-extender"} or name.startswith("smapi ")


diag, mods, deployment = fetch_state()
baseline_preview = request("GET", f"/api/games/{app_id}/deploy/preview")
game = diag.get("game") or {}
game_path = pathlib.Path(game.get("game_path") or "").expanduser().resolve()
if not game_path:
    raise RuntimeError("diagnostics did not include a game path")
if not deployment.get("deployed"):
    raise RuntimeError("no active deployment exists; deploy the selected profile before running this check")

enabled_mods = [mod for mod in mods if mod.get("enabled") and mod.get("status") == "staged"]
if not enabled_mods:
    raise RuntimeError("no enabled profile mods are available to toggle")

target_counts = {}
targets_by_mod = {}
for mod in enabled_mods:
    targets = manifest_targets(mod)
    targets_by_mod[mod["id"]] = targets
    for target in targets:
        target_counts[target] = target_counts.get(target, 0) + 1

selected = None
selected_targets = []
toggle_candidates = [mod for mod in enabled_mods if not is_runtime_mod(mod)] or enabled_mods
for mod in toggle_candidates:
    targets = targets_by_mod.get(mod["id"], [])
    unique = [target for target in targets if target_counts.get(target) == 1]
    if unique:
        selected = mod
        selected_targets = unique
        break

if selected is None:
    preview_targets_by_mod = {}
    preview_target_counts = {}
    for action in baseline_preview.get("actions") or []:
        mod_id = action.get("installed_mod_id")
        target = action.get("target_relative")
        if mod_id is None or not target:
            continue
        target_path = pathlib.PurePosixPath(str(target).replace("\\", "/"))
        if target_path.is_absolute() or ".." in target_path.parts:
            continue
        preview_targets_by_mod.setdefault(mod_id, set()).add(str(target_path))
        preview_target_counts[str(target_path)] = preview_target_counts.get(str(target_path), 0) + 1
    for mod in toggle_candidates:
        targets = sorted(preview_targets_by_mod.get(mod["id"], set()))
        unique = [target for target in targets if preview_target_counts.get(target) == 1]
        if unique:
            selected = mod
            selected_targets = unique
            break

if selected is None:
    raise RuntimeError("could not find an enabled mod with unique deployment targets in the active API preview")

profile_id = selected["profile_id"]
mod_id = selected["id"]
name = selected.get("name") or f"mod {mod_id}"
target_paths = [game_path / target for target in selected_targets]
can_assert_files = game_path.exists() and data_dir.exists()
print(f"selected_mod={name} id={mod_id} profile={profile_id}")
print(f"unique_targets={len(target_paths)}")
if not can_assert_files:
    print("filesystem_assertions=skipped (run this script on the Steam Deck to inspect deployed files)")

restored = False
try:
    disabled = set_enabled(profile_id, mod_id, False)
    disabled_apply = disabled.get("apply") or {}
    if disabled_apply.get("status") != "applied":
        raise RuntimeError(f"disabling the selected mod did not apply cleanly: {disabled_apply}")
    time.sleep(0.2)

    if can_assert_files:
        still_present = [str(path) for path in target_paths if path.exists() or path.is_symlink()]
        if still_present:
            raise RuntimeError("disabled mod target(s) still exist after deploy: " + ", ".join(still_present[:8]))

    enabled = set_enabled(profile_id, mod_id, True)
    enabled_apply = enabled.get("apply") or {}
    if enabled_apply.get("status") != "applied":
        raise RuntimeError(f"re-enabling the selected mod did not apply cleanly: {enabled_apply}")
    restored = True

    if can_assert_files:
        missing = []
        external = []
        for path in target_paths:
            if not path.exists() and not path.is_symlink():
                missing.append(str(path))
                continue
            if path.is_symlink():
                try:
                    target = path.resolve(strict=True)
                    target.relative_to(data_dir)
                except (FileNotFoundError, ValueError):
                    external.append(str(path))
        if missing:
            raise RuntimeError("re-enabled mod target(s) are missing after deploy: " + ", ".join(missing[:8]))
        if external:
            raise RuntimeError("re-enabled mod symlink(s) do not point into DMM data: " + ", ".join(external[:8]))
finally:
    if not restored:
        try:
            set_enabled(profile_id, mod_id, True)
            print("restored selected mod after failed check")
        except Exception as exc:
            print(f"failed to restore selected mod: {exc}", file=sys.stderr)

diag_after, mods_after, deployment_after = fetch_state()
selected_after = next((mod for mod in mods_after if mod.get("id") == mod_id), None)
if not selected_after or not selected_after.get("enabled"):
    raise RuntimeError("selected mod was not restored to enabled state")
if not deployment_after.get("deployed"):
    raise RuntimeError("deployment manifest missing after restore")

print("summary:")
print(f"  toggled_mod={name}")
print(f"  deployment_files={deployment_after.get('file_count')}")
print(f"  preview_conflicts={(diag_after.get('preview') or {}).get('conflicts', 0)}")
print("\nProfile toggle deployment check passed")
PY
