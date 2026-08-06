#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ID="${APP_ID:-413150}"
PORT="${PORT:-17957}"
PACKAGE="${PACKAGE:-${HOME}/.testing/decky-mod-manager.tar.gz}"
DATA_SOURCE="${DATA_SOURCE:-${HOME}/.local/share/decky-mod-manager}"
GAME_SOURCE="${GAME_SOURCE:-${HOME}/.local/share/Steam/steamapps/common/Stardew Valley}"
TMP_ROOT="${TMP_ROOT:-/tmp/dmm-deploy-rehearsal}"
SERVER_PID=""

DATA_COPY="${TMP_ROOT}/data"
CONFIG_HOME="${TMP_ROOT}/config"
PACKAGE_DIR="${TMP_ROOT}/package"
GAME_COPY="${TMP_ROOT}/game"
LOG_FILE="${TMP_ROOT}/server.log"
BASE_URL="http://127.0.0.1:${PORT}"

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  if [[ -z "${KEEP_TMP:-}" ]]; then
    rm -rf "${TMP_ROOT}"
  else
    echo "Kept rehearsal directory: ${TMP_ROOT}"
  fi
}
trap cleanup EXIT

section() {
  printf '\n==> %s\n' "$1"
}

require_file() {
  local path="$1"
  if [[ ! -e "$path" ]]; then
    echo "missing required path: $path" >&2
    exit 1
  fi
}

json_number() {
  local file="$1"
  local key="$2"
  python3 - "$file" "$key" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
value = data
for part in sys.argv[2].split("."):
    value = value[part]
print(value)
PY
}

assert_expected_staged_mods() {
  if [[ -z "${EXPECT_STAGED_MODS:-}" ]]; then
    return 0
  fi
  section "Expected staged mod metadata"
  python3 - "${DATA_COPY}/db/dmm.sqlite" "${EXPECT_STAGED_MODS}" <<'PY'
import json
import sqlite3
import sys

db_path, expected_raw = sys.argv[1], sys.argv[2]
expected = []
for item in expected_raw.split(","):
    item = item.strip()
    if not item:
        continue
    if "=" in item:
        mod_id, planner = item.split("=", 1)
        expected.append((mod_id.strip(), planner.strip()))
    else:
        expected.append((item, ""))

conn = sqlite3.connect(db_path)
rows = conn.execute(
    """
    select m.source_mod_id, mv.source_file_id, m.name, im.checksum_manifest_json
    from installed_mods im
    join mod_versions mv on mv.id = im.mod_version_id
    join mods m on m.id = mv.mod_id
    """
).fetchall()

by_mod = {}
for mod_id, file_id, name, manifest_json in rows:
    try:
        manifest = json.loads(manifest_json)
    except Exception as err:
        manifest = {"_parse_error": str(err)}
    by_mod.setdefault(str(mod_id), []).append((str(file_id), name, manifest))

failures = []
for mod_id, expected_planner in expected:
    candidates = by_mod.get(str(mod_id), [])
    if not candidates:
        failures.append(f"expected source_mod_id {mod_id} to be staged")
        continue
    if expected_planner:
        matches = [
            (file_id, name, manifest)
            for file_id, name, manifest in candidates
            if manifest.get("planner_id") == expected_planner
        ]
        if not matches:
            planners = [candidate[2].get("planner_id") for candidate in candidates]
            failures.append(
                f"expected source_mod_id {mod_id} planner {expected_planner}, found {planners}"
            )
            continue
        candidates = matches
    for file_id, name, manifest in candidates:
        print(
            "  mod={mod} file={file} name={name} planner={planner} type={mod_type} files={files}".format(
                mod=mod_id,
                file=file_id,
                name=name,
                planner=manifest.get("planner_id", ""),
                mod_type=manifest.get("mod_type", ""),
                files=len(manifest.get("files") or []),
            )
        )

if failures:
    for failure in failures:
        print(failure, file=sys.stderr)
    sys.exit(1)
PY
}

assert_expected_staged_metadata() {
  if [[ -z "${EXPECT_STAGED_METADATA:-}" ]]; then
    return 0
  fi
  section "Expected staged manifest metadata"
  python3 - "${DATA_COPY}/db/dmm.sqlite" "${EXPECT_STAGED_METADATA}" <<'PY'
import json
import sqlite3
import sys

db_path, expected_raw = sys.argv[1], sys.argv[2]
expected = []
for item in expected_raw.split(","):
    item = item.strip()
    if not item:
        continue
    if "=" not in item:
        print(f"expected metadata item must use mod_id=unique_id[;unique_id]: {item}", file=sys.stderr)
        sys.exit(1)
    mod_id, unique_ids = item.split("=", 1)
    expected.append((mod_id.strip(), [value.strip() for value in unique_ids.split(";") if value.strip()]))

conn = sqlite3.connect(db_path)
rows = conn.execute(
    """
    select m.source_mod_id, mv.source_file_id, m.name, im.checksum_manifest_json
    from installed_mods im
    join mod_versions mv on mv.id = im.mod_version_id
    join mods m on m.id = mv.mod_id
    """
).fetchall()

by_mod = {}
for mod_id, file_id, name, manifest_json in rows:
    try:
        manifest = json.loads(manifest_json)
    except Exception as err:
        manifest = {"_parse_error": str(err)}
    by_mod.setdefault(str(mod_id), []).append((str(file_id), name, manifest))

failures = []
for mod_id, unique_ids in expected:
    candidates = by_mod.get(str(mod_id), [])
    if not candidates:
        failures.append(f"expected source_mod_id {mod_id} to be staged")
        continue
    seen = set()
    for _, _, manifest in candidates:
        for metadata in manifest.get("metadata") or []:
            unique_id = str(metadata.get("unique_id") or "").strip()
            if unique_id:
                seen.add(unique_id)
                logical_names = [str(value).strip() for value in metadata.get("additional_logical_file_names") or []]
                if unique_id.lower() not in logical_names:
                    failures.append(
                        f"expected source_mod_id {mod_id} metadata unique_id {unique_id} to include logical file name {unique_id.lower()}"
                    )
            version = str(metadata.get("version") or "").strip()
            manifest_version = str(metadata.get("manifest_version") or "").strip()
            if version and manifest_version != version:
                failures.append(
                    f"expected source_mod_id {mod_id} metadata version {version} to mirror manifest_version, got {manifest_version or '<none>'}"
                )
    for unique_id in unique_ids:
        if unique_id not in seen:
            failures.append(
                f"expected source_mod_id {mod_id} metadata unique_id {unique_id}, found {sorted(seen)}"
            )
    print(f"  mod={mod_id} metadata={','.join(sorted(seen)) or '<none>'}")

if failures:
    for failure in failures:
        print(failure, file=sys.stderr)
    sys.exit(1)
PY
}

assert_expected_api_metadata() {
  if [[ -z "${EXPECT_STAGED_METADATA:-}" ]]; then
    return 0
  fi
  section "Expected public mods API metadata"
  curl -fsS "${BASE_URL}/api/games/${APP_ID}/mods" >"${TMP_ROOT}/mods.json"
  python3 - "${TMP_ROOT}/mods.json" "${EXPECT_STAGED_METADATA}" <<'PY'
import json
import sys

mods_path, expected_raw = sys.argv[1], sys.argv[2]
body = open(mods_path, encoding="utf-8").read()
if "manifest_json" in body or "staging_path" in body:
    print("public mods API leaked raw manifest or staging path internals", file=sys.stderr)
    sys.exit(1)

mods = json.loads(body)
expected = []
for item in expected_raw.split(","):
    item = item.strip()
    if not item:
        continue
    if "=" not in item:
        print(f"expected metadata item must use mod_id=unique_id[;unique_id]: {item}", file=sys.stderr)
        sys.exit(1)
    mod_id, unique_ids = item.split("=", 1)
    expected.append((mod_id.strip(), [value.strip() for value in unique_ids.split(";") if value.strip()]))

by_mod = {}
for mod in mods:
    by_mod.setdefault(str(mod.get("source_mod_id", "")), []).append(mod)

failures = []
for mod_id, unique_ids in expected:
    candidates = by_mod.get(str(mod_id), [])
    if not candidates:
        failures.append(f"expected source_mod_id {mod_id} in public mods API")
        continue
    seen = set()
    planners = set()
    mod_types = set()
    for mod in candidates:
        if mod.get("planner_id"):
            planners.add(str(mod.get("planner_id")))
        if mod.get("mod_type"):
            mod_types.add(str(mod.get("mod_type")))
        for metadata in mod.get("metadata") or []:
            unique_id = str(metadata.get("unique_id") or "").strip()
            if unique_id:
                seen.add(unique_id)
                logical_names = [str(value).strip() for value in metadata.get("additional_logical_file_names") or []]
                if unique_id.lower() not in logical_names:
                    failures.append(
                        f"expected public mods API source_mod_id {mod_id} metadata unique_id {unique_id} to include logical file name {unique_id.lower()}"
                    )
            version = str(metadata.get("version") or "").strip()
            manifest_version = str(metadata.get("manifest_version") or "").strip()
            if version and manifest_version != version:
                failures.append(
                    f"expected public mods API source_mod_id {mod_id} metadata version {version} to mirror manifest_version, got {manifest_version or '<none>'}"
                )
    for unique_id in unique_ids:
        if unique_id not in seen:
            failures.append(
                f"expected public mods API source_mod_id {mod_id} metadata unique_id {unique_id}, found {sorted(seen)}"
            )
    print(
        "  mod={mod} metadata={metadata} planners={planners} types={types}".format(
            mod=mod_id,
            metadata=",".join(sorted(seen)) or "<none>",
            planners=",".join(sorted(planners)) or "<none>",
            types=",".join(sorted(mod_types)) or "<none>",
        )
    )

if failures:
    for failure in failures:
        print(failure, file=sys.stderr)
    sys.exit(1)
PY
}

summarize_json() {
  local label="$1"
  local file="$2"
  python3 - "$label" "$file" <<'PY'
import json
import sys

label, path = sys.argv[1], sys.argv[2]
data = json.load(open(path, encoding="utf-8"))
print(label)

if "diagnostics" in label.lower():
    deployment = data.get("deployment", {})
    preview = data.get("preview", {})
    print(
        "  installed={installed} enabled={enabled} needs_recovery={recovery} blocked={blocked} deployed={deployed} deployed_files={files}".format(
            installed=data.get("installed_mods", 0),
            enabled=data.get("enabled_mods", 0),
            recovery=data.get("needs_recovery", 0),
            blocked=data.get("blocked_candidates", 0),
            deployed=deployment.get("deployed", False),
            files=deployment.get("file_count", 0),
        )
    )
    print(
        "  preview_available={available} add={add} replace={replace} remove={remove} keep={keep} conflicts={conflicts}".format(
            available=preview.get("available", False),
            add=preview.get("add", 0),
            replace=preview.get("replace", 0),
            remove=preview.get("remove", 0),
            keep=preview.get("keep", 0),
            conflicts=preview.get("conflicts", 0),
        )
    )
    for warning in data.get("validation_warnings", []):
        print(f"  warning={warning}")
elif "preview" in label.lower():
    actions = data.get("actions", [])
    counts = {}
    for action in actions:
        operation = action.get("operation", "unknown")
        counts[operation] = counts.get(operation, 0) + 1
    print(
        "  actions={actions} add={add} replace={replace} remove={remove} keep={keep} skip={skip} conflicts={conflicts}".format(
            actions=len(actions),
            add=counts.get("add", 0),
            replace=counts.get("replace", 0),
            remove=counts.get("remove", 0),
            keep=counts.get("keep", 0),
            skip=counts.get("skip", 0),
            conflicts=len(data.get("conflicts", [])),
        )
    )
elif "deploy" in label.lower():
    job = data.get("job", {})
    print(f"  job={job.get('id')} status={job.get('status')} message={job.get('message')}")
    print(f"  applied={len(data.get('applied', []))}")
elif "recover" in label.lower():
    job = data.get("job", {})
    print(f"  job={job.get('id')} status={job.get('status')} message={job.get('message')}")
    print(f"  staged={data.get('staged', 0)} skipped={data.get('skipped', 0)}")
elif "purge" in label.lower():
    job = data.get("job", {})
    print(f"  job={job.get('id')} status={job.get('status')} message={job.get('message')}")
else:
    print(json.dumps(data, indent=2))
PY
}

wait_for_server() {
  for _ in $(seq 1 60); do
    if curl -fsS "${BASE_URL}/api/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "server did not become healthy" >&2
  tail -n 80 "${LOG_FILE}" >&2 || true
  exit 1
}

section "Preparing copied-data rehearsal"
require_file "${PACKAGE}"
require_file "${DATA_SOURCE}/db/dmm.sqlite"
require_file "${GAME_SOURCE}"
rm -rf "${TMP_ROOT}"
mkdir -p "${DATA_COPY}" "${CONFIG_HOME}/decky-mod-manager" "${PACKAGE_DIR}" "${GAME_COPY}"
cp -a "${DATA_SOURCE}/." "${DATA_COPY}/"
cp -a "${GAME_SOURCE}/." "${GAME_COPY}/"

sqlite3 "${DATA_COPY}/db/dmm.sqlite" \
  "update games set game_path='${GAME_COPY}', state='clean_candidate' where steam_app_id='${APP_ID}';"
sqlite3 "${DATA_COPY}/db/dmm.sqlite" \
  "update installed_mods set staging_path=replace(staging_path, '${DATA_SOURCE}', '${DATA_COPY}') where staging_path like '${DATA_SOURCE}/%';
   update deployed_files set source_path=replace(source_path, '${DATA_SOURCE}', '${DATA_COPY}') where source_path like '${DATA_SOURCE}/%';
   update deployed_files set target_path=replace(target_path, '${GAME_SOURCE}', '${GAME_COPY}') where target_path like '${GAME_SOURCE}/%';"

tar -xzf "${PACKAGE}" -C "${PACKAGE_DIR}"
cat > "${CONFIG_HOME}/decky-mod-manager/config.json" <<JSON
{
  "listen_addr": "127.0.0.1:${PORT}",
  "lan_only": false,
  "data_dir": "${DATA_COPY}",
  "install": {
    "auto_install_captured_downloads": true,
    "auto_enable_installed_mods": false
  }
}
JSON

section "Starting packaged backend"
XDG_CONFIG_HOME="${CONFIG_HOME}" XDG_DATA_HOME="${TMP_ROOT}/xdg-data" \
  "${PACKAGE_DIR}/decky-mod-manager/bin/dmm-server" >"${LOG_FILE}" 2>&1 &
SERVER_PID="$!"
wait_for_server
curl -fsS "${BASE_URL}/api/health"
printf '\n'

section "Initial diagnostics"
curl -fsS "${BASE_URL}/api/games/${APP_ID}/diagnostics" >"${TMP_ROOT}/diagnostics-before.json"
summarize_json "Initial diagnostics" "${TMP_ROOT}/diagnostics-before.json"

section "Recover downloaded archives"
curl -fsS -X POST "${BASE_URL}/api/games/${APP_ID}/mods/recover-downloads" >"${TMP_ROOT}/recover.json"
summarize_json "Recover result" "${TMP_ROOT}/recover.json"
assert_expected_staged_mods
assert_expected_staged_metadata
assert_expected_api_metadata

section "Preview deployment"
curl -fsS "${BASE_URL}/api/games/${APP_ID}/deploy/preview" >"${TMP_ROOT}/preview.json"
summarize_json "Preview result" "${TMP_ROOT}/preview.json"
conflicts="$(python3 - <<PY
import json
data=json.load(open("${TMP_ROOT}/preview.json", encoding="utf-8"))
print(len(data.get("conflicts", [])))
PY
)"
if [[ "${conflicts}" != "0" ]]; then
  echo "deployment preview has conflicts: ${conflicts}" >&2
  exit 1
fi

section "Deploy copied game"
curl -fsS -X POST "${BASE_URL}/api/games/${APP_ID}/deploy" >"${TMP_ROOT}/deploy.json"
summarize_json "Deploy result" "${TMP_ROOT}/deploy.json"
deployed_links="$(find "${GAME_COPY}/Mods" -type l 2>/dev/null | wc -l | tr -d ' ')"
if [[ "${deployed_links}" -le 0 ]]; then
  echo "expected deployed symlinks in copied game, found ${deployed_links}" >&2
  exit 1
fi
echo "deployed_symlinks=${deployed_links}"

if [[ "${RUN_FILE_VISIBILITY_CHECK:-0}" != "0" ]]; then
  section "Copied-game file visibility"
  env \
    BASE_URL="${BASE_URL}" \
    APP_ID="${APP_ID}" \
    DATA_DIR="${DATA_COPY}" \
    GAME_PATH="${GAME_COPY}" \
    REQUIRE_RUNTIME="${REQUIRE_RUNTIME:-0}" \
    REQUIRE_SMAPI_ROOT="${REQUIRE_SMAPI_ROOT:-0}" \
    "${SCRIPT_DIR}/live_stardew_mod_files_check.sh"
fi

if [[ "${RUN_PROFILE_TOGGLE_CHECK:-0}" != "0" ]]; then
  section "Copied-game profile toggle"
  env \
    BASE_URL="${BASE_URL}" \
    APP_ID="${APP_ID}" \
    DATA_DIR="${DATA_COPY}" \
    "${SCRIPT_DIR}/live_profile_toggle_check.sh"
fi

section "Post-deploy diagnostics"
curl -fsS "${BASE_URL}/api/games/${APP_ID}/diagnostics" >"${TMP_ROOT}/diagnostics-after-deploy.json"
summarize_json "Post-deploy diagnostics" "${TMP_ROOT}/diagnostics-after-deploy.json"
deployed_count="$(json_number "${TMP_ROOT}/diagnostics-after-deploy.json" "deployment.file_count")"
if [[ "${deployed_count}" -le 0 ]]; then
  echo "expected active deployment file count, found ${deployed_count}" >&2
  exit 1
fi

section "Purge copied game"
curl -fsS -X DELETE "${BASE_URL}/api/games/${APP_ID}/deploy" >"${TMP_ROOT}/purge.json"
summarize_json "Purge result" "${TMP_ROOT}/purge.json"
remaining_links="$(find "${GAME_COPY}/Mods" -type l 2>/dev/null | wc -l | tr -d ' ')"
if [[ "${remaining_links}" != "0" ]]; then
  echo "expected purge to remove copied-game symlinks, found ${remaining_links}" >&2
  exit 1
fi
echo "remaining_symlinks=${remaining_links}"

section "Rehearsal passed"
tail -n 40 "${LOG_FILE}"
