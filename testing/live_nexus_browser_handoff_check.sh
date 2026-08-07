#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
APP_ID="${APP_ID:-413150}"
NEXUS_URL="${NEXUS_URL:-https://www.nexusmods.com/stardewvalley/mods/2400?file_id=135998}"

python3 - <<'PY'
import json
import os
import sys
import urllib.error
import urllib.request

base_url = os.environ.get("BASE_URL", "http://127.0.0.1:17942").rstrip("/")
app_id = os.environ.get("APP_ID", "413150")
nexus_url = os.environ.get("NEXUS_URL", "https://www.nexusmods.com/stardewvalley/mods/2400?file_id=135998")

def request(method, path, payload=None):
    data = None
    headers = {}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base_url + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = resp.read().decode("utf-8")
            return json.loads(body) if body else None
    except urllib.error.HTTPError as err:
        body = err.read().decode("utf-8", "replace")
        raise SystemExit(f"{method} {path} failed: {err.code} {body}") from err

health = request("GET", "/api/health")
if not health or not health.get("ok"):
    raise SystemExit(f"backend health check failed: {health}")

result = request("POST", "/api/captured-installs", {
    "url": nexus_url,
    "steam_app_id": app_id,
    "source": "live-browser-handoff-check",
})
job = result.get("job") or {}
if not result.get("browser_required"):
    raise SystemExit(f"browser_required was false: {json.dumps(result, indent=2)}")
if job.get("status") != "completed":
    raise SystemExit(f"handoff job status = {job.get('status')!r}, want completed: {json.dumps(result, indent=2)}")
if "Mod Manager Download" not in job.get("message", ""):
    raise SystemExit(f"handoff message did not guide to Mod Manager Download: {job.get('message')!r}")

jobs = request("GET", "/api/jobs") or []
active_matching = [
    item for item in jobs
    if item.get("id") == job.get("id") and item.get("status") not in ("completed", "canceled")
]
if active_matching:
    raise SystemExit(f"handoff job is still active: {json.dumps(active_matching, indent=2)}")

print("nexus_browser_handoff_ok")
print(f"job_id={job.get('id')}")
print(f"message={job.get('message')}")
PY
