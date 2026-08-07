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
    "starwarsbattlefront22017": "1237950",
    "civilizationvii": "1295660",
    "hollowknight": "367520",
    "kenshi": "233860",
    "rimworld": "294100",
    "x4foundations": "392160",
    "halothemasterchiefcollection": "976730",
    "spyroreignitedtrilogy": "996580",
    "spidermanmilesmorales": "1817190",
    "portal2": "620",
    "projectzomboid": "108600",
    "starwarsjedisurvivor": "1774580",
    "prototype": "10150",
    "prototype2": "115320",
    "metalgearsolidvtpp": "287700",
    "finalfantasyxx2hdremaster": "359870",
    "totalwarrome2": "214950",
}
for extension_id, app_id in required_extensions.items():
    summary = extensions_by_id.get(extension_id)
    if not summary:
        failures.append(f"missing extension summary {extension_id}")
        continue
    if app_id not in [str(item) for item in summary.get("steam_app_ids", [])]:
        failures.append(f"{extension_id} missing app {app_id}: {summary.get('steam_app_ids')}")

def require_domain(extension_id, domain):
    summary = extensions_by_id.get(extension_id) or {}
    if domain not in [str(item) for item in summary.get("nexus_domains", [])]:
        failures.append(f"{extension_id} missing Nexus domain {domain}: {summary.get('nexus_domains')}")

def require_caps(extension_id, *caps):
    summary = extensions_by_id.get(extension_id) or {}
    capabilities = summary.get("capabilities") or {}
    for cap in caps:
        if not capabilities.get(cap):
            failures.append(f"{extension_id} missing capability {cap}: {capabilities}")
    return capabilities

def require_coverage(extension_id, coverage):
    summary = extensions_by_id.get(extension_id) or {}
    if summary.get("coverage") != coverage:
        failures.append(f"{extension_id} coverage={summary.get('coverage')} want {coverage}")

installer_targets = {
    "stardewvalley": ("stardewvalley", ("installers", "runtime_requirements", "launch_tools")),
    "fallout4": ("fallout4", ("installers", "installer_choices", "runtime_requirements", "launch_tools", "plugin_activations")),
    "skyrimse": ("skyrimspecialedition", ("installers", "installer_choices", "runtime_requirements", "launch_tools", "plugin_activations")),
    "witcher3": ("witcher3", ("installers", "event_handlers")),
    "finalfantasy7rebirth": ("finalfantasy7rebirth", ("installers", "target_roots", "event_handlers")),
    "starwarsbattlefront22017": ("starwarsbattlefront22017", ("installers", "runtime_requirements", "launch_tools")),
    "civilizationvii": ("civilizationvii", ("installers", "target_roots", "game_versions")),
    "hollowknight": ("hollowknight", ("installers", "runtime_requirements", "game_versions")),
    "discoelysium": ("discoelysium", ("installers", "runtime_requirements", "game_versions")),
    "citizensleeper": ("citizensleeper", ("installers", "runtime_requirements", "game_versions")),
    "kenshi": ("kenshi", ("installers", "game_versions", "steam_workshop")),
    "rimworld": ("rimworld", ("installers", "game_versions", "steam_workshop")),
    "x4foundations": ("x4foundations", ("installers", "target_roots", "game_versions", "steam_workshop")),
    "halothemasterchiefcollection": ("halothemasterchiefcollection", ("installers", "launch_tools", "game_versions", "event_handlers")),
    "spyroreignitedtrilogy": ("spyroreignitedtrilogy", ("installers", "merges", "load_orders", "event_handlers")),
    "spidermanmilesmorales": ("spidermanmilesmorales", ("installers", "runtime_requirements", "launch_tools", "merges", "load_orders", "event_handlers")),
    "portal2": ("portal2", ("installers",)),
    "thebindingofisaacrebirth": ("thebindingofisaacrebirth", ("installers", "game_versions")),
    "mewgenics": ("mewgenics", ("installers", "launch_tools", "load_orders", "event_handlers", "game_versions")),
    "megabonk": ("megabonk", ("installers", "runtime_requirements", "game_versions")),
    "projectzomboid": ("projectzomboid", ("installers", "target_roots", "steam_workshop")),
    "starwarsjedisurvivor": ("starwarsjedisurvivor", ("installers", "load_orders", "event_handlers")),
}

for extension_id, (domain, caps) in installer_targets.items():
    require_coverage(extension_id, "installer")
    require_domain(extension_id, domain)
    require_caps(extension_id, *caps)

research_targets = {
    "prototype": "prototype",
    "prototype2": "prototype2",
    "metalgearsolidvtpp": "metalgearsolidvtpp",
    "finalfantasyxx2hdremaster": "finalfantasyxx2hdremaster",
    "totalwarrome2": "totalwarrome2",
}
for extension_id, domain in research_targets.items():
    require_coverage(extension_id, "research_blocked")
    require_domain(extension_id, domain)
    require_caps(extension_id, "installers")

zomboid = extensions_by_id.get("projectzomboid") or {}
zomboid_caps = zomboid.get("capabilities") or {}
if not zomboid_caps.get("target_roots") or not zomboid_caps.get("installers"):
    failures.append(f"Project Zomboid archive capabilities missing: {zomboid_caps}")

jedi = extensions_by_id.get("starwarsjedisurvivor") or {}
if "starwarsjedisurvivor" not in [str(item) for item in jedi.get("nexus_domains", [])]:
    failures.append(f"Jedi Survivor missing Nexus domain: {jedi.get('nexus_domains')}")
jedi_caps = jedi.get("capabilities") or {}
if not jedi_caps.get("installers") or not jedi_caps.get("load_orders") or not jedi_caps.get("event_handlers"):
    failures.append(f"Jedi Survivor capabilities missing: {jedi_caps}")

rome2 = extensions_by_id.get("totalwarrome2") or {}
if "totalwarrome2" not in [str(item) for item in rome2.get("nexus_domains", [])]:
    failures.append(f"Total War ROME II missing Nexus domain: {rome2.get('nexus_domains')}")
rome2_caps = rome2.get("capabilities") or {}
if not rome2_caps.get("installers") or rome2_caps.get("steam_workshop"):
    failures.append(f"Total War ROME II capability mismatch: {rome2_caps}")

if failures:
    print("extension target check failed:")
    for failure in failures:
        print(" - " + failure)
    raise SystemExit(1)

print("extension target check passed")
print(f"games_checked={len(games_by_app)} extensions_checked={len(extensions_by_id)}")
PY
