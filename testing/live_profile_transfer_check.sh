#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
PROFILE_NAME="${PROFILE_NAME:-DMM Live Transfer Check}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"
DB_PATH="${DB_PATH:-${DATA_DIR}/db/dmm.sqlite}"
KEEP_PROFILE="${KEEP_PROFILE:-0}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live profile transfer check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "profile_name=${PROFILE_NAME}"
echo "db_path=${DB_PATH}"
echo "keep_profile=${KEEP_PROFILE}"

export BASE_URL
export APP_ID
export PROFILE_NAME
export DB_PATH
export KEEP_PROFILE

python3 - <<'PY'
import json
import os
import pathlib
import sqlite3
import sys
import urllib.error
import urllib.request


base = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ["APP_ID"]
profile_name = os.environ["PROFILE_NAME"]
db_path = pathlib.Path(os.environ["DB_PATH"]).expanduser()
keep_profile = os.environ.get("KEEP_PROFILE", "0") != "0"


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


def profile_mod_count(conn, profile_id, installed_mod_id):
    row = conn.execute(
        "SELECT COUNT(*) FROM profile_mods WHERE profile_id = ? AND installed_mod_id = ?",
        (profile_id, installed_mod_id),
    ).fetchone()
    return int(row[0] or 0)


profiles = request("GET", f"/api/games/{app_id}/profiles")
if not profiles:
    raise RuntimeError("game has no profiles")

source = next((profile for profile in profiles if profile.get("is_default")), profiles[0])
target = next((profile for profile in profiles if profile.get("name") == profile_name), None)
if target is None:
    target = request("POST", f"/api/games/{app_id}/profiles", {"name": profile_name})
    profiles = request("GET", f"/api/games/{app_id}/profiles")

if int(target["id"]) == int(source["id"]):
    raise RuntimeError("test target profile resolved to the active source profile")

mods = request("GET", f"/api/games/{app_id}/mods")
eligible = [mod for mod in mods if int(mod.get("profile_id") or 0) == int(source["id"])]
if not eligible:
    raise RuntimeError("active profile has no installed mods to copy")

conn = sqlite3.connect(db_path)
try:
    selected = None
    for mod in eligible:
        if profile_mod_count(conn, int(target["id"]), int(mod["id"])) == 0:
            selected = mod
            break
    if selected is None:
        selected = eligible[0]
        request("DELETE", f"/api/profiles/{target['id']}/mods/{selected['id']}")
        if profile_mod_count(conn, int(target["id"]), int(selected["id"])) != 0:
            raise RuntimeError("could not clear existing test-profile membership")

    mod_id = int(selected["id"])
    source_id = int(source["id"])
    target_id = int(target["id"])
    api_staging_value = str(selected.get("staging_path") or "").strip()
    staging_path = pathlib.Path(api_staging_value).expanduser() if api_staging_value else None

    if profile_mod_count(conn, source_id, mod_id) != 1:
        raise RuntimeError("selected mod is not a member of the source profile")

    copied = request(
        "POST",
        f"/api/profiles/{source_id}/mods/{mod_id}/copy",
        {"target_profile_id": target_id, "enabled": False},
    )
    copied_mod = copied.get("mod") or {}
    if int(copied_mod.get("profile_id") or 0) != target_id:
        raise RuntimeError(f"copy response did not target profile {target_id}: {copied_mod}")
    if copied_mod.get("enabled"):
        raise RuntimeError("copied mod should be disabled in the target profile")

    if profile_mod_count(conn, source_id, mod_id) != 1:
        raise RuntimeError("copy removed the source profile membership")
    if profile_mod_count(conn, target_id, mod_id) != 1:
        raise RuntimeError("copy did not create the target profile membership")

    removed = request("DELETE", f"/api/profiles/{target_id}/mods/{mod_id}")
    if (removed.get("mod") or {}).get("id") != mod_id:
        raise RuntimeError(f"remove response did not return selected mod: {removed}")

    if profile_mod_count(conn, target_id, mod_id) != 0:
        raise RuntimeError("profile remove left the target profile membership")
    if profile_mod_count(conn, source_id, mod_id) != 1:
        raise RuntimeError("profile remove affected the source profile membership")

    installed_row = conn.execute("SELECT staging_path FROM installed_mods WHERE id = ?", (mod_id,)).fetchone()
    if installed_row is None:
        raise RuntimeError("profile remove deleted the installed mod row")
    stored_staging_value = str(installed_row[0] or "").strip()
    stored_staging = pathlib.Path(stored_staging_value).expanduser() if stored_staging_value else None
    if staging_path is not None and stored_staging is not None and staging_path != stored_staging:
        raise RuntimeError(f"staging path changed unexpectedly: {staging_path} -> {stored_staging}")
    if stored_staging is not None and not stored_staging.exists():
        raise RuntimeError(f"profile remove deleted staged mod files: {stored_staging}")
finally:
    conn.close()

cleanup_status = "kept"
if not keep_profile:
    deleted = request("DELETE", f"/api/profiles/{target_id}")
    cleanup_status = "deleted"
    if (deleted.get("deleted") or {}).get("id") != target_id:
        raise RuntimeError(f"temporary profile cleanup returned unexpected result: {deleted}")

print("summary:")
print(f"  source_profile={source['name']} ({source['id']})")
print(f"  target_profile={target['name']} ({target['id']})")
print(f"  copied_mod={selected.get('name')} ({selected['id']})")
print("  source_membership=kept")
print("  target_membership=removed_after_check")
print("  staging=kept")
print(f"  test_profile={cleanup_status}")
print("\nProfile transfer check passed")
PY
