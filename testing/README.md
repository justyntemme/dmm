# Testing Scripts

## Build Artifacts

From the repo root:

```bash
make package
```

This creates:

- `dist/decky-mod-manager.zip`: Decky Developer install artifact.
- `dist/decky-mod-manager.tar.gz`: SSH test installer artifact.

The ZIP follows the Decky plugin distribution layout. The tarball exists only so the Deck-side test script can extract and replace the plugin directory quickly.

## Documented Decky Install Path

For manual testing through Decky itself, copy or host `dist/decky-mod-manager.zip`, then install it from the Decky Loader Developer settings. This is the path to use when validating packaging behavior.

## Install Decky Plugin

From the repo root on this machine:

```bash
./testing/install_decky_plugin.sh
```

Defaults:

- Deck host: `192.168.8.102`
- Deck user: `deck`
- Deck stage folder: `/home/deck/.local/share/decky-mod-manager-dev/plugin-test`
- Decky plugin folder: `/home/deck/homebrew/plugins/decky-mod-manager`

Override if needed:

```bash
DECK_HOST=192.168.8.102 DECK_USER=deck ./testing/install_decky_plugin.sh
```

If macOS/1Password SSH agent auth fails with `agent refused operation`, pass explicit SSH options without editing the script:

```bash
DECK_SSH_OPTS="-o IdentityAgent=none -o IdentitiesOnly=yes -i ~/.ssh/decky_mod_manager_test" ./testing/install_decky_plugin.sh
```

The dedicated test key is intentionally passwordless for unattended package transfers. Install or repair that public key on the Deck with:

```bash
ssh-copy-id -i ~/.ssh/decky_mod_manager_test.pub deck@192.168.8.102
```

If the Deck still rejects the key, restore SSH access on the Deck first, then rerun the same command. A quick auth check is:

```bash
ssh -i ~/.ssh/decky_mod_manager_test -o IdentityAgent=none -o BatchMode=yes -o IdentitiesOnly=yes deck@192.168.8.102 'echo ssh-ok'
```

If SSH is unavailable, create a manual transfer bundle instead:

```bash
make deck-transfer
```

Copy `dist/deck-transfer/` or `dist/decky-mod-manager-deck-transfer.tar.gz` to the Deck. If you copied the folder, run from the Deck:

```sh
mkdir -p ~/.testing
cp -a /path/to/deck-transfer/. ~/.testing/
```

If you copied the archive, run from the Deck:

```sh
mkdir -p ~/.testing
tar -xzf /path/to/decky-mod-manager-deck-transfer.tar.gz -C /tmp
cp -a /tmp/deck-transfer/. ~/.testing/
```

Then run:

```sh
chmod +x ~/.testing/*.sh
cd ~/.testing
sha256sum -c SHA256SUMS
./deck_package_smoke.sh
./deck_rehearsal.sh
./install_decky_plugin_from_package.sh
```

The script builds the Go backend, Svelte web UI, and Decky sidebar plugin, copies both package formats to the Deck, then installs only the `decky-mod-manager` plugin directory with sudo.

You may be prompted for the Steam Deck sudo/root password.

## Deck-Side Package Install

If the package and script are already in `~/.testing` on the Deck:

```bash
~/.testing/install_decky_plugin_from_package.sh
```

The Deck-side installer validates the package shape before touching the live plugin, stops an existing DMM backend/plugin process, backs up the current plugin directory under:

```text
/home/deck/.local/share/decky-mod-manager/backups/plugin-installs/
```

It then replaces `/home/deck/homebrew/plugins/decky-mod-manager` and reboots the Steam Deck. The reboot is intentional for this test installer because Gaming Mode sometimes keeps stale Decky frontend state after only restarting Decky Plugin Loader. DMM app data, logs, downloads, staging, and SQLite state under `/home/deck/.local/share/decky-mod-manager` and `/home/deck/.local/state/decky-mod-manager` are not removed by plugin installation.

Before rebooting, the installer runs `live_installed_package_check.sh` when that script is available. If the installed plugin does not match the package, it restores the backup and exits with an error instead of leaving a stale or partial plugin install.

To install the ZIP instead of the tarball:

```bash
PACKAGE=~/.testing/decky-mod-manager.zip ~/.testing/install_decky_plugin_from_package.sh
```

## Temporary Passwordless Test Installs

For unattended live testing, do not grant passwordless sudo to a writable script
inside `~/.testing`. A script in that directory could be overwritten and then run
as root.

The transfer bundle includes a safer temporary setup:

```bash
cd ~/.testing
./install_decky_testing_sudoers.sh
```

That command installs a root-owned wrapper and package installer under:

```text
/opt/decky-mod-manager-testing/bin/
```

It also installs a narrow sudoers drop-in that allows the `deck` user to run only
that wrapper and a few fixed Decky/log/reboot commands without a password. The
wrapper refuses packages outside `~/.testing`, verifies the root-owned installer
copy, checks the package mode, and then runs the normal installer. The package
itself remains owned by `deck` so development builds can still be replaced by
SCP without needing a sudo prompt first.

After setup, unattended installs use:

```bash
sudo /opt/decky-mod-manager-testing/bin/decky-mod-manager-test-install
```

To remove the temporary rule and wrapper:

```bash
sudo ~/.testing/install_decky_testing_sudoers.sh --remove
```

## Local Backend Smoke Check

Before copying to the Deck, create a local MVP release candidate on this machine:

```bash
make mvp-release
```

It runs Go tests, web and Decky builds, Decky Python syntax, testing script syntax, the MVP UI product audit, local backend smoke, package creation, local package shape validation, whitespace checks, and then creates the Deck transfer bundle.

The UI product audit keeps dependency/server controls in Decky, keeps Nexus import inside a selected game workspace, and keeps file-level deployment tools behind the advanced disclosure. It is a source-tree check and is intentionally not included in the Deck transfer bundle.

For audit only, run:

```bash
make mvp-audit
```

For a faster backend-only check, run:

```bash
./testing/local_smoke.sh
```

It builds the native backend, starts it with isolated temporary XDG config/data directories, and verifies health, status, jobs, dependencies, and the bundled web index. Set `KEEP_TMP=1` to keep the temporary directory for log inspection, or `PORT=17960` if the default smoke port is busy.

This does not replace the Steam Deck validation below because it does not exercise Decky, Gaming Mode, Linux AMD64 packaging, `nxm://` desktop integration, or the actual Stardew install.

## Steam Deck Smoke Checks

After copying a package to `~/.testing`, verify the tarball can start outside Decky without touching the installed plugin:

```sh
~/.testing/deck_package_smoke.sh
```

Expected health response inside the script:

```json
{"ok":true,"version":"dev"}
```

Set `KEEP_TMP=1` to inspect the temporary extracted package, isolated config, isolated data directory, and server log.

Package smoke also checks that the built Svelte and Decky bundles contain the MVP profile-first UI and Decky-owned server/dependency controls, so a stale or mismatched package fails before replacing the live plugin.

On a non-Linux development machine, use shape-only mode to verify the tarball contents without trying to execute the Steam Deck Linux binary:

```bash
PACKAGE=dist/decky-mod-manager.tar.gz SHAPE_ONLY=1 ./testing/deck_package_smoke.sh
```

To rehearse deployment safely against copied live DMM/Stardew data, never point this check at the real game folder. Use the Deck-side rehearsal script:

```sh
~/.testing/deck_rehearsal.sh
```

It copies the DMM data directory and Stardew install to `/tmp`, rewrites the copied SQLite game path, starts the packaged backend on a spare port, runs recovery, diagnostics, preview, deploy, and purge, then verifies copied-game symlinks were created and removed.

Set `KEEP_TMP=1` to inspect the copied data and server log after the run.

For Stardew test data, the rehearsal should mark legacy rows as `needs_recovery` when needed, recover supported archives, record unsupported archives as blocked candidates, preview with zero conflicts, deploy the expected DMM-managed links for the copied profile, and purge the same managed links. The exact link count changes as new test mods are added.

When validating a specific Vortex metadata ingestion case from cached Nexus downloads, pass expected source mod IDs and optional planner IDs:

```sh
EXPECT_STAGED_MODS='2400=vortex:stardewvalley:smapi-installer,8897=vortex:stardewvalley:stardew-valley-installer' ~/.testing/deck_rehearsal.sh
```

To also assert Vortex/SMAPI manifest metadata, pass source mod IDs with one or more expected manifest unique IDs:

```sh
EXPECT_STAGED_MODS='2400=vortex:stardewvalley:smapi-installer,8897=vortex:stardewvalley:stardew-valley-installer' \
EXPECT_STAGED_METADATA='2400=SMAPI.ConsoleCommands;SMAPI.SaveBackup,8897=shekurika.WaterFish' \
~/.testing/deck_rehearsal.sh
```

These checks validate both the staged manifest metadata in SQLite and the public `/api/games/{appID}/mods` response used by the phone/tablet UI. They are optional so the rehearsal still works on Decks that do not have those exact cached archives.

To also run the live file visibility checker and profile enable/disable verifier against the copied game while the rehearsal backend is still running:

```sh
RUN_FILE_VISIBILITY_CHECK=1 RUN_PROFILE_TOGGLE_CHECK=1 REQUIRE_SMAPI_ROOT=1 \
EXPECT_STAGED_MODS='2400=vortex:stardewvalley:smapi-installer,8897=vortex:stardewvalley:stardew-valley-installer' \
EXPECT_STAGED_METADATA='2400=SMAPI.ConsoleCommands;SMAPI.SaveBackup,8897=shekurika.WaterFish' \
~/.testing/deck_rehearsal.sh
```

This keeps the real Stardew folder untouched while proving DMM-managed SMAPI root symlinks and SMAPI mod manifest symlinks are visible in the copied game. With `RUN_PROFILE_TOGGLE_CHECK=1`, it also disables one enabled profile mod, applies the copied profile, verifies that mod's unique managed files are removed, re-enables it, reapplies the profile, and verifies the links point back into copied DMM storage.

## MVP Live Validation

The MVP is not considered verified until the installed Decky plugin passes this full Gaming Mode path:

1. Install the latest package with `~/.testing/install_decky_plugin_from_package.sh`.
2. Open Decky Mod Manager in Gaming Mode and start the server.
3. Confirm the Decky panel shows the phone/tablet URL and NXM handler registered.
4. If existing developer-test rows show `needs_recovery`, use Recover Downloads for supported archives or Remove for rows that should not be managed.
5. Open Nexus from the Deck, click a fresh Stardew Valley Mod Manager Download link, and confirm a captured install appears in the phone/tablet Action Center when user action is required.
6. Install from the cached captured action in the phone/tablet UI when auto-install is disabled or installer choices are needed.
7. Confirm the Stardew Mods tab shows exactly one profile mod row for the installed mod/file.
8. Preview deployment and confirm zero conflicts.
9. Deploy and confirm the job completes.
10. Launch Stardew/SMAPI enough to confirm the deployed mod is visible or loaded.
11. Purge deployed files and confirm the manifest-owned links are removed.
12. Redeploy and confirm the same preview/deploy path still works.

After installing the package and starting the server, collect the current live state with:

```sh
~/.testing/live_post_install_check.sh
```

This runs the installed-package verifier, backend health check, web UI asset check, backend status snapshot, MVP live acceptance check, and Stardew file visibility check in order. Use it as the first command after installing a package and starting the server from Decky.

The web UI asset check verifies both reachability and key profile-first UI strings from the served JavaScript bundle.

After capturing at least one fresh Nexus link with the current package, the MVP live check can also require structured job metadata:

```sh
REQUIRE_JOB_PAYLOAD=1 ~/.testing/mvp_live_check.sh
```

The automatic install verifier always checks that the fresh captured install includes app/catalog/domain/mod/file identifiers in the job payload.

For narrower debugging, run the checks individually.

```sh
~/.testing/live_status.sh
```

The script prints health/status, games, a single game diagnostics summary, jobs, profile mods, install candidates, deployment status, deployment preview, and recent DMM logs. Override `APP_ID`, `HOST`, `PORT`, or `LOG_LINES` when testing another game or server instance.

To verify the installed Decky plugin exactly matches the staged package in `~/.testing`, run:

```sh
~/.testing/live_installed_package_check.sh
```

This compares packaged files such as `main.py`, the Decky frontend bundle, the web UI bundle, and the Go binaries against `/home/deck/homebrew/plugins/decky-mod-manager`. It does not restart Decky or touch DMM app data.

To verify the running backend can serve the phone/tablet Svelte UI and every hashed asset referenced by `index.html`, run:

```sh
~/.testing/live_web_asset_check.sh
```

After installing and deploying a fresh test mod, run the stricter live acceptance check:

```sh
~/.testing/mvp_live_check.sh
```

It fails if the selected game has no installed/enabled profile mods, still has active install/deploy work, has no active deployment manifest, has no deployed files, cannot build a deployment preview, or reports preview conflicts. Override `APP_ID`, `HOST`, `PORT`, `REQUIRE_DEPLOYED=0`, or `ALLOW_WARNINGS=0` for narrower debugging runs.

The check also prints handler-derived runtime requirements from diagnostics. For Stardew, deployed SMAPI mods can be present in the game folder while still having no in-game effect if SMAPI is not installed or the game is not launched through it. Use `ALLOW_WARNINGS=0` when you want missing runtime requirements to fail the scripted check.

To verify the core profile enable/disable workflow against live DMM-managed files:

```sh
~/.testing/live_profile_toggle_check.sh
```

The script picks one enabled profile mod with unique deployment targets, disables it through the same profile-mod API used by the web UI, applies the profile, verifies its DMM-managed files are removed from the live game folder, re-enables it, reapplies the profile, and verifies the restored symlinks point back into DMM storage. It attempts to restore the mod if the check fails before completion.

To verify profile copy/remove behavior without changing the active deployment, run:

```sh
~/.testing/live_profile_transfer_check.sh
```

The script creates or reuses a secondary test profile, copies one active-profile mod into it disabled, removes that copied membership, and verifies the source profile membership plus staged files remain intact.

Before launching Stardew, verify that DMM-managed SMAPI mod files are actually visible in the game folder:

```sh
~/.testing/live_stardew_mod_files_check.sh
```

This checks the active deployment manifest, the Stardew `Mods/` directory, and visible `manifest.json` symlinks. It fails if enabled profile mods are not represented by DMM-managed symlinked manifests under the game folder. This does not replace launching Stardew/SMAPI, but it proves the file-system side of the in-game visibility path.

The script prints handler-derived runtime requirements when the installed backend supports them. Set `REQUIRE_RUNTIME=1` to fail when SMAPI or another required runtime is missing.

After installing and deploying SMAPI through DMM, make the file-system check strict for SMAPI's game-root payload too:

```sh
REQUIRE_SMAPI_ROOT=1 REQUIRE_RUNTIME=1 ~/.testing/live_stardew_mod_files_check.sh
```

`REQUIRE_SMAPI_ROOT=1` verifies DMM-managed symlinked root markers such as `StardewModdingAPI`, `StardewModdingAPI.dll`, `StardewModdingAPI.deps.json`, and `smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll`.

## Auto-Install Live Check

After installing the latest package and starting the server from Decky, use this script to verify the `Auto-install captured downloads` setting with a real Nexus capture:

```sh
~/.testing/live_auto_install_check.sh
```

The script records the current install settings, enables automatic install for captured downloads, and waits for a new Stardew `nxm://` captured install. While it is waiting, click a fresh Nexus Mod Manager Download link from the Deck browser. The check passes only if the captured install moves past the manual local-install gate and completes through download/install. By default it restores the previous auto-install setting at the end; set `RESTORE_SETTING=0` to leave the setting enabled after the test.
