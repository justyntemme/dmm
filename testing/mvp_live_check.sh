#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-17942}"
HOST="${HOST:-127.0.0.1}"
BASE_URL="${BASE_URL:-http://${HOST}:${PORT}}"
APP_ID="${APP_ID:-413150}"
REQUIRE_DEPLOYED="${REQUIRE_DEPLOYED:-1}"
ALLOW_WARNINGS="${ALLOW_WARNINGS:-1}"
REQUIRE_JOB_PAYLOAD="${REQUIRE_JOB_PAYLOAD:-0}"

section() {
  printf '\n==> %s\n' "$1"
}

fetch() {
  local url="$1"
  curl -fsS "$url"
}

section "MVP live acceptance check"
echo "base_url=${BASE_URL}"
echo "app_id=${APP_ID}"
echo "require_deployed=${REQUIRE_DEPLOYED}"
echo "require_job_payload=${REQUIRE_JOB_PAYLOAD}"

health_json="$(fetch "${BASE_URL}/api/health")"
diagnostics_json="$(fetch "${BASE_URL}/api/games/${APP_ID}/diagnostics")"
jobs_json="$(fetch "${BASE_URL}/api/jobs")"

export HEALTH_JSON="${health_json}"
export DIAGNOSTICS_JSON="${diagnostics_json}"
export JOBS_JSON="${jobs_json}"
export REQUIRE_DEPLOYED
export ALLOW_WARNINGS
export REQUIRE_JOB_PAYLOAD

python3 - <<'PY'
import json
import os
import sys


def load_env_json(name):
    try:
        return json.loads(os.environ[name])
    except KeyError:
        print(f"missing {name}", file=sys.stderr)
        sys.exit(2)
    except json.JSONDecodeError as exc:
        print(f"{name} is not valid JSON: {exc}", file=sys.stderr)
        sys.exit(2)


health = load_env_json("HEALTH_JSON")
diag = load_env_json("DIAGNOSTICS_JSON")
jobs = load_env_json("JOBS_JSON")
require_deployed = os.environ.get("REQUIRE_DEPLOYED", "1") != "0"
allow_warnings = os.environ.get("ALLOW_WARNINGS", "1") != "0"
require_job_payload = os.environ.get("REQUIRE_JOB_PAYLOAD", "0") != "0"

failures = []
warnings = []

if health.get("ok") is not True:
    failures.append("backend health is not ok")

profiles = int(diag.get("profile_count") or 0)
installed = int(diag.get("installed_mods") or 0)
enabled = int(diag.get("enabled_mods") or 0)
needs_recovery = int(diag.get("needs_recovery") or 0)
blocked = int(diag.get("blocked_candidates") or 0)
active_install = int(diag.get("active_install_jobs") or 0)
active_deploy = int(diag.get("active_deploy_jobs") or 0)
deployment = diag.get("deployment") or {}
preview = diag.get("preview") or {}
validation_warnings = list(diag.get("validation_warnings") or [])
runtime_requirements = list(diag.get("runtime_requirements") or [])

deployed = bool(deployment.get("deployed"))
deployed_files = int(deployment.get("file_count") or 0)
preview_available = bool(preview.get("available"))
preview_conflicts = int(preview.get("conflicts") or 0)
preview_error = preview.get("error") or ""

if profiles < 1:
    failures.append("no profile exists for the selected game")
if installed < 1:
    failures.append("no installed profile mods are available")
if enabled < 1:
    failures.append("no enabled mods are available")
if active_install:
    failures.append(f"{active_install} install job(s) are still active")
if active_deploy:
    failures.append(f"{active_deploy} deploy/purge/repair job(s) are still active")
if needs_recovery:
    warnings.append(f"{needs_recovery} installed mod(s) need recovery")
if blocked:
    warnings.append(f"{blocked} downloaded archive(s) are blocked by unsupported install planning")
if require_deployed and not deployed:
    failures.append("no active deployment manifest exists")
if require_deployed and deployed_files < 1:
    failures.append("active deployment has no files")
if not preview_available:
    failures.append(f"deployment preview is unavailable: {preview_error or 'unknown error'}")
if preview_conflicts:
    failures.append(f"{preview_conflicts} deployment conflict(s) are present")
if validation_warnings:
    warnings.extend(validation_warnings)
    if not allow_warnings:
        failures.append("diagnostics reported validation warnings")

job_items = jobs.get("jobs") if isinstance(jobs, dict) else jobs
if isinstance(job_items, list):
    structured_app_jobs = []
    active = [
        job for job in job_items
        if job.get("status") not in ("completed", "canceled", "failed")
    ]
    for job in job_items:
        payload = job.get("payload") or {}
        if payload.get("app_id"):
            structured_app_jobs.append(job)
        if job.get("type") in ("captured-install", "deploy", "purge", "repair", "recover-downloads") and job.get("status") not in ("completed", "canceled", "failed"):
            if not payload.get("app_id"):
                failures.append(f"active {job.get('type')} job {job.get('id')} is missing structured app_id payload")
            if job.get("type") == "captured-install":
                missing = [key for key in ("catalog", "game_domain", "mod_id", "file_id") if not payload.get(key)]
                if missing:
                    failures.append(f"active captured install {job.get('id')} is missing payload fields: {', '.join(missing)}")
    if active:
        labels = ", ".join(f"{job.get('type', 'job')}:{job.get('status', 'unknown')}" for job in active[:5])
        failures.append(f"job list still has active work: {labels}")
    if require_job_payload and not structured_app_jobs:
        failures.append("job list does not contain any structured app_id payloads")

print("summary:")
print(f"  game={diag.get('game', {}).get('name', diag.get('app_id', 'unknown'))}")
print(f"  profiles={profiles} default={diag.get('default_profile') or 'none'}")
print(f"  installed={installed} enabled={enabled} needs_recovery={needs_recovery} blocked={blocked}")
print(f"  deployed={deployed} files={deployed_files} strategy={deployment.get('strategy') or 'none'}")
print(
    "  preview="
    f"available={preview_available} add={preview.get('add', 0)} "
    f"replace={preview.get('replace', 0)} remove={preview.get('remove', 0)} "
    f"keep={preview.get('keep', 0)} skip={preview.get('skip', 0)} "
    f"conflicts={preview_conflicts}"
)
print(f"  active_install_jobs={active_install} active_deploy_jobs={active_deploy}")
if runtime_requirements:
    print("  runtime_requirements=")
    for requirement in runtime_requirements:
        print(
            "    - "
            f"{requirement.get('name', requirement.get('id', 'unknown'))}: "
            f"{requirement.get('status', 'unknown')}"
        )

if warnings:
    print("\nwarnings:")
    for warning in dict.fromkeys(warnings):
        print(f"  - {warning}")

if failures:
    print("\nfailures:")
    for failure in failures:
        print(f"  - {failure}")
    sys.exit(1)

print("\nMVP live acceptance check passed")
PY
