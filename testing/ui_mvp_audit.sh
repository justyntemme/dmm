#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Rendering phone UI shell"
(
  cd "${ROOT_DIR}"
  NODE_ENV=production node testing/ui_render_smoke.mjs
)

echo "==> Building Decky controller UI"
(
  cd "${ROOT_DIR}"
  npm --prefix decky run build >/dev/null
)

echo "UI behavior audit passed"
