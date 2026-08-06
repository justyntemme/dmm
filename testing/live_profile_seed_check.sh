#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
PROFILE_NAME="${PROFILE_NAME:-DMM Live Seed Profile Check}"
DATA_DIR="${DATA_DIR:-${HOME}/.local/share/decky-mod-manager}"
DB_PATH="${DB_PATH:-${DATA_DIR}/db/dmm.sqlite}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live seeded profile check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "profile_name=${PROFILE_NAME}"
echo "db_path=${DB_PATH}"

export BASE_URL
export APP_ID
export PROFILE_NAME
export DB_PATH

python3 - <<'PY'
import json
import os
import pathlib
import sqlite3
import urllib.error
import urllib.request


base = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ["APP_ID"]
profile_name = os.environ["PROFILE_NAME"]
db_path = pathlib.Path(os.environ["DB_PATH"]).expanduser()


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


def profile_rows(conn, profile_id):
    return [
        (int(row[0]), int(row[1]), int(row[2]))
        for row in conn.execute(
            """
            SELECT installed_mod_id, enabled, priority
            FROM profile_mods
            WHERE profile_id = ?
            ORDER BY installed_mod_id ASC
            """,
            (profile_id,),
        )
    ]


profiles = request("GET", f"/api/games/{app_id}/profiles")
if not profiles:
    raise RuntimeError("game has no profiles")

for profile in profiles:
    if profile.get("name") == profile_name:
        request("DELETE", f"/api/profiles/{profile['id']}")

profiles = request("GET", f"/api/games/{app_id}/profiles")
source = next((profile for profile in profiles if profile.get("is_default")), profiles[0])
source_id = int(source["id"])

conn = sqlite3.connect(db_path)
try:
    source_rows = profile_rows(conn, source_id)
    if not source_rows:
        raise RuntimeError("source profile has no mod memberships to seed")
    source_enabled = sum(1 for _, enabled, _ in source_rows if enabled != 0)

    created = request(
        "POST",
        f"/api/games/{app_id}/profiles",
        {"name": profile_name, "source_profile_id": source_id},
    )
    target_id = int(created["id"])
    target_rows = profile_rows(conn, target_id)
    if target_rows != source_rows:
        raise RuntimeError(f"seeded profile rows did not match source: {target_rows} != {source_rows}")
    if int(created.get("mod_count") or 0) != len(source_rows):
        raise RuntimeError(f"created profile mod_count mismatch: {created}")
    if int(created.get("enabled_mod_count") or 0) != source_enabled:
        raise RuntimeError(f"created profile enabled_mod_count mismatch: {created}")
finally:
    conn.close()

deleted = request("DELETE", f"/api/profiles/{target_id}")
if int((deleted.get("deleted") or {}).get("id") or 0) != target_id:
    raise RuntimeError(f"cleanup returned unexpected profile: {deleted}")

print("summary:")
print(f"  source_profile={source['name']} ({source_id})")
print(f"  seeded_profile={profile_name} ({target_id})")
print(f"  copied_memberships={len(source_rows)}")
print(f"  copied_enabled={source_enabled}")
print("  cleanup=deleted")
print("\nSeeded profile check passed")
PY
