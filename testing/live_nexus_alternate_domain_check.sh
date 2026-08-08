#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
APP_ID="${APP_ID:-377160}"
ALT_DOMAIN="${ALT_DOMAIN:-fallout4london}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

python3 - <<'PY'
import json
import os
import urllib.error
import urllib.parse
import urllib.request

from dmm_test_auth import auth_headers

base_url = os.environ.get("BASE_URL", "http://127.0.0.1:17942").rstrip("/")
app_id = os.environ.get("APP_ID", "377160")
alt_domain = os.environ.get("ALT_DOMAIN", "fallout4london")

def request(path):
    req = urllib.request.Request(base_url + path, headers=auth_headers(), method="GET")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = resp.read().decode("utf-8")
            return resp.status, json.loads(body) if body else None
    except urllib.error.HTTPError as err:
        body = err.read().decode("utf-8", "replace")
        return err.code, body

status, health = request("/api/health")
if status != 200 or not isinstance(health, dict) or not health.get("ok"):
    raise SystemExit(f"backend health check failed: status={status} body={health!r}")

status, games = request("/api/games")
if status != 200 or not isinstance(games, list):
    raise SystemExit(f"games request failed: status={status} body={games!r}")
game = next((item for item in games if str(item.get("app_id")) == app_id), None)
if not game:
    raise SystemExit(f"app {app_id} was not found in /api/games")
domains = [str(item).lower() for item in game.get("nexus_domains") or []]
if alt_domain.lower() not in domains:
    raise SystemExit(f"{alt_domain} not registered for app {app_id}: {domains}")

query = urllib.parse.urlencode({"domain": alt_domain, "count": 1, "vortex_only": "true"})
status, result = request(f"/api/games/{urllib.parse.quote(app_id)}/nexus/mods?{query}")
if status != 200 or not isinstance(result, dict):
    raise SystemExit(f"alternate domain search failed: status={status} body={result!r}")
mods = result.get("mods") or []
if not mods:
    raise SystemExit(f"alternate domain search returned no mods: {json.dumps(result, indent=2)}")
first = mods[0]
if alt_domain.lower() not in str(first.get("url", "")).lower():
    raise SystemExit(f"first result URL does not use {alt_domain}: {first!r}")

bad_query = urllib.parse.urlencode({"domain": "skyrimspecialedition", "count": 1})
status, _ = request(f"/api/games/{urllib.parse.quote(app_id)}/nexus/mods?{bad_query}")
if status != 404:
    raise SystemExit(f"unregistered domain status={status}, want 404")

print("nexus_alternate_domain_ok")
print(f"app_id={app_id}")
print(f"domain={alt_domain}")
print(f"first_mod={first.get('mod_id')} {first.get('name')}")
PY
