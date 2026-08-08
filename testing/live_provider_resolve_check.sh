#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:17942}"
APP_ID="${APP_ID:-413150}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PYTHONPATH="${SCRIPT_DIR}${PYTHONPATH:+:${PYTHONPATH}}"

python3 - "${BASE_URL}" "${APP_ID}" <<'PY'
import json
import os
import sys
import urllib.error
import urllib.request

from dmm_test_auth import auth_headers

base_url = sys.argv[1].rstrip("/")
app_id = sys.argv[2]


def request(method, path, body=None):
    data = None
    headers = {}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(base_url + path, data=data, headers=auth_headers(headers), method=method)
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            payload = resp.read()
            if not payload:
                return None
            return json.loads(payload.decode("utf-8"))
    except urllib.error.HTTPError as err:
        detail = err.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path} failed: {err.code} {detail}") from err


def section(title):
    print(f"\n==> {title}")


section("Catalog status")
catalogs = request("GET", "/api/catalogs") or []
by_id = {item.get("id"): item for item in catalogs}
for catalog_id in [
    "nexus",
    "thunderstore",
    "github",
    "modrinth",
    "gamebanana",
    "direct",
    "local",
    "modio",
    "curseforge",
    "moddb",
    "itchio",
    "steam_workshop",
]:
    item = by_id.get(catalog_id)
    if not item:
        raise RuntimeError(f"missing catalog status for {catalog_id}")
    print(f"{catalog_id}: {item.get('status')} source={item.get('source_tag')}")

moddb = by_id.get("moddb", {})
if moddb.get("status") != "deferred":
    raise RuntimeError(f"expected ModDB to remain deferred until a supported API/client path is verified, got {moddb}")
itchio = by_id.get("itchio", {})
if itchio.get("status") != "deferred":
    raise RuntimeError(f"expected itch.io to remain deferred until a supported arbitrary mod import path is verified, got {itchio}")

cases = [
    (
        "direct",
        "https://example.com/archive.zip",
        "direct",
    ),
    (
        "github",
        "https://github.com/BepInEx/BepInEx/releases/download/v5.4.23.3/BepInEx_x64_5.4.23.3.zip",
        "github",
    ),
    (
        "thunderstore",
        "https://thunderstore.io/c/lethal-company/p/BepInEx/BepInExPack/",
        "thunderstore",
    ),
    (
        "modrinth",
        "https://modrinth.com/mod/sodium",
        "modrinth",
    ),
    (
        "gamebanana",
        "https://gamebanana.com/mods/1363",
        "gamebanana",
    ),
]

optional_cases = [
    (
        "modio",
        os.environ.get("DMM_TEST_MODIO_URL", ""),
        "modio",
    ),
    (
        "curseforge",
        os.environ.get("DMM_TEST_CURSEFORGE_URL", ""),
        "curseforge",
    ),
]

section("Provider URL resolution")
for label, raw_url, expected in cases:
    resolved = request("POST", "/api/catalogs/resolve", {
        "url": raw_url,
        "steam_app_id": app_id,
        "source": "live-provider-resolve-check",
    })
    catalog = resolved.get("resolved", {}).get("catalog")
    links = resolved.get("resolved", {}).get("download_links") or []
    if catalog != expected:
        raise RuntimeError(f"{label}: catalog={catalog!r}, expected {expected!r}; response={resolved}")
    if not links:
        raise RuntimeError(f"{label}: no download links returned; response={resolved}")
    print(f"{label}: ok catalog={catalog} file={resolved.get('resolved', {}).get('file_name')}")

section("Credential-gated providers")
for label, raw_url, expected in optional_cases:
    status = by_id.get(expected, {}).get("status")
    if not raw_url:
        print(f"{label}: skipped; set DMM_TEST_{expected.upper().replace('.', '')}_URL after configuring credentials")
        continue
    if status != "ready":
        raise RuntimeError(f"{label}: test URL was provided but catalog status is {status!r}")
    resolved = request("POST", "/api/catalogs/resolve", {
        "url": raw_url,
        "steam_app_id": app_id,
        "source": "live-provider-resolve-check",
    })
    catalog = resolved.get("resolved", {}).get("catalog")
    if catalog != expected:
        raise RuntimeError(f"{label}: catalog={catalog!r}, expected {expected!r}; response={resolved}")
    print(f"{label}: ok catalog={catalog}")

print("\nprovider resolve check passed")
PY
