#!/usr/bin/env bash
set -euo pipefail

PACKAGE="${PACKAGE:-${HOME}/.testing/decky-mod-manager.tar.gz}"
PLUGIN_DIR="${PLUGIN_DIR:-${HOME}/homebrew/plugins/decky-mod-manager}"

section() {
  printf '\n==> %s\n' "$1"
}

if [[ ! -f "${PACKAGE}" ]]; then
  echo "package not found: ${PACKAGE}" >&2
  exit 1
fi
if [[ ! -d "${PLUGIN_DIR}" ]]; then
  echo "installed plugin directory not found: ${PLUGIN_DIR}" >&2
  exit 1
fi

section "Comparing installed Decky plugin to package"
python3 - "$PACKAGE" "$PLUGIN_DIR" <<'PY'
import hashlib
import os
import sys
import tarfile
import tempfile
import zipfile
from pathlib import Path

package = Path(sys.argv[1]).expanduser()
installed = Path(sys.argv[2]).expanduser()

required = [
    "plugin.json",
    "package.json",
    "main.py",
    "bin/dmm-server",
    "bin/dmm-nxm-handler",
    "dist/index.js",
    "web/dist/index.html",
    "build-info.json",
]


def digest(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def collect(root: Path) -> dict[str, str]:
    files: dict[str, str] = {}
    for path in sorted(root.rglob("*")):
        if not path.is_file():
            continue
        rel = path.relative_to(root).as_posix()
        files[rel] = digest(path)
    return files


with tempfile.TemporaryDirectory(prefix="dmm-installed-package-") as tmp:
    extract_dir = Path(tmp)
    if package.suffix == ".zip":
        with zipfile.ZipFile(package) as zf:
            zf.extractall(extract_dir)
    else:
        with tarfile.open(package, "r:*") as tf:
            try:
                tf.extractall(extract_dir, filter="data")
            except TypeError:
                tf.extractall(extract_dir)

    roots = [p for p in extract_dir.iterdir() if p.is_dir()]
    if len(roots) != 1:
        print(f"expected package to contain one root directory, found {len(roots)}", file=sys.stderr)
        sys.exit(1)
    package_root = roots[0]

    for rel in required:
        if not (package_root / rel).exists():
            print(f"package missing required file: {rel}", file=sys.stderr)
            sys.exit(1)
        if not (installed / rel).exists():
            print(f"installed plugin missing required file: {rel}", file=sys.stderr)
            sys.exit(1)

    package_files = collect(package_root)
    installed_files = collect(installed)

    missing = sorted(set(package_files) - set(installed_files))
    changed = sorted(rel for rel in package_files if rel in installed_files and package_files[rel] != installed_files[rel])
    extra = sorted(set(installed_files) - set(package_files))

    if missing or changed:
        print("installed package check failed", file=sys.stderr)
        for rel in missing[:25]:
            print(f"missing installed file: {rel}", file=sys.stderr)
        for rel in changed[:25]:
            print(f"changed installed file: {rel}", file=sys.stderr)
        if len(missing) > 25 or len(changed) > 25:
            print("additional mismatches omitted", file=sys.stderr)
        sys.exit(1)

    print(f"package_files={len(package_files)}")
    print(f"installed_files={len(installed_files)}")
    print(f"extra_installed_files={len(extra)}")
    if extra:
        print("extra installed files are ignored:")
        for rel in extra[:20]:
            print(f"  {rel}")
        if len(extra) > 20:
            print("  ...")
    print("installed package matches staged package")
PY
