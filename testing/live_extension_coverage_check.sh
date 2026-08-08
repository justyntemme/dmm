#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
REQUIRE_NO_UNSUPPORTED="${REQUIRE_NO_UNSUPPORTED:-0}"
REQUIRE_NO_RESEARCH_BLOCKED="${REQUIRE_NO_RESEARCH_BLOCKED:-0}"
REQUIRE_NO_BROWSE_ONLY="${REQUIRE_NO_BROWSE_ONLY:-0}"
EXPECTED_COVERAGE="${EXPECTED_COVERAGE:-413150=installer,774361=installer,1868140=installer,761830=installer,1210320=installer,2210=installer,4760=installer,4770=installer,70=installer,107100=installer,17410=installer,239350=installer,412830=installer,281990=workshop_only}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

section() {
  printf '\n==> %s\n' "$1"
}

section "DMM live extension coverage check"
echo "base_url=${BASE_URL}"
echo "require_no_unsupported=${REQUIRE_NO_UNSUPPORTED}"
echo "require_no_research_blocked=${REQUIRE_NO_RESEARCH_BLOCKED}"
echo "require_no_browse_only=${REQUIRE_NO_BROWSE_ONLY}"
echo "expected_coverage=${EXPECTED_COVERAGE}"
export BASE_URL REQUIRE_NO_UNSUPPORTED REQUIRE_NO_RESEARCH_BLOCKED REQUIRE_NO_BROWSE_ONLY EXPECTED_COVERAGE

python3 - <<'PY'
import json
import os
import sys
import urllib.error
import urllib.request

from dmm_test_auth import auth_headers

base_url = os.environ["BASE_URL"].rstrip("/")
require_no_unsupported = os.environ.get("REQUIRE_NO_UNSUPPORTED", "0").strip().lower() in ("1", "true", "yes", "on")
require_no_research_blocked = os.environ.get("REQUIRE_NO_RESEARCH_BLOCKED", "0").strip().lower() in ("1", "true", "yes", "on")
require_no_browse_only = os.environ.get("REQUIRE_NO_BROWSE_ONLY", "0").strip().lower() in ("1", "true", "yes", "on")
expected_raw = os.environ.get("EXPECTED_COVERAGE", "")
valid_coverages = {"installer", "research_blocked", "browse_only", "workshop_only", "metadata_only"}


def request(path):
    req = urllib.request.Request(base_url + path, headers=auth_headers(), method="GET")
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            body = resp.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "replace")
        raise RuntimeError(f"GET {path} failed: HTTP {exc.code}: {detail}") from exc
    return json.loads(body.decode("utf-8")) if body else None


health = request("/api/health")
if not health or not health.get("ok"):
    raise RuntimeError(f"backend health check failed: {health}")

extensions = request("/api/extensions") or []
extension_by_id = {item.get("id"): item for item in extensions}
missing_summary_coverage = [item.get("id") for item in extensions if item.get("coverage") not in valid_coverages]
if missing_summary_coverage:
    raise RuntimeError(f"extensions missing valid coverage: {missing_summary_coverage}")

games = request("/api/games") or []
unsupported = []
research_blocked = []
browse_only = []
coverage_counts = {}
problems = []
for game in games:
    app_id = str(game.get("app_id", "")).strip()
    name = game.get("name") or app_id
    extension = game.get("extension")
    if not extension:
        unsupported.append((app_id, name))
        continue
    coverage = extension.get("coverage")
    coverage_counts[coverage] = coverage_counts.get(coverage, 0) + 1
    if coverage not in valid_coverages:
        problems.append(f"{app_id} {name}: invalid coverage {coverage!r}")
        continue
    if coverage == "research_blocked":
        research_blocked.append((app_id, name, extension.get("id")))
    if coverage == "browse_only":
        browse_only.append((app_id, name, extension.get("id")))
    if coverage == "installer" and not (extension.get("installers") or extension.get("installer_choices")):
        problems.append(f"{app_id} {name}: installer coverage without installer capability")
    if coverage == "research_blocked" and not extension.get("installers"):
        problems.append(f"{app_id} {name}: research-blocked coverage without blocked installer capability")
    if coverage == "workshop_only" and not extension.get("steam_workshop"):
        problems.append(f"{app_id} {name}: workshop-only coverage without Workshop capability")
    if extension.get("id") not in extension_by_id:
        problems.append(f"{app_id} {name}: game extension {extension.get('id')!r} missing from /api/extensions")

expected = {}
for item in expected_raw.split(","):
    item = item.strip()
    if not item:
        continue
    if "=" not in item:
        raise RuntimeError(f"EXPECTED_COVERAGE entry must use app=coverage: {item!r}")
    app_id, coverage = item.split("=", 1)
    expected[app_id.strip()] = coverage.strip()

games_by_app = {str(game.get("app_id", "")).strip(): game for game in games}
for app_id, want in expected.items():
    game = games_by_app.get(app_id)
    if not game:
        print(f"expected {app_id}={want}: skipped; app is not installed/visible")
        continue
    got = (game.get("extension") or {}).get("coverage")
    if got != want:
        problems.append(f"{app_id} {game.get('name')}: coverage {got!r}, want {want!r}")

print("coverage_counts:")
for coverage in sorted(coverage_counts):
    print(f"  {coverage}={coverage_counts[coverage]}")
print(f"unsupported={len(unsupported)}")
for app_id, name in unsupported[:25]:
    print(f"  unsupported {app_id} {name}")
if len(unsupported) > 25:
    print(f"  ... {len(unsupported) - 25} more unsupported games")
print(f"research_blocked={len(research_blocked)}")
for app_id, name, extension_id in research_blocked[:25]:
    print(f"  research_blocked {app_id} {name} ({extension_id})")
if len(research_blocked) > 25:
    print(f"  ... {len(research_blocked) - 25} more research-blocked games")
print(f"browse_only={len(browse_only)}")
for app_id, name, extension_id in browse_only[:25]:
    print(f"  browse_only {app_id} {name} ({extension_id})")
if len(browse_only) > 25:
    print(f"  ... {len(browse_only) - 25} more browse-only games")

if require_no_unsupported and unsupported:
    problems.append(f"{len(unsupported)} visible games have no DMM extension")
if require_no_research_blocked and research_blocked:
    problems.append(f"{len(research_blocked)} visible games still have research-blocked extension coverage")
if require_no_browse_only and browse_only:
    problems.append(f"{len(browse_only)} visible games still have browse-only extension coverage")
if problems:
    raise RuntimeError("; ".join(problems))

print("extension coverage check passed")
PY
