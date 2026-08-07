#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"

python3 - "$BASE_URL" <<'PY'
import json
import sys
import urllib.request

base_url = sys.argv[1].rstrip("/")

def request(path):
    with urllib.request.urlopen(base_url + path, timeout=20) as response:
        return json.load(response)

games = request("/api/games")
extensions = request("/api/extensions")
games_by_app = {str(game.get("app_id", "")): game for game in games}
extensions_by_id = {str(extension.get("id", "")): extension for extension in extensions}

failures = []

workshop_only = {
    "2168680": "nuclearoption",
    "2951630": "totalwarpharaohdynasties",
    "885970": "totalwarromeremastered",
    "1611600": "warno",
}
for app_id, extension_id in workshop_only.items():
    game = games_by_app.get(app_id)
    if not game:
        failures.append(f"{app_id} missing from live game list")
        continue
    extension = game.get("extension") or {}
    workshop = game.get("steam_workshop") or {}
    if extension.get("id") != extension_id or not extension.get("supported"):
        failures.append(f"{app_id} extension mismatch: {extension}")
    if game.get("state") != "clean_candidate":
        failures.append(f"{app_id} state={game.get('state')} markers={game.get('markers')}")
    if not workshop.get("coexistence_allowed") or not workshop.get("management_supported"):
        failures.append(f"{app_id} workshop policy={workshop}")
    summary = extensions_by_id.get(extension_id) or {}
    if summary.get("nexus_domains"):
        failures.append(f"{extension_id} should not declare Nexus domains: {summary.get('nexus_domains')}")

fallout = games_by_app.get("377160")
if not fallout:
    failures.append("Fallout 4 missing from live game list")
else:
    if fallout.get("state") != "clean_candidate" or fallout.get("markers"):
        failures.append(f"Fallout 4 should ignore extension-known F4SE marker: state={fallout.get('state')} markers={fallout.get('markers')}")
    extension = fallout.get("extension") or {}
    if not extension.get("plugin_activation") or not extension.get("launch_tools") or not extension.get("installer_choices"):
        failures.append(f"Fallout 4 extension capability mismatch: {extension}")

required_extensions = {
    "stardewvalley": "413150",
    "fallout4": "377160",
    "skyrimse": "489830",
    "witcher3": "292030",
    "finalfantasy7rebirth": "2909400",
}
for extension_id, app_id in required_extensions.items():
    summary = extensions_by_id.get(extension_id)
    if not summary:
        failures.append(f"missing extension summary {extension_id}")
        continue
    if app_id not in [str(item) for item in summary.get("steam_app_ids", [])]:
        failures.append(f"{extension_id} missing app {app_id}: {summary.get('steam_app_ids')}")

if failures:
    print("extension target check failed:")
    for failure in failures:
        print(" - " + failure)
    raise SystemExit(1)

print("extension target check passed")
print(f"games_checked={len(games_by_app)} extensions_checked={len(extensions_by_id)}")
PY
