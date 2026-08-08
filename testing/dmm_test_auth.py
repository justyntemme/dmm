import os
from pathlib import Path


def token() -> str:
    value = os.environ.get("DMM_AUTH_TOKEN", "").strip()
    if value:
        return value
    token_file = os.environ.get("DMM_TOKEN_FILE", "").strip()
    candidates = []
    if token_file:
        candidates.append(Path(token_file).expanduser())
    candidates.append(Path.home() / ".local" / "state" / "decky-mod-manager" / "api-token")
    for path in candidates:
        try:
            value = path.read_text(encoding="utf-8").strip()
        except OSError:
            continue
        if value:
            return value
    return ""


def auth_headers(headers=None):
    out = dict(headers or {})
    value = token()
    if value:
        out.setdefault("X-DMM-Token", value)
    return out
