import asyncio
import configparser
import datetime
import json
import os
import re
import secrets
import signal
import shutil
import socket
import subprocess
import tarfile
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path, PurePosixPath


class Plugin:
    backend_process = None
    log_dir = Path(os.environ.get("XDG_STATE_HOME", "/home/deck/.local/state")) / "decky-mod-manager"
    plugin_log = log_dir / "plugin.log"
    backend_log = log_dir / "backend.log"
    auth_token_file = log_dir / "api-token"
    desktop_id = "decky-mod-manager-nxm.desktop"
    nxm_schemes = ["x-scheme-handler/nxm", "x-scheme-handler/nxm-protocol"]
    sensitive_query_pattern = re.compile(r"(?i)(key|expires|md5)=([^&\"'\s]+)")
    update_package_name = "decky-mod-manager.tar.gz"
    update_max_package_bytes = 512 * 1024 * 1024
    update_required_files = [
        "plugin.json",
        "package.json",
        "main.py",
        "bin/dmm-server",
        "bin/dmm-nxm-handler",
        "dist/index.js",
        "web/dist/index.html",
        "build-info.json",
    ]

    async def _main(self):
        self._log("plugin loaded")
        asyncio.create_task(self._autostart_server())

    async def _unload(self):
        self._log("plugin unloading; leaving backend running")

    async def _autostart_server(self):
        try:
            result = await self._start_server("plugin autostart")
            self._log(f"plugin autostart result running={result.get('running')} pid={result.get('pid')} error={result.get('error')}")
        except Exception as exc:
            self._log(f"plugin autostart failed: {self._redact_url(str(exc))}")

    async def start_server(self):
        return await self._start_server("manual start")

    async def _start_server(self, reason):
        self._log(f"start_server requested reason={reason}")
        if self.backend_process and self.backend_process.poll() is None:
            self._log("backend already running")
            return await self.status()
        if self._backend_responds():
            backend_status, backend_error = self._backend_json_result("GET", "/api/status")
            if isinstance(backend_status, dict):
                self._log("backend already reachable on localhost with current auth")
                return await self.status()
            self._log(f"backend reachable but status check failed; restarting untracked backend error={self._redact_url(str(backend_error))}")
            await self._stop_backend("stale auth or unhealthy untracked backend")

        plugin_dir = Path(__file__).resolve().parent
        binary = plugin_dir / "bin" / "dmm-server"
        if not binary.exists():
            self._log(f"backend binary missing: {binary}")
            return {
                "running": False,
                "error": f"Backend binary not found: {binary}",
                "url": None,
            }

        env = os.environ.copy()
        env.setdefault("DMM_DECKY_PLUGIN_DIR", str(plugin_dir))
        loot_sorter = plugin_dir / "bin" / "dmm-loot-sorter"
        if loot_sorter.exists():
            env["DMM_LOOT_SORTER"] = str(loot_sorter)
            env["PATH"] = f"{loot_sorter.parent}:{env.get('PATH', '')}"
        token = self._ensure_auth_token()
        env["DMM_AUTH_TOKEN"] = token
        env["DMM_AUTH_TOKEN_FILE"] = str(self.auth_token_file)
        self.log_dir.mkdir(parents=True, exist_ok=True)
        backend_log = open(self.backend_log, "ab", buffering=0)
        self.backend_process = subprocess.Popen(
            [str(binary)],
            cwd=str(plugin_dir),
            env=env,
            stdout=backend_log,
            stderr=backend_log,
            start_new_session=True,
        )
        self._log(f"backend started pid={self.backend_process.pid} auth_token_file={self.auth_token_file}")
        for _ in range(10):
            if self._backend_responds():
                break
            await asyncio.sleep(0.25)
        return await self.status()

    async def stop_server(self):
        self._log("stop_server requested")
        await self._stop_backend("manual stop")
        return await self.status()

    async def reset_api_token(self):
        self._log("reset api token requested")
        was_running = self.backend_process is not None and self.backend_process.poll() is None or self._backend_responds()
        await self._stop_backend("api token reset")
        for path in [self.auth_token_file, self.auth_token_file.with_suffix(".tmp")]:
            try:
                path.unlink(missing_ok=True)
            except Exception as exc:
                self._log(f"reset api token cleanup failed path={path}: {self._redact_url(str(exc))}")
        if was_running:
            status = await self._start_server("api token reset")
        else:
            self._ensure_auth_token()
            status = await self.status()
        self._log(f"reset api token finished running={status.get('running')} token_file={self.auth_token_file}")
        return status

    async def _stop_backend(self, reason):
        self._log(f"backend stop requested reason={reason}")
        if self.backend_process and self.backend_process.poll() is None:
            os.killpg(os.getpgid(self.backend_process.pid), signal.SIGTERM)
            try:
                self.backend_process.wait(timeout=5)
                self._log("tracked backend stopped with SIGTERM")
            except subprocess.TimeoutExpired:
                os.killpg(os.getpgid(self.backend_process.pid), signal.SIGKILL)
                self._log("tracked backend killed after timeout")
        self.backend_process = None

        plugin_dir = Path(__file__).resolve().parent
        binary = plugin_dir / "bin" / "dmm-server"
        if self._backend_responds() and binary.exists():
            pattern = f"^{re.escape(str(binary))}$"
            result = self._run_command(["pkill", "-TERM", "-f", pattern], check=False)
            self._log(f"untracked backend stop requested rc={result.returncode} pattern={pattern}")
            for _ in range(20):
                if not self._backend_responds():
                    break
                await asyncio.sleep(0.25)
        if self._backend_responds() and binary.exists():
            pattern = f"^{re.escape(str(binary))}$"
            result = self._run_command(["pkill", "-KILL", "-f", pattern], check=False)
            self._log(f"untracked backend kill requested rc={result.returncode} pattern={pattern}")
            for _ in range(10):
                if not self._backend_responds():
                    break
                await asyncio.sleep(0.2)

    async def status(self):
        tracked_running = self.backend_process is not None and self.backend_process.poll() is None
        reachable = self._backend_responds()
        running = tracked_running or reachable
        ip = self._lan_ip()
        error = None
        pid = None
        if tracked_running:
            pid = self.backend_process.pid
        elif self.backend_process is not None and self.backend_process.returncode is not None:
            error = f"Backend exited with code {self.backend_process.returncode}. See {self.backend_log}."
        backend_status = self._backend_json("GET", "/api/status")
        auth_token = self._read_auth_token()
        result = {
            "running": running,
            "ip": ip,
            "port": 17942,
            "url": self._paired_phone_url(ip),
            "plain_url": f"http://{ip}:17942" if ip else None,
            "pid": pid,
            "backend": backend_status,
            "auth": {
                "enabled": bool(auth_token),
                "token": auth_token,
                "token_file": str(self.auth_token_file),
            },
            "logs": {
                "plugin": str(self.plugin_log),
                "backend": str(self.backend_log),
            },
            "build": self._build_info(),
            "error": error,
            "warning": "API authentication is enabled for paired phones. Keep LAN-only mode enabled unless using a trusted VPN/tunnel.",
        }
        return result

    async def nxm_status(self):
        return self._nxm_status()

    async def register_nxm_handler(self):
        self._log("register nxm handler requested")
        try:
            plugin_dir = Path(__file__).resolve().parent
            handler = plugin_dir / "bin" / "dmm-nxm-handler"
            self._log(f"register nxm handler plugin_dir={plugin_dir}")
            self._log(f"register nxm handler binary={handler} exists={handler.exists()}")
            if not handler.exists():
                return {"ok": False, "error": f"NXM handler missing: {handler}", "status": self._nxm_status()}

            desktop_dir = Path.home() / ".local" / "share" / "applications"
            desktop_dir.mkdir(parents=True, exist_ok=True)
            desktop_path = desktop_dir / self.desktop_id
            self._log(f"register nxm handler desktop_path={desktop_path}")
            desktop_path.write_text(
                "\n".join([
                    "[Desktop Entry]",
                    "Name=Decky Mod Manager NXM Handler",
                    f"Exec={handler} %u",
                    "Type=Application",
                    "NoDisplay=true",
                    "MimeType=x-scheme-handler/nxm;x-scheme-handler/nxm-protocol;",
                    "",
                ]),
                encoding="utf-8",
            )
            for scheme in self.nxm_schemes:
                self._write_mimeapps_default(scheme, self.desktop_id)
            self._run_command(["update-desktop-database", str(desktop_dir)], check=False)
            status = self._nxm_status()
            self._log(f"registered nxm handler status={status}")
            return {"ok": status["registered"], "status": status, "error": None if status["registered"] else "NXM handler was written but did not become the active handler."}
        except subprocess.CalledProcessError as exc:
            error = self._redact_url(exc.stderr or str(exc))
            self._log(f"register nxm handler failed: {error}")
            return {"ok": False, "error": error, "status": self._nxm_status()}
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"register nxm handler failed: {error}")
            return {"ok": False, "error": error, "status": self._nxm_status()}

    async def test_nxm_handler(self):
        test_url = self._debug_nxm_test_url()
        try:
            self._log("test nxm handler requested")
            if not self._backend_responds():
                return {"ok": False, "error": "Server must be running before testing the NXM handler.", "url": test_url}
            plugin_dir = Path(__file__).resolve().parent
            handler = plugin_dir / "bin" / "dmm-nxm-handler"
            if not handler.exists():
                return {"ok": False, "error": f"NXM handler missing: {handler}", "url": test_url}
            self._run_command([str(handler), test_url], check=True)
            return {"ok": True, "url": test_url}
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"test nxm handler failed: {error}")
            return {"ok": False, "error": error, "url": test_url}

    async def test_nxm_dispatch(self):
        test_url = self._debug_nxm_test_url()
        try:
            self._log("test nxm dispatch requested")
            if not self._backend_responds():
                return {"ok": False, "error": "Server must be running before testing NXM dispatch.", "url": test_url}
            result = self._run_command(["xdg-open", test_url], check=False)
            error = self._redact_url(result.stderr.strip()) if result.stderr.strip() else None
            return {"ok": result.returncode == 0, "error": error, "url": test_url}
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"test nxm dispatch failed: {error}")
            return {"ok": False, "error": error, "url": test_url}

    def _debug_nxm_test_url(self):
        domain = "examplegame"
        games = self._backend_json("GET", "/api/games")
        if isinstance(games, list):
            for game in games:
                if not isinstance(game, dict):
                    continue
                domains = game.get("nexus_domains")
                if isinstance(domains, list) and domains:
                    candidate = str(domains[0] or "").strip()
                    if candidate:
                        domain = candidate
                        break
        return f"nxm://{domain}/mods/3753/files/135998?mod_id=3753&file_id=135998&key=test&expires=1"

    async def add_captured_install(self, url, app_id="", profile_id=0):
        url = str(url or "").strip()
        app_id = str(app_id or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        self._log(f"add captured install requested app_id={app_id} profile_id={profile_id} url={self._redact_url(url)}")
        if not url:
            return {"ok": False, "error": "Mod URL is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {"url": url, "steam_app_id": app_id, "source": "decky-plugin"}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        payload = json.dumps(payload).encode("utf-8")
        result, error = self._backend_json_result("POST", "/api/captured-installs", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to capture mod link."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"add captured install accepted job={job}")
        return {"ok": True, "result": result}

    async def activate_game(self, app_id, profile_id=0):
        app_id = str(app_id or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {"source": "decky-active-game"}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        result, error = self._backend_json_result(
            "POST",
            f"/api/games/{urllib.parse.quote(app_id)}/activate",
            json.dumps(payload).encode("utf-8"),
        )
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to report the active game."}
        acquisitions = result.get("acquisitions")
        self._log(f"game activation reported app_id={app_id} event_handled={bool(result.get('event_handled'))} acquisitions={len(acquisitions) if isinstance(acquisitions, list) else 0}")
        return {"ok": True, "result": result}

    async def acquire_runtime_requirement(self, app_id, requirement_id, profile_id=0):
        app_id = str(app_id or "").strip()
        requirement_id = str(requirement_id or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if not app_id or not requirement_id:
            return {"ok": False, "error": "app_id and requirement_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        result, error = self._backend_json_result(
            "POST",
            f"/api/games/{urllib.parse.quote(app_id)}/requirements/{urllib.parse.quote(requirement_id)}/acquire",
            json.dumps(payload).encode("utf-8"),
        )
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to acquire runtime requirement."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"runtime requirement acquisition requested app_id={app_id} requirement_id={requirement_id} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def local_archives(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "roots": [], "files": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "roots": [], "files": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/local-archives")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load Deck archive files.", "roots": [], "files": []}
        files = result.get("files")
        roots = result.get("roots")
        if not isinstance(files, list):
            return {"ok": False, "error": "Unexpected local archive response.", "roots": [], "files": []}
        if not isinstance(roots, list):
            roots = []
        self._log(f"local archives loaded app_id={app_id} files={len(files)} roots={len(roots)}")
        return {"ok": True, "roots": roots, "files": files}

    async def browse_local_archives(self, app_id, path=""):
        app_id = str(app_id or "").strip()
        path = str(path or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "roots": [], "entries": [], "current_path": "", "parent_path": ""}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "roots": [], "entries": [], "current_path": "", "parent_path": ""}
        query = ""
        if path:
            query = "?path=" + urllib.parse.quote(path)
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/local-archives/browse{query}")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to browse Deck archive files.", "roots": [], "entries": [], "current_path": "", "parent_path": ""}
        entries = result.get("entries")
        roots = result.get("roots")
        if not isinstance(entries, list):
            return {"ok": False, "error": "Unexpected local archive browse response.", "roots": [], "entries": [], "current_path": "", "parent_path": ""}
        if not isinstance(roots, list):
            roots = []
        current_path = str(result.get("current_path") or "")
        parent_path = str(result.get("parent_path") or "")
        self._log(f"local archive browser loaded app_id={app_id} path={current_path} entries={len(entries)} roots={len(roots)}")
        return {"ok": True, "roots": roots, "entries": entries, "current_path": current_path, "parent_path": parent_path}

    async def import_local_archive(self, app_id, path, profile_id=0):
        app_id = str(app_id or "").strip()
        path = str(path or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if not app_id or not path:
            return {"ok": False, "error": "app_id and path are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {"path": path, "source": "decky-plugin-local-archive"}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        result, error = self._backend_json_result(
            "POST",
            f"/api/games/{urllib.parse.quote(app_id)}/local-archives/import",
            json.dumps(payload).encode("utf-8"),
        )
        if result is None:
            return {"ok": False, "error": error or "Unable to import Deck archive file."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"local archive import requested app_id={app_id} profile_id={profile_id} path={path} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def jobs(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "jobs": []}
        result, error = self._backend_json_result("GET", "/api/jobs")
        if result is None:
            return {"ok": False, "error": error or "Unable to load jobs.", "jobs": []}
        if isinstance(result, list):
            return {"ok": True, "jobs": result}
        return {"ok": False, "error": "Unexpected jobs response.", "jobs": []}

    async def cancel_job(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/jobs/{urllib.parse.quote(job_id)}/cancel", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to cancel job."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"job canceled job_id={job_id} status={(job or {}).get('status', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def install_captured_install(self, job_id, profile_id=0):
        job_id = str(job_id or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        result, error = self._backend_json_result("POST", f"/api/captured-installs/{urllib.parse.quote(job_id)}/install", json.dumps(payload).encode("utf-8"))
        if result is None:
            return {"ok": False, "error": error or "Unable to install captured mod."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"captured install started job_id={job_id} profile_id={profile_id} status={(job or {}).get('status', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def retry_captured_install(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/captured-installs/{urllib.parse.quote(job_id)}/retry", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to retry captured install."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"captured install retried job_id={job_id} status={(job or {}).get('status', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def games(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "games": []}
        result, error = self._backend_json_result("GET", "/api/games")
        if result is None:
            return {"ok": False, "error": error or "Unable to load games.", "games": []}
        if isinstance(result, list):
            return {"ok": True, "games": result}
        return {"ok": False, "error": "Unexpected games response.", "games": []}

    async def catalogs(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "catalogs": []}
        result, error = self._backend_json_result("GET", "/api/catalogs")
        if result is None:
            return {"ok": False, "error": error or "Unable to load mod sources.", "catalogs": []}
        if isinstance(result, list):
            self._log(f"catalogs loaded count={len(result)}")
            return {"ok": True, "catalogs": result}
        return {"ok": False, "error": "Unexpected catalogs response.", "catalogs": []}

    async def nexus_mods(self, app_id, query="", sort="downloads", time_window="all", count=20, offset=0, vortex_only=True, domain=""):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "mods": [], "total_count": 0}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "mods": [], "total_count": 0}
        try:
            count = int(count)
        except (TypeError, ValueError):
            count = 20
        try:
            offset = int(offset)
        except (TypeError, ValueError):
            offset = 0
        vortex_only_value = str(vortex_only).strip().lower()
        vortex_only_enabled = vortex_only_value not in ("0", "false", "no", "off")
        params = urllib.parse.urlencode({
            "q": str(query or "").strip(),
            "domain": str(domain or "").strip(),
            "sort": str(sort or "downloads").strip(),
            "time_window": str(time_window or "all").strip(),
            "count": max(1, min(count, 50)),
            "offset": max(0, offset),
            "vortex_only": "true" if vortex_only_enabled else "false",
        })
        path = f"/api/games/{urllib.parse.quote(app_id)}/nexus/mods?{params}"
        result, error = self._backend_json_result("GET", path)
        if result is None:
            return {"ok": False, "error": error or "Unable to search Nexus mods.", "mods": [], "total_count": 0}
        mods = result.get("mods") if isinstance(result, dict) else None
        if not isinstance(mods, list):
            return {"ok": False, "error": "Unexpected Nexus search response.", "mods": [], "total_count": 0}
        self._log(f"nexus mods searched app_id={app_id} domain_present={bool(str(domain or '').strip())} query_present={bool(str(query or '').strip())} sort={sort} time_window={time_window} count={len(mods)}")
        return {"ok": True, "mods": mods, "total_count": int(result.get("total_count") or len(mods))}

    async def game_profiles(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "profiles": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "profiles": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/profiles")
        if result is None:
            return {"ok": False, "error": error or "Unable to load profiles.", "profiles": []}
        if isinstance(result, list):
            return {"ok": True, "profiles": result}
        return {"ok": False, "error": "Unexpected profiles response.", "profiles": []}

    async def create_game_profile(self, app_id, name, source_profile_id=0):
        app_id = str(app_id or "").strip()
        name = str(name or "").strip()
        try:
            source_profile_id = int(source_profile_id or 0)
        except Exception:
            source_profile_id = 0
        if not app_id or not name:
            return {"ok": False, "error": "app_id and name are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"name": name, "source_profile_id": source_profile_id}).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/profiles", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to create profile."}
        self._log(f"profile created app_id={app_id} profile_id={result.get('id', '') if isinstance(result, dict) else ''} source_profile_id={source_profile_id}")
        return {"ok": True, "profile": result}

    async def game_mods(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "mods": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "mods": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/mods")
        if result is None:
            return {"ok": False, "error": error or "Unable to load mods.", "mods": []}
        if isinstance(result, list):
            return {"ok": True, "mods": result}
        return {"ok": False, "error": "Unexpected mods response.", "mods": []}

    async def game_diagnostics(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "diagnostics": None}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "diagnostics": None}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/diagnostics")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load game diagnostics.", "diagnostics": None}
        requirements = result.get("runtime_requirements")
        requirement_count = len(requirements) if isinstance(requirements, list) else 0
        self._log(f"game diagnostics loaded app_id={app_id} runtime_requirements={requirement_count}")
        return {"ok": True, "diagnostics": result}

    async def game_load_order(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "load_order": {"supported": False, "plugins": []}}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "load_order": {"supported": False, "plugins": []}}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/load-order")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load plugin load order.", "load_order": {"supported": False, "plugins": []}}
        plugins = result.get("plugins")
        if not isinstance(plugins, list):
            return {"ok": False, "error": "Unexpected load order response.", "load_order": {"supported": False, "plugins": []}}
        self._log(f"plugin load order loaded app_id={app_id} supported={bool(result.get('supported'))} plugins={len(plugins)}")
        return {"ok": True, "load_order": result}

    async def game_info(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "info": None}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "info": None}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/info")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load game info.", "info": None}
        details = result.get("details")
        detail_count = len(details) if isinstance(details, list) else 0
        self._log(f"game info loaded app_id={app_id} ran={bool(result.get('ran'))} details={detail_count}")
        return {"ok": True, "info": result}

    async def game_extension_actions(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "actions": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "actions": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/extension-actions")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load extension actions.", "actions": []}
        actions = result.get("actions")
        if not isinstance(actions, list):
            return {"ok": False, "error": "Unexpected extension actions response.", "actions": []}
        self._log(f"extension actions loaded app_id={app_id} actions={len(actions)}")
        return {"ok": True, "actions": actions}

    async def run_game_extension_action(self, app_id, action_id, profile_id=0):
        app_id = str(app_id or "").strip()
        action_id = str(action_id or "").strip()
        if not app_id or not action_id:
            return {"ok": False, "error": "app_id and action_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {}
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if profile_id > 0:
            payload["profile_id"] = profile_id
        result, error = self._backend_json_result(
            "POST",
            f"/api/games/{urllib.parse.quote(app_id)}/extension-actions/{urllib.parse.quote(action_id)}/run",
            json.dumps(payload).encode("utf-8"),
        )
        if result is None:
            return {"ok": False, "error": error or "Unable to run extension action."}
        self._log(f"extension action queued app_id={app_id} action_id={action_id} profile_id={profile_id}")
        return {"ok": True, "result": result}

    async def set_profile_plugin_activation(self, app_id, profile_id, activation_id, plugin_name, enabled):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        activation_id = str(activation_id or "").strip()
        plugin_name = str(plugin_name or "").strip()
        if not app_id or not profile_id or not activation_id or not plugin_name:
            return {"ok": False, "error": "app_id, profile_id, activation_id, and plugin_name are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"name": plugin_name, "enabled": bool(enabled)}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/plugin-activation/{urllib.parse.quote(activation_id)}/plugins", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to update plugin activation."}
        self._log(f"profile plugin activation updated app_id={app_id} profile_id={profile_id} activation_id={activation_id} plugin={plugin_name} enabled={bool(enabled)}")
        return {"ok": True, "result": result}

    async def set_profile_plugin_activation_order(self, app_id, profile_id, activation_id, plugins):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        activation_id = str(activation_id or "").strip()
        if not app_id or not profile_id or not activation_id:
            return {"ok": False, "error": "app_id, profile_id, and activation_id are required."}
        if not isinstance(plugins, list) or len(plugins) == 0:
            return {"ok": False, "error": "plugins must be a non-empty list."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"plugins": plugins}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/plugin-activation/{urllib.parse.quote(activation_id)}/order", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to update plugin load order."}
        self._log(f"profile plugin activation order updated app_id={app_id} profile_id={profile_id} activation_id={activation_id} plugins={len(plugins)}")
        return {"ok": True, "result": result}

    async def game_deploy_preview(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "plan": None}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "plan": None}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/deploy/preview")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to preview deployment.", "plan": None}
        actions = result.get("actions")
        conflicts = result.get("conflicts")
        if not isinstance(actions, list) or not isinstance(conflicts, list):
            return {"ok": False, "error": "Unexpected deployment preview response.", "plan": None}
        self._log(f"deployment preview loaded app_id={app_id} actions={len(actions)} conflicts={len(conflicts)}")
        return {"ok": True, "plan": result}

    async def game_deploy_status(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "status": None}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "status": None}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/deploy/status")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load deployment status.", "status": None}
        self._log(f"deployment status loaded app_id={app_id} deployed={bool(result.get('deployed'))} files={result.get('file_count')}")
        return {"ok": True, "status": result}

    async def game_deploy_history(self, app_id, limit=10):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "deployments": []}
        try:
            parsed_limit = int(limit)
        except Exception:
            parsed_limit = 10
        parsed_limit = max(1, min(parsed_limit, 50))
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "deployments": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/deploy/history?limit={parsed_limit}")
        if not isinstance(result, dict) or not isinstance(result.get("deployments"), list):
            return {"ok": False, "error": error or "Unable to load deployment history.", "deployments": []}
        deployments = result.get("deployments")
        self._log(f"deployment history loaded app_id={app_id} deployments={len(deployments)}")
        return {"ok": True, "deployments": deployments}

    async def preview_game_deployment_restore(self, app_id, deployment_id):
        app_id = str(app_id or "").strip()
        try:
            parsed_deployment_id = int(deployment_id)
        except Exception:
            parsed_deployment_id = 0
        if not app_id or parsed_deployment_id <= 0:
            return {"ok": False, "error": "app_id and deployment_id are required.", "preview": None}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "preview": None}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/deploy/history/{parsed_deployment_id}/preview")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to preview restore point.", "preview": None}
        self._log(f"deployment restore preview loaded app_id={app_id} deployment_id={parsed_deployment_id}")
        return {"ok": True, "preview": result}

    async def restore_game_deployment_point(self, app_id, deployment_id):
        app_id = str(app_id or "").strip()
        try:
            parsed_deployment_id = int(deployment_id)
        except Exception:
            parsed_deployment_id = 0
        if not app_id or parsed_deployment_id <= 0:
            return {"ok": False, "error": "app_id and deployment_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/deploy/history/{parsed_deployment_id}/restore", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to restore selected deployment point."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"deployment point restore requested app_id={app_id} deployment_id={parsed_deployment_id} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {
            "ok": True,
            "job": job,
            "plan": result.get("plan") if isinstance(result, dict) else None,
            "deployment_id": result.get("deployment_id") if isinstance(result, dict) else None,
        }

    async def set_file_conflict_winner(self, app_id, profile_id, target_path, winner_installed_mod_id):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        target_path = str(target_path or "").strip()
        winner_installed_mod_id = str(winner_installed_mod_id or "").strip()
        if not app_id or not profile_id or not target_path or not winner_installed_mod_id:
            return {"ok": False, "error": "app_id, profile_id, target_path, and winner_installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({
            "target_path": target_path,
            "winner_installed_mod_id": int(winner_installed_mod_id),
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/conflicts/winner", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to set file winner."}
        self._log(f"file conflict winner set app_id={app_id} profile_id={profile_id} target_path={target_path} winner_installed_mod_id={winner_installed_mod_id}")
        return {"ok": True, "result": result, "winner": result.get("winner"), "apply": result.get("apply")}

    async def clear_file_conflict_winner(self, app_id, profile_id, target_path):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        target_path = str(target_path or "").strip()
        if not app_id or not profile_id or not target_path:
            return {"ok": False, "error": "app_id, profile_id, and target_path are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        query = urllib.parse.urlencode({"target_path": target_path})
        result, error = self._backend_json_result("DELETE", f"/api/profiles/{urllib.parse.quote(profile_id)}/conflicts/winner?{query}")
        if result is None:
            return {"ok": False, "error": error or "Unable to reset file winner."}
        self._log(f"file conflict winner cleared app_id={app_id} profile_id={profile_id} target_path={target_path}")
        return {"ok": True, "result": result, "apply": result.get("apply")}

    async def check_game_mod_updates(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "results": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "results": []}
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/mods/check-updates")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to check mod updates.", "results": []}
        results = result.get("results")
        if not isinstance(results, list):
            return {"ok": False, "error": "Unexpected update check response.", "results": []}
        self._log(f"mod updates checked app_id={app_id} checked={result.get('checked')} results={len(results)}")
        return {"ok": True, "checked": int(result.get("checked") or 0), "results": results}

    async def game_workshop(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "items": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "items": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/workshop")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load Steam Workshop state.", "items": []}
        items = result.get("items")
        if not isinstance(items, list):
            return {"ok": False, "error": "Unexpected Steam Workshop response.", "items": []}
        return {"ok": True, "state": result, "items": items}

    async def queue_workshop_action(self, app_id, item_id, kind):
        app_id = str(app_id or "").strip()
        item_id = str(item_id or "").strip()
        kind = str(kind or "").strip()
        if not app_id or not item_id or not kind:
            return {"ok": False, "error": "app_id, item_id, and kind are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        path = f"/api/games/{urllib.parse.quote(app_id)}/workshop/items/{urllib.parse.quote(item_id)}/actions/{urllib.parse.quote(kind)}"
        result, error = self._backend_json_result("POST", path, b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to queue Steam Workshop action."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"workshop action queued app_id={app_id} item_id={item_id} kind={kind} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def queue_workshop_order(self, app_id, item_ids):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not isinstance(item_ids, list):
            return {"ok": False, "error": "item_ids must be a list."}
        cleaned = [str(item_id or "").strip() for item_id in item_ids]
        if not all(cleaned):
            return {"ok": False, "error": "item_ids cannot include empty values."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"item_ids": cleaned}).encode("utf-8")
        path = f"/api/games/{urllib.parse.quote(app_id)}/workshop/order"
        result, error = self._backend_json_result("PUT", path, payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to queue Steam Workshop load order."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"workshop order queued app_id={app_id} items={len(cleaned)} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def game_install_candidates(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required.", "candidates": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "candidates": []}
        result, error = self._backend_json_result("GET", f"/api/games/{urllib.parse.quote(app_id)}/install-candidates")
        if result is None:
            return {"ok": False, "error": error or "Unable to load installer choices.", "candidates": []}
        if isinstance(result, list):
            return {"ok": True, "candidates": result}
        return {"ok": False, "error": "Unexpected installer choices response.", "candidates": []}

    async def clear_game_install_candidates(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("DELETE", f"/api/games/{urllib.parse.quote(app_id)}/install-candidates")
        if result is None:
            return {"ok": False, "error": error or "Unable to clear installer items."}
        self._log(f"install candidates cleared app_id={app_id} result={result}")
        return {"ok": True, "result": result}

    async def retry_install_candidate(self, app_id, candidate_id):
        app_id = str(app_id or "").strip()
        candidate_id = str(candidate_id or "").strip()
        if not app_id or not candidate_id:
            return {"ok": False, "error": "app_id and candidate_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        path = f"/api/games/{urllib.parse.quote(app_id)}/install-candidates/{urllib.parse.quote(candidate_id)}/retry"
        result, error = self._backend_json_result("POST", path)
        if result is None:
            return {"ok": False, "error": error or "Unable to retry installer item."}
        job = result.get("job") if isinstance(result, dict) else None
        candidate = result.get("candidate") if isinstance(result, dict) else None
        mod = result.get("mod") if isinstance(result, dict) else None
        self._log(
            "install candidate retry requested "
            f"app_id={app_id} candidate_id={candidate_id} "
            f"job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''} "
            f"candidate_status={(candidate or {}).get('status', '') if isinstance(candidate, dict) else ''}"
        )
        return {"ok": True, "result": result, "job": job, "candidate": candidate, "mod": mod}

    async def apply_install_candidate(self, app_id, candidate_id, selections, profile_id=0):
        app_id = str(app_id or "").strip()
        candidate_id = str(candidate_id or "").strip()
        try:
            profile_id = int(profile_id or 0)
        except (TypeError, ValueError):
            profile_id = 0
        if not app_id or not candidate_id:
            return {"ok": False, "error": "app_id and candidate_id are required."}
        if not isinstance(selections, dict):
            selections = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = {"selections": selections}
        if profile_id > 0:
            payload["profile_id"] = profile_id
        payload = json.dumps(payload).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/install-candidates/{urllib.parse.quote(candidate_id)}/apply", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to apply installer choices."}
        job = result.get("job") if isinstance(result, dict) else None
        if isinstance(job, dict) and str(job.get("status") or "").lower() == "failed":
            return {"ok": False, "error": job.get("message") or "Unable to apply installer choices.", "result": result}
        self._log(f"installer choices applied app_id={app_id} candidate_id={candidate_id} profile_id={profile_id}")
        return {"ok": True, "result": result}

    async def save_install_candidate_choices(self, app_id, candidate_id, selections):
        app_id = str(app_id or "").strip()
        candidate_id = str(candidate_id or "").strip()
        if not app_id or not candidate_id:
            return {"ok": False, "error": "app_id and candidate_id are required."}
        if not isinstance(selections, dict):
            selections = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"selections": selections}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/games/{urllib.parse.quote(app_id)}/install-candidates/{urllib.parse.quote(candidate_id)}/choices", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to save installer choices."}
        self._log(f"installer choices saved app_id={app_id} candidate_id={candidate_id} groups={len(selections)}")
        return {"ok": True, "candidate": result.get("candidate")}

    async def set_default_profile(self, profile_id):
        profile_id = str(profile_id or "").strip()
        if not profile_id:
            return {"ok": False, "error": "profile_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/default", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to select profile."}
        self._log(f"default profile selected profile_id={profile_id}")
        return {"ok": True, "profile": result.get("profile"), "apply": result.get("apply")}

    async def set_profile_mod_enabled(self, app_id, profile_id, installed_mod_id, enabled):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not profile_id or not installed_mod_id:
            return {"ok": False, "error": "app_id, profile_id, and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({
            "enabled": bool(enabled),
            "cascade_dependencies": True,
            "include_recommended_dependencies": True,
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/mods/{urllib.parse.quote(installed_mod_id)}", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to update mod."}
        cascade = result.get("cascade") or []
        notes = result.get("cascade_notes") or []
        self._log(f"profile mod updated app_id={app_id} profile_id={profile_id} installed_mod_id={installed_mod_id} enabled={bool(enabled)} cascade={len(cascade)} notes={len(notes)}")
        return {"ok": True, "mod": result.get("mod"), "cascade": cascade, "cascade_notes": notes, "apply": result.get("apply")}

    async def set_profile_mod_order(self, app_id, profile_id, mod_ids):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        if not app_id or not profile_id:
            return {"ok": False, "error": "app_id and profile_id are required."}
        if not isinstance(mod_ids, list) or len(mod_ids) == 0:
            return {"ok": False, "error": "mod_ids must be a non-empty list."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"mod_ids": mod_ids}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/mods/order", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to update mod order."}
        self._log(f"profile mod order updated app_id={app_id} profile_id={profile_id} mods={len(mod_ids)}")
        return {"ok": True, "mods": result.get("mods"), "apply": result.get("apply")}

    async def remove_profile_mod(self, app_id, profile_id, installed_mod_id):
        app_id = str(app_id or "").strip()
        profile_id = str(profile_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not profile_id or not installed_mod_id:
            return {"ok": False, "error": "app_id, profile_id, and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("DELETE", f"/api/profiles/{urllib.parse.quote(profile_id)}/mods/{urllib.parse.quote(installed_mod_id)}")
        if result is None:
            return {"ok": False, "error": error or "Unable to remove mod."}
        self._log(f"profile mod removed app_id={app_id} profile_id={profile_id} installed_mod_id={installed_mod_id}")
        return {"ok": True, "result": result}

    async def delete_game_mod(self, app_id, installed_mod_id):
        app_id = str(app_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not installed_mod_id:
            return {"ok": False, "error": "app_id and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("DELETE", f"/api/games/{urllib.parse.quote(app_id)}/mods/{urllib.parse.quote(installed_mod_id)}")
        if result is None:
            return {"ok": False, "error": error or "Unable to uninstall mod."}
        self._log(f"installed mod deleted app_id={app_id} installed_mod_id={installed_mod_id}")
        return {"ok": True, "result": result}

    async def reinstall_game_mod(self, app_id, installed_mod_id, prompt_installer_choices=False):
        app_id = str(app_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not installed_mod_id:
            return {"ok": False, "error": "app_id and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"prompt_installer_choices": bool(prompt_installer_choices)}).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/mods/{urllib.parse.quote(installed_mod_id)}/reinstall", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to reinstall mod."}
        self._log(f"mod reinstall requested app_id={app_id} installed_mod_id={installed_mod_id} prompt_installer_choices={bool(prompt_installer_choices)}")
        return {"ok": True, "result": result}

    async def update_game_mod(self, app_id, installed_mod_id):
        app_id = str(app_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not installed_mod_id:
            return {"ok": False, "error": "app_id and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/mods/{urllib.parse.quote(installed_mod_id)}/update", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to install mod update."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"mod update requested app_id={app_id} installed_mod_id={installed_mod_id} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "result": result, "job": job}

    async def sync_workshop(self, app_id, items):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not isinstance(items, list):
            items = []
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"items": items}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/games/{urllib.parse.quote(app_id)}/workshop/sync", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to sync Steam Workshop state."}
        self._log(f"workshop synced app_id={app_id} items={len(items)}")
        return {"ok": True, "result": result}

    async def workshop_actions(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "actions": []}
        result, error = self._backend_json_result("GET", "/api/workshop/actions")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load Steam Workshop actions.", "actions": []}
        actions = result.get("actions")
        if not isinstance(actions, list):
            return {"ok": False, "error": "Unexpected Steam Workshop actions response.", "actions": []}
        if actions:
            self._log(f"workshop actions available count={len(actions)}")
        return {"ok": True, "actions": actions}

    async def start_workshop_action(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required.", "proceed": False}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "proceed": False}
        result, error = self._backend_json_result("POST", f"/api/workshop/actions/{urllib.parse.quote(job_id)}/start", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to start Steam Workshop action.", "proceed": False}
        proceed = bool(result.get("proceed")) if isinstance(result, dict) else False
        self._log(f"workshop action start job_id={job_id} proceed={proceed}")
        return {"ok": True, "proceed": proceed, "job": result.get("job") if isinstance(result, dict) else None}

    async def retry_workshop_action(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/workshop/actions/{urllib.parse.quote(job_id)}/retry", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to retry Steam Workshop action."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"workshop action retry queued job_id={job_id} status={(job or {}).get('status', '') if isinstance(job, dict) else ''}")
        return {"ok": True, "job": job, "result": result}

    async def record_workshop_action(self, job_id, report):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not isinstance(report, dict):
            report = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps(report).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/workshop/actions/{urllib.parse.quote(job_id)}/complete", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to record Steam Workshop action."}
        self._log(f"workshop action report job_id={job_id} applied={bool(report.get('applied'))} error={report.get('error', '')}")
        return {"ok": True, "job": result.get("job") if isinstance(result, dict) else None}

    async def start_extension_notice_action(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required.", "proceed": False}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "proceed": False}
        result, error = self._backend_json_result("POST", f"/api/extension-notices/{urllib.parse.quote(job_id)}/start", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to start extension notice action.", "proceed": False}
        proceed = bool(result.get("proceed")) if isinstance(result, dict) else False
        self._log(f"extension notice action start job_id={job_id} proceed={proceed}")
        return {"ok": True, "proceed": proceed, "job": result.get("job") if isinstance(result, dict) else None}

    async def record_extension_notice_action(self, job_id, report):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not isinstance(report, dict):
            report = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps(report).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/extension-notices/{urllib.parse.quote(job_id)}/complete", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to record extension notice action."}
        self._log(f"extension notice action report job_id={job_id} applied={bool(report.get('applied'))} error={report.get('error', '')}")
        return {"ok": True, "job": result.get("job") if isinstance(result, dict) else None}

    async def tool_actions(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "actions": []}
        result, error = self._backend_json_result("GET", "/api/tool/actions")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load extension tool actions.", "actions": []}
        actions = result.get("actions")
        if not isinstance(actions, list):
            return {"ok": False, "error": "Unexpected extension tool actions response.", "actions": []}
        if actions:
            self._log(f"extension tool actions available count={len(actions)}")
        return {"ok": True, "actions": actions}

    async def start_tool_action(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required.", "proceed": False}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "proceed": False}
        result, error = self._backend_json_result("POST", f"/api/tool/actions/{urllib.parse.quote(job_id)}/start", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to start extension tool action.", "proceed": False}
        proceed = bool(result.get("proceed")) if isinstance(result, dict) else False
        self._log(f"extension tool action start job_id={job_id} proceed={proceed}")
        return {"ok": True, "proceed": proceed, "job": result.get("job") if isinstance(result, dict) else None}

    async def record_tool_action(self, job_id, report):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not isinstance(report, dict):
            report = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps(report).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/tool/actions/{urllib.parse.quote(job_id)}/complete", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to record extension tool action."}
        self._log(f"extension tool action report job_id={job_id} applied={bool(report.get('applied'))} error={report.get('error', '')}")
        return {"ok": True, "job": result.get("job") if isinstance(result, dict) else None}

    async def open_directory_actions(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "actions": []}
        result, error = self._backend_json_result("GET", "/api/open-directory/actions")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load open-directory actions.", "actions": []}
        actions = result.get("actions")
        if not isinstance(actions, list):
            return {"ok": False, "error": "Unexpected open-directory actions response.", "actions": []}
        if actions:
            self._log(f"open-directory actions available count={len(actions)}")
        return {"ok": True, "actions": actions}

    async def start_open_directory_action(self, job_id):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required.", "proceed": False}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "proceed": False}
        result, error = self._backend_json_result("POST", f"/api/open-directory/actions/{urllib.parse.quote(job_id)}/start", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to start open-directory action.", "proceed": False}
        proceed = bool(result.get("proceed")) if isinstance(result, dict) else False
        self._log(f"open-directory action start job_id={job_id} proceed={proceed}")
        return {"ok": True, "proceed": proceed, "job": result.get("job") if isinstance(result, dict) else None}

    async def record_open_directory_action(self, job_id, report):
        job_id = str(job_id or "").strip()
        if not job_id:
            return {"ok": False, "error": "job_id is required."}
        if not isinstance(report, dict):
            report = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps(report).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/open-directory/actions/{urllib.parse.quote(job_id)}/complete", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to record open-directory action."}
        self._log(f"open-directory action report job_id={job_id} applied={bool(report.get('applied'))} error={report.get('error', '')}")
        return {"ok": True, "job": result.get("job") if isinstance(result, dict) else None}

    async def open_directory_path(self, path):
        target = Path(str(path or "")).expanduser()
        if not target.is_absolute():
            return {"ok": False, "error": "Path must be absolute."}
        try:
            resolved = target.resolve(strict=True)
        except FileNotFoundError:
            return {"ok": False, "error": "Path does not exist."}
        except Exception as exc:
            return {"ok": False, "error": self._redact_url(str(exc))}
        try:
            subprocess.Popen(["xdg-open", str(resolved)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)
            self._log(f"open-path launched path={resolved}")
            return {"ok": True}
        except FileNotFoundError:
            return {"ok": False, "error": "xdg-open is not installed."}
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"open-path launch failed path={resolved}: {error}")
            return {"ok": False, "error": error}

    async def launch_actions(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "actions": []}
        result, error = self._backend_json_result("GET", "/api/launch/actions")
        if not isinstance(result, dict):
            return {"ok": False, "error": error or "Unable to load launch actions.", "actions": []}
        actions = result.get("actions")
        if not isinstance(actions, list):
            return {"ok": False, "error": "Unexpected launch actions response.", "actions": []}
        if actions:
            self._log(f"launch actions available count={len(actions)}")
        return {"ok": True, "actions": actions}

    async def record_launch_action(self, app_id, report):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not isinstance(report, dict):
            report = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps(report).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/launch/configure", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to record launch configuration."}
        self._log(f"launch action report app_id={app_id} applied={bool(report.get('applied'))} error={report.get('error', '')}")
        return {"ok": True, "status": result}

    async def frontend_log(self, message, detail=None):
        message = str(message or "").strip()
        if not isinstance(detail, dict):
            detail = {}
        safe_detail = {}
        for key, value in detail.items():
            key = str(key)
            if key.lower() in {"url", "source_url", "nxm_url", "api_key", "key", "token"}:
                safe_detail[key] = "[redacted]"
            else:
                safe_detail[key] = self._redact_url(str(value))
        self._log(f"frontend {message} detail={safe_detail}")
        return {"ok": True}

    async def dependencies(self):
        return [
            self._dependency(
                "7-Zip",
                "7z",
                "Extracts .7z and many Nexus archive formats.",
                "p7zip",
                "sudo pacman -S --needed p7zip",
                "https://wiki.archlinux.org/title/P7zip",
            ),
            self._dependency(
                "bsdtar",
                "bsdtar",
                "Extracts tar and zip archives.",
                "libarchive",
                "sudo pacman -S --needed libarchive",
                "https://man.archlinux.org/man/bsdtar.1",
            ),
            self._dependency(
                "unzip",
                "unzip",
                "Extracts .zip archives.",
                "unzip",
                "sudo pacman -S --needed unzip",
                "https://man.archlinux.org/man/unzip.1.en",
            ),
            self._dependency(
                "unrar",
                "unrar",
                "Extracts .rar archives when available.",
                "unrar",
                "sudo pacman -S --needed unrar",
                "https://man.archlinux.org/man/unrar.1.en",
            ),
        ]

    async def set_lan_only(self, lan_only):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        payload = json.dumps({"lan_only": bool(lan_only)}).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/security", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update server settings.",
            }
        return {
            "ok": True,
            "status": result,
        }

    async def set_auto_install_captured_downloads(self, auto_install):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        status, status_error = self._backend_json_result("GET", "/api/status")
        if not isinstance(status, dict):
            return {
                "ok": False,
                "error": status_error or "Unable to load current install settings.",
            }
        install = status.get("install") if isinstance(status.get("install"), dict) else {}
        payload = json.dumps({
            "auto_install_captured_downloads": bool(auto_install),
            "auto_enable_installed_mods": bool(install.get("auto_enable_installed_mods", False)),
            "auto_show_fomod_installers": bool(install.get("auto_show_fomod_installers", True)),
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/install", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update install settings.",
            }
        self._log(f"auto-install captured downloads set to {bool(auto_install)}")
        return {
            "ok": True,
            "status": result,
        }

    async def set_auto_enable_installed_mods(self, auto_enable):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        status, status_error = self._backend_json_result("GET", "/api/status")
        if not isinstance(status, dict):
            return {
                "ok": False,
                "error": status_error or "Unable to load current install settings.",
            }
        install = status.get("install") if isinstance(status.get("install"), dict) else {}
        payload = json.dumps({
            "auto_install_captured_downloads": bool(install.get("auto_install_captured_downloads", False)),
            "auto_enable_installed_mods": bool(auto_enable),
            "auto_show_fomod_installers": bool(install.get("auto_show_fomod_installers", True)),
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/install", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update enable settings.",
            }
        self._log(f"auto-enable installed mods set to {bool(auto_enable)}")
        return {
            "ok": True,
            "status": result,
        }

    async def set_auto_show_fomod_installers(self, auto_show):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        status, status_error = self._backend_json_result("GET", "/api/status")
        if not isinstance(status, dict):
            return {
                "ok": False,
                "error": status_error or "Unable to load current install settings.",
            }
        install = status.get("install") if isinstance(status.get("install"), dict) else {}
        payload = json.dumps({
            "auto_install_captured_downloads": bool(install.get("auto_install_captured_downloads", False)),
            "auto_enable_installed_mods": bool(install.get("auto_enable_installed_mods", False)),
            "auto_show_fomod_installers": bool(auto_show),
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/install", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update FOMOD installer settings.",
            }
        self._log(f"auto-show FOMOD installers set to {bool(auto_show)}")
        return {
            "ok": True,
            "status": result,
        }

    async def set_max_concurrent_captured_downloads(self, max_downloads):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        try:
            max_downloads = int(max_downloads)
        except (TypeError, ValueError):
            return {
                "ok": False,
                "error": "Download slot count is invalid.",
            }
        payload = json.dumps({
            "max_concurrent_captured_downloads": max_downloads,
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/downloads", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update download settings.",
            }
        normalized = ((result.get("download") or {}).get("max_concurrent_captured_downloads")
                      if isinstance(result, dict) else max_downloads)
        self._log(f"max concurrent captured downloads set to {normalized}")
        return {
            "ok": True,
            "status": result,
        }

    async def set_max_concurrent_captured_downloads_per_game(self, max_downloads):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        try:
            max_downloads = int(max_downloads)
        except (TypeError, ValueError):
            return {
                "ok": False,
                "error": "Per-game download slot count is invalid.",
            }
        payload = json.dumps({
            "max_concurrent_captured_downloads_per_game": max_downloads,
        }).encode("utf-8")
        result, error = self._backend_json_result("PUT", "/api/settings/downloads", payload)
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update per-game download settings.",
            }
        normalized = ((result.get("download") or {}).get("max_concurrent_captured_downloads_per_game")
                      if isinstance(result, dict) else max_downloads)
        self._log(f"max concurrent captured downloads per game set to {normalized}")
        return {
            "ok": True,
            "status": result,
        }

    async def patch_ui_preferences(self, patch=None):
        if not self._backend_responds():
            return {
                "ok": False,
                "error": "Server is not running.",
            }
        if not isinstance(patch, dict):
            return {
                "ok": False,
                "error": "UI preference patch is invalid.",
            }
        payload = {}
        favorite_game_id = str(patch.get("favorite_game_id") or "").strip()
        if favorite_game_id:
            payload["favorite_game_id"] = favorite_game_id
            payload["favorite"] = bool(patch.get("favorite"))
        recent_game_id = str(patch.get("recent_game_id") or "").strip()
        if recent_game_id:
            payload["recent_game_id"] = recent_game_id
            try:
                payload["recent_at"] = int(patch.get("recent_at") or 0)
            except (TypeError, ValueError):
                payload["recent_at"] = 0
        game_sort = str(patch.get("game_sort") or "").strip()
        if game_sort:
            payload["game_sort"] = game_sort
        result, error = self._backend_json_result("PATCH", "/api/settings/ui", json.dumps(payload).encode("utf-8"))
        if result is None:
            return {
                "ok": False,
                "error": error or "Unable to update UI preferences.",
            }
        self._log(f"ui preferences patched favorite_game_id={favorite_game_id} recent_game_id={recent_game_id} sort={game_sort}")
        return {
            "ok": True,
            "status": await self.status(),
        }

    async def diagnostics(self):
        self._log("diagnostics requested")
        nxm_log = self.log_dir / "nxm-handler.log"
        update_install_log = self.log_dir / "update-install.log"
        steam_js = Path.home() / ".local" / "share" / "Steam" / "logs" / "webhelper_js.txt"
        tail_lines = 160
        tail_bytes = 96000
        return {
            "status": await self.status(),
            "nxm": self._nxm_status(),
            "dependencies": await self.dependencies(),
            "logs": {
                "plugin": {
                    "path": str(self.plugin_log),
                    "tail": self._tail_file(self.plugin_log, max_lines=tail_lines, max_bytes=tail_bytes),
                },
                "backend": {
                    "path": str(self.backend_log),
                    "tail": self._tail_file(self.backend_log, max_lines=tail_lines, max_bytes=tail_bytes),
                },
                "nxm": {
                    "path": str(nxm_log),
                    "tail": self._tail_file(nxm_log, max_lines=tail_lines, max_bytes=tail_bytes),
                },
                "update_install": {
                    "path": str(update_install_log),
                    "tail": self._tail_file(update_install_log, max_lines=tail_lines, max_bytes=tail_bytes),
                },
                "steam_js": {
                    "path": str(steam_js),
                    "tail": self._tail_file(steam_js, max_lines=tail_lines, max_bytes=tail_bytes),
                },
            },
        }

    async def install_latest_update(self):
        repo = os.environ.get("DMM_UPDATE_REPO", "justyntemme/dmm").strip() or "justyntemme/dmm"
        package_url = os.environ.get("DMM_UPDATE_PACKAGE_URL", "").strip()
        release = os.environ.get("DMM_UPDATE_RELEASE", "").strip()
        try:
            if not package_url:
                package_url, release = await asyncio.to_thread(self._resolve_update_package, repo, release)
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"latest update resolve failed repo={repo} release={release or 'latest'} error={error}")
            return {
                "ok": False,
                "error": error,
                "message": "DMM could not find a published GitHub release package.",
                "url": "",
            }
        testing_dir = Path(os.environ.get("DMM_UPDATE_STAGING_DIR", "/home/deck/.testing")).expanduser()
        package_path = testing_dir / self.update_package_name
        tmp_path = testing_dir / f".{self.update_package_name}.download"
        wrapper = Path(os.environ.get("DMM_UPDATE_WRAPPER", "/opt/decky-mod-manager-testing/bin/decky-mod-manager-test-install"))
        update_log = self.log_dir / "update-install.log"

        self._log(f"latest update requested repo={repo} release={release} package_url={self._redact_url(package_url)} wrapper={wrapper}")
        if not wrapper.exists():
            error = f"Privileged installer wrapper is not installed: {wrapper}"
            self._log(f"latest update blocked: {error}")
            return {
                "ok": False,
                "error": error,
                "message": "Run the testing sudoers bootstrap once from Konsole or SSH before using Gaming Mode updates.",
                "package": str(package_path),
                "url": self._redact_url(package_url),
                "log": str(update_log),
            }
        if not os.access(wrapper, os.X_OK):
            error = f"Privileged installer wrapper is not executable: {wrapper}"
            self._log(f"latest update blocked: {error}")
            return {
                "ok": False,
                "error": error,
                "package": str(package_path),
                "url": self._redact_url(package_url),
                "log": str(update_log),
            }

        try:
            downloaded = await asyncio.to_thread(self._download_update_package, package_url, tmp_path, package_path)
        except Exception as exc:
            try:
                tmp_path.unlink(missing_ok=True)
            except Exception:
                pass
            error = self._redact_url(str(exc))
            self._log(f"latest update download failed: {error}")
            return {
                "ok": False,
                "error": error,
                "message": "DMM could not download or validate the latest release package.",
                "package": str(package_path),
                "url": self._redact_url(package_url),
                "log": str(update_log),
            }

        try:
            update_log.parent.mkdir(parents=True, exist_ok=True)
            log_handle = open(update_log, "ab", buffering=0)
            try:
                process = subprocess.Popen(
                    ["sudo", "-n", str(wrapper)],
                    cwd=str(Path.home()),
                    env=self._clean_env(),
                    stdout=log_handle,
                    stderr=log_handle,
                    start_new_session=True,
                )
            finally:
                log_handle.close()
            await asyncio.sleep(1)
            return_code = process.poll()
            if return_code is not None and return_code != 0:
                tail = self._tail_file(update_log, max_lines=20)
                error = f"Installer exited with code {return_code}."
                self._log(f"latest update installer failed rc={return_code} tail={tail[-500:]}")
                return {
                    "ok": False,
                    "error": error,
                    "message": tail or error,
                    "package": str(package_path),
                    "url": self._redact_url(package_url),
                    "bytes": downloaded,
                    "log": str(update_log),
                }
            self._log(f"latest update installer started pid={process.pid} package={package_path} bytes={downloaded}")
            return {
                "ok": True,
                "message": "Latest package downloaded. The installer started and will update DMM without rebooting by default.",
                "package": str(package_path),
                "url": self._redact_url(package_url),
                "bytes": downloaded,
                "installer_pid": process.pid,
                "log": str(update_log),
            }
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"latest update installer launch failed: {error}")
            return {
                "ok": False,
                "error": error,
                "message": "DMM downloaded the package but could not start the privileged installer.",
                "package": str(package_path),
                "url": self._redact_url(package_url),
                "bytes": downloaded,
                "log": str(update_log),
            }

    def _resolve_update_package(self, repo, release):
        repo = str(repo or "").strip()
        if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo):
            raise RuntimeError("DMM_UPDATE_REPO must look like owner/repo")
        release = str(release or "").strip()
        if release:
            return f"https://github.com/{repo}/releases/download/{urllib.parse.quote(release)}/{self.update_package_name}", release

        endpoint = f"https://api.github.com/repos/{repo}/releases/latest"
        request = urllib.request.Request(endpoint, headers={"User-Agent": "Decky-Mod-Manager-Updater"})
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                payload = json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            raise RuntimeError(f"GitHub latest release lookup failed with HTTP {exc.code}") from exc
        except urllib.error.URLError as exc:
            raise RuntimeError(f"GitHub latest release lookup failed: {exc.reason}") from exc

        tag = str(payload.get("tag_name") or "").strip()
        assets = payload.get("assets") if isinstance(payload, dict) else None
        if not tag or not isinstance(assets, list):
            raise RuntimeError("GitHub latest release response did not include a tag and assets")
        for asset in assets:
            if not isinstance(asset, dict):
                continue
            if str(asset.get("name") or "") != self.update_package_name:
                continue
            url = str(asset.get("browser_download_url") or "").strip()
            if not url:
                break
            return url, tag
        raise RuntimeError(f"latest release {tag} does not include {self.update_package_name}")

    def _download_update_package(self, package_url, tmp_path, package_path):
        tmp_path.parent.mkdir(parents=True, exist_ok=True)
        downloaded = self._download_file(package_url, tmp_path, self.update_max_package_bytes)
        self._validate_update_package(tmp_path)
        tmp_path.replace(package_path)
        package_path.chmod(0o644)
        return downloaded

    def _download_file(self, url, target, max_bytes):
        curl = shutil.which("curl")
        if not curl:
            raise RuntimeError("curl is required to download DMM updates")
        result = subprocess.run(
            [
                curl,
                "-fL",
                "--retry",
                "3",
                "--connect-timeout",
                "20",
                "--max-time",
                "600",
                "-A",
                "Decky-Mod-Manager-Updater",
                "-o",
                str(target),
                url,
            ],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=self._clean_env(),
        )
        stderr = self._redact_url(result.stderr.strip())
        stdout = self._redact_url(result.stdout.strip())
        self._log(f"update download curl result rc={result.returncode} stdout={stdout[:500]} stderr={stderr[:1200]}")
        if result.returncode != 0:
            raise RuntimeError(f"curl failed with code {result.returncode}: {stderr or stdout or 'no output'}")
        total = target.stat().st_size if target.exists() else 0
        if total == 0:
            raise RuntimeError("downloaded package was empty")
        if total > max_bytes:
            raise RuntimeError(f"package exceeded maximum size: {max_bytes} bytes")
        return total

    def _validate_update_package(self, package_path):
        members = {}
        try:
            with tarfile.open(package_path, "r:gz") as archive:
                for member in archive.getmembers():
                    clean_name = member.name.strip("/")
                    self._validate_update_member(member, clean_name)
                    members[clean_name] = member
        except tarfile.TarError as exc:
            raise RuntimeError(f"package is not a valid tar.gz archive: {exc}") from exc

        root = "decky-mod-manager"
        if not any(name == root or name.startswith(f"{root}/") for name in members):
            raise RuntimeError("package does not contain decky-mod-manager/")
        for relative in self.update_required_files:
            member = members.get(f"{root}/{relative}")
            if member is None:
                raise RuntimeError(f"package is missing required file: {relative}")
            if not member.isfile():
                raise RuntimeError(f"package entry is not a regular file: {relative}")
        for relative in ["bin/dmm-server", "bin/dmm-nxm-handler"]:
            member = members[f"{root}/{relative}"]
            if member.mode & 0o111 == 0:
                raise RuntimeError(f"package binary is not executable: {relative}")

    def _validate_update_member(self, member, clean_name):
        if member.issym() or member.islnk():
            raise RuntimeError(f"package contains unsupported link entry: {member.name}")
        path = PurePosixPath(member.name)
        if path.is_absolute() or any(part in ("", ".", "..") for part in path.parts):
            raise RuntimeError(f"package contains unsafe path: {member.name}")
        if not clean_name:
            raise RuntimeError("package contains an empty path entry")
        if not (member.isfile() or member.isdir()):
            raise RuntimeError(f"package contains unsupported entry type: {member.name}")

    def _build_info(self):
        plugin_dir = Path(__file__).resolve().parent
        info_path = plugin_dir / "build-info.json"
        package_path = plugin_dir / "package.json"
        info = {"path": str(info_path)}
        try:
            data = json.loads(info_path.read_text(encoding="utf-8"))
            if isinstance(data, dict):
                for key in ["commit", "short_commit", "built_at", "channel"]:
                    value = data.get(key)
                    if isinstance(value, str) and value.strip():
                        info[key] = value.strip()
        except FileNotFoundError:
            info["error"] = "build-info.json is missing"
        except Exception as exc:
            info["error"] = self._redact_url(str(exc))

        try:
            package = json.loads(package_path.read_text(encoding="utf-8"))
            if isinstance(package, dict) and isinstance(package.get("version"), str):
                info["version"] = package["version"].strip()
        except Exception as exc:
            if "error" not in info:
                info["error"] = self._redact_url(str(exc))
        return info

    def _ensure_auth_token(self):
        token = self._read_auth_token()
        if token:
            return token
        self.log_dir.mkdir(parents=True, exist_ok=True)
        token = secrets.token_urlsafe(32)
        tmp_path = self.auth_token_file.with_suffix(".tmp")
        tmp_path.write_text(token + "\n", encoding="utf-8")
        tmp_path.chmod(0o600)
        tmp_path.replace(self.auth_token_file)
        try:
            self.auth_token_file.chmod(0o600)
        except Exception:
            pass
        self._log(f"generated backend auth token file={self.auth_token_file}")
        return token

    def _read_auth_token(self):
        env_token = os.environ.get("DMM_AUTH_TOKEN", "").strip()
        if env_token:
            return env_token
        try:
            token = self.auth_token_file.read_text(encoding="utf-8").strip()
            return token if len(token) >= 24 else ""
        except FileNotFoundError:
            return ""
        except Exception as exc:
            self._log(f"auth token read failed file={self.auth_token_file}: {self._redact_url(str(exc))}")
            return ""

    def _paired_phone_url(self, ip):
        if not ip:
            return None
        token = self._read_auth_token()
        if not token:
            return f"http://{ip}:17942"
        return f"http://{ip}:17942/#token={urllib.parse.quote(token)}"

    def _lan_ip(self):
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.connect(("8.8.8.8", 80))
            return sock.getsockname()[0]
        except Exception:
            return None
        finally:
            try:
                sock.close()
            except Exception:
                pass

    def _backend_responds(self):
        try:
            with socket.create_connection(("127.0.0.1", 17942), timeout=0.2) as sock:
                sock.settimeout(0.2)
                request = b"GET /api/health HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n"
                sock.sendall(request)
                response = sock.recv(128)
                return b" 200 " in response
        except Exception:
            return False

    def _backend_json(self, method, path, body=None):
        result, _ = self._backend_json_result(method, path, body)
        return result

    def _backend_json_result(self, method, path, body=None):
        try:
            headers = {
                "Accept": "application/json",
                "Content-Type": "application/json",
            }
            token = self._read_auth_token()
            if token:
                headers["X-DMM-Token"] = token
            request = urllib.request.Request(
                f"http://127.0.0.1:17942{path}",
                data=body,
                method=method,
                headers=headers,
            )
            with urllib.request.urlopen(request, timeout=2) as response:
                return json.loads(response.read().decode("utf-8")), None
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace").strip()
            detail = self._redact_url(detail)
            self._log(f"backend json request failed: {method} {path}: HTTP {exc.code}: {detail[:500]}")
            return None, f"Backend HTTP {exc.code}: {detail}" if detail else f"Backend HTTP {exc.code}"
        except Exception as exc:
            error = self._redact_url(str(exc))
            self._log(f"backend json request failed: {method} {path}: {error}")
            return None, error

    def _dependency(self, name, command, description, package_name, install_command, docs_url):
        path = shutil.which(command)
        return {
            "name": name,
            "command": command,
            "installed": path is not None,
            "path": path,
            "description": description,
            "package_name": package_name,
            "install_command": install_command,
            "install_hint": f"Install {package_name} from a trusted SteamOS package source if this command is missing.",
            "docs_url": docs_url,
        }

    def _nxm_status(self):
        desktop_path = Path.home() / ".local" / "share" / "applications" / self.desktop_id
        current = self._read_mimeapps_default("x-scheme-handler/nxm")
        protocol_current = self._read_mimeapps_default("x-scheme-handler/nxm-protocol")
        xdg_current = ""
        try:
            result = self._run_command(["xdg-mime", "query", "default", "x-scheme-handler/nxm"], check=False)
            xdg_current = result.stdout.strip()
            if not current:
                current = xdg_current
        except Exception as exc:
            self._log(f"nxm xdg-mime query failed: {exc}")
        return {
            "desktop_path": str(desktop_path),
            "desktop_exists": desktop_path.exists(),
            "current_handler": current,
            "protocol_handler": protocol_current,
            "xdg_handler": xdg_current,
            "registered": current == self.desktop_id and protocol_current == self.desktop_id and desktop_path.exists(),
        }

    def _write_mimeapps_default(self, scheme, desktop_id):
        config_dir = Path.home() / ".config"
        config_dir.mkdir(parents=True, exist_ok=True)
        path = config_dir / "mimeapps.list"
        parser = configparser.ConfigParser(interpolation=None, strict=False)
        parser.optionxform = str
        if path.exists():
            parser.read(path, encoding="utf-8")
        if "Default Applications" not in parser:
            parser["Default Applications"] = {}
        if "Added Associations" not in parser:
            parser["Added Associations"] = {}
        previous = parser["Default Applications"].get(scheme, "")
        parser["Default Applications"][scheme] = desktop_id
        parser["Added Associations"][scheme] = desktop_id
        with path.open("w", encoding="utf-8") as handle:
            parser.write(handle, space_around_delimiters=False)
        self._log(f"mimeapps associations updated path={path} scheme={scheme} previous_default={previous} next={desktop_id}")

    def _read_mimeapps_default(self, scheme):
        for path in [
            Path.home() / ".config" / "mimeapps.list",
            Path.home() / ".local" / "share" / "applications" / "mimeapps.list",
        ]:
            if not path.exists():
                continue
            parser = configparser.ConfigParser(interpolation=None, strict=False)
            parser.optionxform = str
            try:
                parser.read(path, encoding="utf-8")
            except Exception as exc:
                self._log(f"mimeapps read failed path={path}: {exc}")
                continue
            if parser.has_section("Default Applications") and parser.has_option("Default Applications", scheme):
                return parser.get("Default Applications", scheme).strip()
        return ""

    def _run_command(self, args, check):
        self._log(f"run command args={self._format_args(args)}")
        result = subprocess.run(
            args,
            check=check,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=self._clean_env(),
        )
        stdout = self._redact_url(result.stdout.strip())
        stderr = self._redact_url(result.stderr.strip())
        self._log(f"command result rc={result.returncode} stdout={stdout[:500]} stderr={stderr[:500]}")
        return result

    def _clean_env(self):
        env = os.environ.copy()
        for key in [
            "LD_LIBRARY_PATH",
            "LD_PRELOAD",
            "PYTHONPATH",
            "PYTHONHOME",
            "STEAM_COMPAT_CLIENT_INSTALL_PATH",
            "STEAM_COMPAT_DATA_PATH",
        ]:
            env.pop(key, None)
        env["PATH"] = "/usr/local/sbin:/usr/local/bin:/usr/bin:/bin"
        env.setdefault("HOME", str(Path.home()))
        env.setdefault("XDG_DATA_HOME", str(Path.home() / ".local" / "share"))
        env.setdefault("XDG_CONFIG_HOME", str(Path.home() / ".config"))
        return env

    def _format_args(self, args):
        out = []
        for item in args:
            text = str(item)
            text = self._redact_url(text)
            out.append(text)
        return out

    def _redact_url(self, text):
        return self.sensitive_query_pattern.sub(r"\1=[redacted]", str(text))

    def _tail_file(self, path, max_lines=40, max_bytes=24000):
        try:
            path = Path(path)
            if not path.exists():
                return ""
            size = path.stat().st_size
            with path.open("rb") as handle:
                if size > max_bytes:
                    handle.seek(size - max_bytes)
                    handle.readline()
                data = handle.read(max_bytes)
            text = data.decode("utf-8", errors="replace")
            lines = text.splitlines()[-max_lines:]
            return "\n".join(self._redact_url(line) for line in lines)
        except Exception as exc:
            return f"Unable to read {path}: {exc}"

    def _log(self, message):
        try:
            self.log_dir.mkdir(parents=True, exist_ok=True)
            now = datetime.datetime.now(datetime.timezone.utc).isoformat()
            with open(self.plugin_log, "a", encoding="utf-8") as handle:
                handle.write(f"{now} {message}\n")
        except Exception:
            pass
