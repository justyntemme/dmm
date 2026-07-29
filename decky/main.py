import asyncio
import datetime
import json
import os
import signal
import shutil
import socket
import subprocess
import urllib.error
import urllib.request
from pathlib import Path


class Plugin:
    backend_process = None
    log_dir = Path(os.environ.get("XDG_STATE_HOME", "/home/deck/.local/state")) / "decky-mod-manager"
    plugin_log = log_dir / "plugin.log"
    backend_log = log_dir / "backend.log"

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
        result = self._backend_json("PUT", "/api/settings/security", payload)
        if result is None:
            return {
                "ok": False,
                "error": "Unable to update server settings.",
            }
        return {
            "ok": True,
            "status": result,
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
                return json.loads(response.read().decode("utf-8"))
        except Exception as exc:
            self._log(f"backend json request failed: {method} {path}: {exc}")
            return None

    def _dependency(self, name, command, description):
        path = shutil.which(command)
        return {
            "name": name,
            "command": command,
            "installed": path is not None,
            "path": path,
            "description": description,
        }

    def _log(self, message):
        try:
            self.log_dir.mkdir(parents=True, exist_ok=True)
            now = datetime.datetime.now(datetime.timezone.utc).isoformat()
            with open(self.plugin_log, "a", encoding="utf-8") as handle:
                handle.write(f"{now} {message}\n")
        except Exception:
            pass
