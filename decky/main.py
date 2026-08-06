import asyncio
import configparser
import datetime
import json
import os
import re
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
        self._log("plugin unloading")
        await self.stop_server()

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
            self._log("backend already reachable on localhost")
            return await self.status()

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
        self._log(f"backend started pid={self.backend_process.pid}")
        for _ in range(10):
            if self._backend_responds():
                break
            await asyncio.sleep(0.25)
        return await self.status()

    async def stop_server(self):
        self._log("stop_server requested")
        if self.backend_process and self.backend_process.poll() is None:
            os.killpg(os.getpgid(self.backend_process.pid), signal.SIGTERM)
            try:
                self.backend_process.wait(timeout=5)
                self._log("backend stopped with SIGTERM")
            except subprocess.TimeoutExpired:
                os.killpg(os.getpgid(self.backend_process.pid), signal.SIGKILL)
                self._log("backend killed after timeout")
        self.backend_process = None
        return await self.status()

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
        result = {
            "running": running,
            "ip": ip,
            "port": 17942,
            "url": f"http://{ip}:17942" if ip else None,
            "pid": pid,
            "backend": backend_status,
            "logs": {
                "plugin": str(self.plugin_log),
                "backend": str(self.backend_log),
            },
            "build": self._build_info(),
            "error": error,
            "warning": "No app authentication is enabled. Keep LAN-only mode enabled unless using a trusted VPN/tunnel.",
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

    async def add_captured_install(self, url, app_id=""):
        url = str(url or "").strip()
        app_id = str(app_id or "").strip()
        self._log(f"add captured install requested app_id={app_id} url={self._redact_url(url)}")
        if not url:
            return {"ok": False, "error": "Mod URL is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"url": url, "steam_app_id": app_id, "source": "decky-plugin"}).encode("utf-8")
        result, error = self._backend_json_result("POST", "/api/captured-installs", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to capture mod link."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"add captured install accepted job={job}")
        return {"ok": True, "result": result}

    async def jobs(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "jobs": []}
        result, error = self._backend_json_result("GET", "/api/jobs")
        if result is None:
            return {"ok": False, "error": error or "Unable to load jobs.", "jobs": []}
        if isinstance(result, list):
            return {"ok": True, "jobs": result}
        return {"ok": False, "error": "Unexpected jobs response.", "jobs": []}

    async def games(self):
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "games": []}
        result, error = self._backend_json_result("GET", "/api/games")
        if result is None:
            return {"ok": False, "error": error or "Unable to load games.", "games": []}
        if isinstance(result, list):
            return {"ok": True, "games": result}
        return {"ok": False, "error": "Unexpected games response.", "games": []}

    async def nexus_mods(self, app_id, query="", sort="downloads", time_window="all", count=20, offset=0, vortex_only=True):
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
        self._log(f"nexus mods searched app_id={app_id} query_present={bool(str(query or '').strip())} sort={sort} time_window={time_window} count={len(mods)}")
        return {"ok": True, "mods": mods, "total_count": int(result.get("total_count") or len(mods))}

    async def nexus_mod_files(self, app_id, mod_id):
        app_id = str(app_id or "").strip()
        mod_id = str(mod_id or "").strip()
        if not app_id or not mod_id:
            return {"ok": False, "error": "app_id and mod_id are required.", "files": []}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running.", "files": []}
        path = f"/api/games/{urllib.parse.quote(app_id)}/nexus/mods/{urllib.parse.quote(mod_id)}/files"
        result, error = self._backend_json_result("GET", path)
        if result is None:
            return {"ok": False, "error": error or "Unable to load Nexus files.", "files": []}
        files = result.get("files") if isinstance(result, dict) else None
        if not isinstance(files, list):
            return {"ok": False, "error": "Unexpected Nexus files response.", "files": []}
        self._log(f"nexus mod files loaded app_id={app_id} mod_id={mod_id} count={len(files)}")
        return {"ok": True, "files": files}

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

    async def restore_game_deployment(self, app_id):
        app_id = str(app_id or "").strip()
        if not app_id:
            return {"ok": False, "error": "app_id is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/deploy/restore", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to restore the last DMM-applied state."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"deployment restore requested app_id={app_id} job_id={(job or {}).get('id', '') if isinstance(job, dict) else ''}")
        return {
            "ok": True,
            "job": job,
            "result": result.get("result") if isinstance(result, dict) else None,
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

    async def apply_install_candidate(self, app_id, candidate_id, selections):
        app_id = str(app_id or "").strip()
        candidate_id = str(candidate_id or "").strip()
        if not app_id or not candidate_id:
            return {"ok": False, "error": "app_id and candidate_id are required."}
        if not isinstance(selections, dict):
            selections = {}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"selections": selections}).encode("utf-8")
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/install-candidates/{urllib.parse.quote(candidate_id)}/apply", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to apply installer choices."}
        self._log(f"installer choices applied app_id={app_id} candidate_id={candidate_id}")
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
        payload = json.dumps({"enabled": bool(enabled)}).encode("utf-8")
        result, error = self._backend_json_result("PUT", f"/api/profiles/{urllib.parse.quote(profile_id)}/mods/{urllib.parse.quote(installed_mod_id)}", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to update mod."}
        self._log(f"profile mod updated app_id={app_id} profile_id={profile_id} installed_mod_id={installed_mod_id} enabled={bool(enabled)}")
        return {"ok": True, "mod": result.get("mod"), "apply": result.get("apply")}

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

    async def reinstall_game_mod(self, app_id, installed_mod_id):
        app_id = str(app_id or "").strip()
        installed_mod_id = str(installed_mod_id or "").strip()
        if not app_id or not installed_mod_id:
            return {"ok": False, "error": "app_id and installed_mod_id are required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        result, error = self._backend_json_result("POST", f"/api/games/{urllib.parse.quote(app_id)}/mods/{urllib.parse.quote(installed_mod_id)}/reinstall", b"{}")
        if result is None:
            return {"ok": False, "error": error or "Unable to reinstall mod."}
        self._log(f"mod reinstalled app_id={app_id} installed_mod_id={installed_mod_id}")
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
            self._dependency("7-Zip", "7z", "Extracts .7z and many Nexus archive formats."),
            self._dependency("bsdtar", "bsdtar", "Extracts tar and zip archives."),
            self._dependency("unzip", "unzip", "Extracts .zip archives."),
            self._dependency("unrar", "unrar", "Extracts .rar archives when available."),
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
        release = os.environ.get("DMM_UPDATE_RELEASE", "dev-latest").strip() or "dev-latest"
        package_url = os.environ.get("DMM_UPDATE_PACKAGE_URL", "").strip()
        if not package_url:
            package_url = f"https://github.com/{repo}/releases/download/{release}/{self.update_package_name}"
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
                "message": "Latest package downloaded. The installer started and will reboot the Deck if installation succeeds.",
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
            request = urllib.request.Request(
                f"http://127.0.0.1:17942{path}",
                data=body,
                method=method,
                headers={
                    "Accept": "application/json",
                    "Content-Type": "application/json",
                },
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

    def _dependency(self, name, command, description):
        path = shutil.which(command)
        return {
            "name": name,
            "command": command,
            "installed": path is not None,
            "path": path,
            "description": description,
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
