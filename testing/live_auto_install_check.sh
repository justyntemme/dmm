#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
RESTORE_SETTING="${RESTORE_SETTING:-1}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live automatic install check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "timeout_seconds=${TIMEOUT_SECONDS}"
echo "restore_setting=${RESTORE_SETTING}"
export BASE_URL APP_ID TIMEOUT_SECONDS RESTORE_SETTING

python3 - <<'PY'
import json
import os
import sys
import time
import urllib.error
import urllib.request


base_url = os.environ["BASE_URL"].rstrip("/")
app_id = os.environ.get("APP_ID", "413150")
timeout = int(os.environ.get("TIMEOUT_SECONDS", "180"))
restore_setting = os.environ.get("RESTORE_SETTING", "1") != "0"


def request(method, path, payload=None):
    data = None
    headers = {}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base_url + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path} failed: HTTP {exc.code}: {detail}") from exc
    if not body:
        return None
    return json.loads(body.decode("utf-8"))


def jobs_list():
    jobs = request("GET", "/api/jobs")
    if isinstance(jobs, dict):
        return list(jobs.get("jobs") or [])
    return list(jobs or [])


def matches_game(job):
    payload = job.get("payload") or {}
    if payload.get("app_id") == app_id:
        return True
    domain = str(payload.get("game_domain") or "").lower()
    domains = {
        "413150": ("stardewvalley",),
    }
    if domain and domain in domains.get(app_id, ()):
        return True
    haystack = f"{job.get('title', '')} {job.get('message', '')}".lower()
    aliases = {
        "413150": ("stardewvalley", "stardew", "413150"),
    }
    return any(alias in haystack for alias in aliases.get(app_id, (app_id,)))


def validate_job_payload(job):
    payload = job.get("payload") or {}
    missing = [key for key in ("app_id", "catalog", "game_domain", "mod_id", "file_id") if not payload.get(key)]
    if missing:
        raise RuntimeError(
            "fresh captured-install job is missing structured payload fields "
            + ", ".join(missing)
            + f": {payload}"
        )
    if payload.get("app_id") != app_id:
        raise RuntimeError(f"fresh captured-install job payload app_id={payload.get('app_id')} does not match {app_id}")


status = request("GET", "/api/status")
install = status.get("install") or {}
previous_auto_install = bool(install.get("auto_install_captured_downloads"))
previous_auto_enable = bool(install.get("auto_enable_installed_mods"))

baseline_ids = {job.get("id") for job in jobs_list()}

print("previous_auto_install=", previous_auto_install, sep="")
print("previous_auto_enable=", previous_auto_enable, sep="")
print("baseline_jobs=", len(baseline_ids), sep="")

request("PUT", "/api/settings/install", {
    "auto_install_captured_downloads": True,
    "auto_enable_installed_mods": previous_auto_enable,
})
print("\nAuto-install is enabled for this check.")
print("Now use the Deck browser to click a fresh Nexus Mod Manager Download link.")
print("This script will pass when the new captured install runs/completes without stopping at the local install gate.")

seen_states = {}
deadline = time.monotonic() + timeout
exit_code = 1

try:
    while time.monotonic() < deadline:
        candidates = [
            job for job in jobs_list()
            if job.get("type") == "captured-install"
            and job.get("id") not in baseline_ids
            and matches_game(job)
        ]
        if not candidates:
            time.sleep(2)
            continue
        candidates.sort(key=lambda item: item.get("updated_at", ""), reverse=True)
        job = candidates[0]
        job_id = job.get("id")
        state = (job.get("status"), job.get("message"))
        if seen_states.get(job_id) != state:
            seen_states[job_id] = state
            print(f"job={job_id} status={job.get('status')} message={job.get('message', '')}")

        status_name = job.get("status")
        validate_job_payload(job)
        if status_name == "waiting":
            print("\nFAIL: captured install is waiting at the local install gate even though auto-install is enabled.", file=sys.stderr)
            exit_code = 1
            break
        if status_name in ("queued", "running"):
            time.sleep(2)
            continue
        if status_name == "completed":
            diagnostics = request("GET", f"/api/games/{app_id}/diagnostics")
            print("\nPASS: captured install completed without stopping at the local install gate.")
            print(
                "summary: "
                f"installed={diagnostics.get('installed_mods')} "
                f"enabled={diagnostics.get('enabled_mods')} "
                f"blocked={diagnostics.get('blocked_candidates')} "
                f"active_install={diagnostics.get('active_install_jobs')}"
            )
            exit_code = 0
            break
        if status_name in ("failed", "canceled"):
            print(f"\nFAIL: captured install reached terminal status {status_name}: {job.get('message', '')}", file=sys.stderr)
            exit_code = 1
            break
        time.sleep(2)
    else:
        print(f"\nFAIL: no matching captured install completed within {timeout} seconds.", file=sys.stderr)
        exit_code = 1
finally:
    if restore_setting:
        request("PUT", "/api/settings/install", {
            "auto_install_captured_downloads": previous_auto_install,
            "auto_enable_installed_mods": previous_auto_enable,
        })
        print("restored_auto_install=", previous_auto_install, sep="")

sys.exit(exit_code)
PY
