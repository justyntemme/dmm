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
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


class Plugin:
    backend_process = None
    log_dir = Path(os.environ.get("XDG_STATE_HOME", "/home/deck/.local/state")) / "decky-mod-manager"
    plugin_log = log_dir / "plugin.log"
    backend_log = log_dir / "backend.log"
    desktop_id = "decky-mod-manager-nxm.desktop"
    nxm_schemes = ["x-scheme-handler/nxm", "x-scheme-handler/nxm-protocol"]
    sensitive_query_pattern = re.compile(r"(?i)(key|expires|md5)=([^&\"'\s]+)")

    async def _main(self):
        self._log("plugin loaded")

    async def _unload(self):
        self._log("plugin unloading")
        await self.stop_server()

    async def start_server(self):
        self._log("start_server requested")
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
            "error": error,
            "warning": "No app authentication is enabled. Keep LAN-only mode enabled unless using a trusted VPN/tunnel.",
        }
        return result

    async def open_nexus(self, game_domain=None):
        url = "https://www.nexusmods.com"
        if game_domain:
            url = f"https://www.nexusmods.com/{game_domain}"
        self._log(f"open nexus navigation requested: {url}")
        return {"ok": True, "url": url}

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

    async def add_pending_import(self, url):
        url = str(url or "").strip()
        self._log(f"add pending import requested url={self._redact_url(url)}")
        if not url:
            return {"ok": False, "error": "Nexus URL is required."}
        if not self._backend_responds():
            return {"ok": False, "error": "Server is not running."}
        payload = json.dumps({"url": url, "source": "decky-plugin"}).encode("utf-8")
        result, error = self._backend_json_result("POST", "/api/imports/pending", payload)
        if result is None:
            return {"ok": False, "error": error or "Unable to add install request."}
        job = result.get("job") if isinstance(result, dict) else None
        self._log(f"add pending import accepted job={job}")
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
            self._dependency("bsdtar", "bsdtar", "Extracts tar/zip archives and is useful as a fallback."),
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

    async def diagnostics(self):
        self._log("diagnostics requested")
        nxm_log = self.log_dir / "nxm-handler.log"
        steam_js = Path.home() / ".local" / "share" / "Steam" / "logs" / "webhelper_js.txt"
        return {
            "status": await self.status(),
            "nxm": self._nxm_status(),
            "dependencies": await self.dependencies(),
            "logs": {
                "plugin": {
                    "path": str(self.plugin_log),
                    "tail": self._tail_file(self.plugin_log),
                },
                "backend": {
                    "path": str(self.backend_log),
                    "tail": self._tail_file(self.backend_log),
                },
                "nxm": {
                    "path": str(nxm_log),
                    "tail": self._tail_file(nxm_log),
                },
                "steam_js": {
                    "path": str(steam_js),
                    "tail": self._tail_file(steam_js),
                },
            },
        }

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
