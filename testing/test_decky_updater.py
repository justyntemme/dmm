import hashlib
import importlib.util
import io
import tarfile
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("dmm_decky_main", ROOT / "decky" / "main.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)
Plugin = MODULE.Plugin


class DeckyUpdaterTest(unittest.TestCase):
    def setUp(self):
        self.plugin = Plugin()
        self.plugin._log = lambda _message: None

    def test_release_asset_url_is_pinned(self):
        valid = "https://github.com/justyntemme/dmm/releases/download/v0.0.2/decky-mod-manager.tar.gz"
        self.plugin._validate_update_asset_url(valid, "v0.0.2", self.plugin.update_package_name)
        for invalid in (
            "http://github.com/justyntemme/dmm/releases/download/v0.0.2/decky-mod-manager.tar.gz",
            "https://example.com/justyntemme/dmm/releases/download/v0.0.2/decky-mod-manager.tar.gz",
            "https://github.com/other/dmm/releases/download/v0.0.2/decky-mod-manager.tar.gz",
            valid + "?token=unsafe",
        ):
            with self.subTest(invalid=invalid), self.assertRaises(RuntimeError):
                self.plugin._validate_update_asset_url(invalid, "v0.0.2", self.plugin.update_package_name)

    def test_download_accepts_matching_release_digest(self):
        with tempfile.TemporaryDirectory() as raw_dir:
            directory = Path(raw_dir)
            package_bytes = self._package_bytes(directory)
            digest = hashlib.sha256(package_bytes).hexdigest()
            self._install_fake_downloader(package_bytes, f"{digest}  {self.plugin.update_package_name}\n".encode())
            paths = self._update_paths(directory)
            downloaded = self.plugin._download_update_package(*paths)
            self.assertEqual(downloaded, len(package_bytes))
            self.assertEqual(paths[-1].read_bytes(), package_bytes)
            self.assertFalse(paths[-2].exists())

    def test_download_rejects_mismatched_release_digest(self):
        with tempfile.TemporaryDirectory() as raw_dir:
            directory = Path(raw_dir)
            package_bytes = self._package_bytes(directory)
            self._install_fake_downloader(package_bytes, ("0" * 64 + f"  {self.plugin.update_package_name}\n").encode())
            paths = self._update_paths(directory)
            with self.assertRaisesRegex(RuntimeError, "SHA-256 mismatch"):
                self.plugin._download_update_package(*paths)
            self.assertFalse(paths[-1].exists())
            self.assertFalse(paths[-3].exists())

    def test_checksum_manifest_rejects_duplicate_package_entries(self):
        with tempfile.TemporaryDirectory() as raw_dir:
            checksum_path = Path(raw_dir) / self.plugin.update_checksum_name
            checksum_path.write_text(
                f"{'1' * 64}  {self.plugin.update_package_name}\n"
                f"{'2' * 64}  {self.plugin.update_package_name}\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(RuntimeError, "exactly one digest"):
                self.plugin._read_update_digest(checksum_path)

    def _update_paths(self, directory):
        release = "v0.0.2"
        return (
            self.plugin._expected_release_asset_url(release, self.plugin.update_package_name),
            self.plugin._expected_release_asset_url(release, self.plugin.update_checksum_name),
            release,
            directory / ".package.download",
            directory / ".checksums.download",
            directory / self.plugin.update_package_name,
        )

    def _install_fake_downloader(self, package_bytes, checksum_bytes):
        def download(url, target, _max_bytes):
            body = checksum_bytes if url.endswith(self.plugin.update_checksum_name) else package_bytes
            target.write_bytes(body)
            return len(body)

        self.plugin._download_file = download

    def _package_bytes(self, directory):
        package_path = directory / "fixture.tar.gz"
        with tarfile.open(package_path, "w:gz") as archive:
            for relative in self.plugin.update_required_files:
                body = b"fixture"
                member = tarfile.TarInfo(f"decky-mod-manager/{relative}")
                member.size = len(body)
                member.mode = 0o755 if relative.startswith("bin/") else 0o644
                archive.addfile(member, io.BytesIO(body))
        return package_path.read_bytes()


if __name__ == "__main__":
    unittest.main()
