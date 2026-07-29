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
import sqlite3
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
    db_path = data_dir / "db" / "dmm.sqlite"
    if not db_path.exists():
        raise RuntimeError(f"DMM database is missing: {db_path}")
    # The public mods API intentionally does not expose raw staged manifests.
    # This live verifier reads DMM's local DB directly to validate file effects.
    conn = sqlite3.connect(db_path)
    row = conn.execute(
        """
        select im.checksum_manifest_json
        from installed_mods im
        where im.id = ?
        """,
        (mod.get("id"),),
    ).fetchone()
    conn.close()
    if not row or not row[0]:
        return []
    parsed = json.loads(row[0])
    files = parsed.get("files") if isinstance(parsed, dict) else parsed
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


diag, mods, deployment = fetch_state()
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
for mod in enabled_mods:
    targets = targets_by_mod.get(mod["id"], [])
    unique = [target for target in targets if target_counts.get(target) == 1]
    if unique:
        selected = mod
        selected_targets = unique
        break

if selected is None:
    raise RuntimeError("could not find an enabled mod with unique deployment targets")

profile_id = selected["profile_id"]
mod_id = selected["id"]
name = selected.get("name") or f"mod {mod_id}"
target_paths = [game_path / target for target in selected_targets]
print(f"selected_mod={name} id={mod_id} profile={profile_id}")
print(f"unique_targets={len(target_paths)}")

restored = False
try:
    set_enabled(profile_id, mod_id, False)
    preview = request("GET", f"/api/games/{app_id}/deploy/preview")
    removes = [action for action in preview.get("actions", []) if action.get("operation") == "remove"]
    if not removes:
        raise RuntimeError("disabling the selected mod did not produce any remove actions")
    deploy_profile()
    time.sleep(0.2)

    still_present = [str(path) for path in target_paths if path.exists() or path.is_symlink()]
    if still_present:
        raise RuntimeError("disabled mod target(s) still exist after deploy: " + ", ".join(still_present[:8]))

    set_enabled(profile_id, mod_id, True)
    deploy_profile()
    restored = True

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
            deploy_profile()
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
