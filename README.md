# Decky Mod Manager

Decky Mod Manager is a Steam Deck-first mod manager for Nexus/Vortex-compatible mods.

The MVP is a Decky plugin that starts a bundled Go backend and shows a URL for a phone or tablet browser. The backend owns mod-management logic; the Svelte + Vite web UI displays state, collects choices, calls REST APIs, and receives live updates over WebSocket.

## Current Status

MVP vertical slice in active testing:

- Decky plugin starts/stops a bundled Go backend and shows the phone/tablet URL.
- User-level `nxm://` registration captures Nexus Mod Manager Download links from the Deck browser.
- Svelte + Vite phone/tablet UI shows games, Action Center items, profile mods, profiles, advanced deployment tools, jobs, and settings.
- SQLite persistence covers games, profiles, jobs, captured installs, installed profile mods, downloads, checksums, and deployment manifests.
- Nexus API key configuration, URL parsing, download-link resolution, and archive download are implemented.
- Captured URL parsing goes through a catalog resolver boundary so future upstreams can plug in without changing the HTTP captured-install handlers; Nexus remains the only MVP download provider.
- Captured install actions and active download/extraction work can be canceled from the phone/tablet UI.
- Jobs persist structured source/game metadata for reliable game-scoped action and activity filtering after backend restarts.
- Captured Nexus links download immediately so short-lived URLs are consumed while valid; when auto-install is disabled, Action Center gates the local install from the cached archive.
- Decky Settings exposes "Auto-install captured downloads" and "Auto-enable installed mods"; auto-install defaults on, while auto-enable defaults off so newly installed mods remain disabled until the user enables them.
- The Decky plugin shows Gaming Mode notifications for Nexus captured-link and download job transitions while it is loaded.
- Stardew Valley (`413150`) is the first supported deploy target.
- Install planning uses Vortex-modeled metadata specs: the current Stardew slice handles manifest-based mods, root-folder `Content/` archives, and SMAPI installer archives with Linux embedded-payload extraction.
- Installer selection, mod type deployment roots, metadata extraction, and deployment eligibility are separate spec-owned concerns. Installed mod manifests preserve Vortex-style planner evidence plus manifest attributes such as logical file names, unique IDs, versions, content-pack targets, and dependencies.
- Repeated downloads/reinstalls of the same Nexus file are de-duplicated in the profile mod list.
- Profile mods can be removed from the Mods pane without deleting the cached download.
- Older developer-test records without install-plan target mappings are shown as `needs_recovery` and skipped by deployment; use Recover Downloads to reinstall supported archives with the current planner, or remove the affected row.
- ZIP extraction is handled in-process with path-traversal checks.
- Extensionless Nexus CDN archive paths are detected by file signature.
- Failed downloads that reached DMM-managed storage can be recovered from the game Mods pane.
- Installed mod display names come from install-plan metadata when the selected installer declares manifest-derived naming, otherwise DMM falls back to the Nexus archive name.
- Game diagnostics and the mobile Review tab surface extension-derived runtime requirements from enabled mod metadata. For example, Stardew SMAPI mods are reported separately from whether SMAPI itself is present to load them.
- The Review tab can also report missing required Stardew framework/dependency mods derived from installed manifest metadata.
- 7z/RAR extraction is supported through external helper tools.
- FOMOD archives pause as installer-choice actions; the phone/tablet UI and Decky modal flow can apply selected files through the normal profile install path.
- Deployment uses a Vortex-style staging/manifest model with symlink deployment, conflict detection, profile-aware keep/add/replace/remove planning, verification, repair, purge, and apply-time rollback for DMM-owned files.
- Profile-scoped mod priority can be changed from the Mods pane; lower priority numbers win duplicate target conflicts.
- Profile switching can deploy an empty profile to remove the previously deployed profile's DMM-owned links.
- Decky Debug can show recent plugin, backend, NXM handler, and Steam JS log tails.

See [GUIDELINES.md](GUIDELINES.md) for product and architecture decisions, [ROADMAP.md](ROADMAP.md) for MVP/post-MVP tracking, and [parity.md](parity.md) for Vortex parity gaps.

## Development

```bash
make test
make build
make web
make package
```

Run the backend:

```bash
./bin/dmm-server
```

Open:

```text
http://127.0.0.1:17942
```

## Steam Deck Dev Server

A temporary development bundle can run from:

```text
/home/deck/.local/share/decky-mod-manager-dev
```

Current Deck test URL:

```text
http://192.168.8.102:17942
```

The live Decky plugin directory is root-owned on the test Deck. Installing the packaged plugin into `/home/deck/homebrew/plugins` requires sudo/root interaction on the Deck.

## Decky Packaging

The packaged Decky plugin is built with the distribution layout documented by the Decky plugin template:

```text
decky-mod-manager/
  bin/dmm-server
  dist/index.js
  package.json
  plugin.json
  main.py
  web/dist/
```

`make package` creates two artifacts:

```text
dist/decky-mod-manager.zip
dist/decky-mod-manager.tar.gz
```

Use the ZIP for the normal Decky Developer install flow. The tarball is only for the SSH-based test installer in `testing/`, which lets us update the test Deck quickly without navigating the Decky UI for every build.

## Logs

Decky Loader plugin logs:

```text
/home/deck/homebrew/logs/decky-mod-manager/
```

Decky Mod Manager plugin/backend logs:

```text
/home/deck/.local/state/decky-mod-manager/plugin.log
/home/deck/.local/state/decky-mod-manager/backend.log
/home/deck/.local/state/decky-mod-manager/nxm-handler.log
```

The phone/tablet web UI writes redacted WebSocket lifecycle and domain-event diagnostics to `backend.log` as `web client event` entries.

Steam frontend JavaScript/plugin load errors:

```text
/home/deck/.local/share/Steam/logs/webhelper_js.txt
```

Backend app data:

```text
/home/deck/.local/share/decky-mod-manager/
```

## Decky Launch-Action Bridge

Some Steam client operations are only available from the Decky frontend JavaScript context, not from the Go backend. Setting Steam launch options for a specific app is one of those operations.

DMM handles this through an explicit action bridge:

1. Game extensions describe required primary launch tools through extension metadata. Stardew's extension can therefore request that Steam launch Stardew through SMAPI without putting Stardew-specific logic in the generic Decky code.
2. The Go backend evaluates installed/enabled mods and extension metadata, then exposes pending launch actions from `GET /api/launch/actions`.
3. The Decky Python bridge exposes `launch_actions` and `record_launch_action` methods to the Decky frontend.
4. The Decky frontend starts module-level background monitors when the plugin is loaded, outside the React panel component. Closing the Decky sidebar does not stop those monitors.
5. The monitor syncs pending launch actions at startup and after backend domain events, then calls `SteamClient.Apps.SetAppLaunchOptions(appid, desiredOptions)` when an action is available.
6. After the Steam API call, Decky reports the result back to the backend with `POST /api/games/{appID}/launch/configure`, so diagnostics and the phone/tablet UI can show whether the action is configured.

This means the Decky panel does not need to be open for launch-option actions to be applied. Decky Loader must still have the DMM plugin loaded, and the DMM backend must be running. If the Steam frontend API is unavailable, Decky records that failure and leaves the action pending for retry through the Steam client API. The Go backend does not patch Steam's `localconfig.vdf` as a product path.

## Firefox Flatpak Notes

When Firefox is installed as a Flatpak and launched from Gaming Mode, clipboard access and protocol handler dispatch can be blocked by the Flatpak sandbox or the desktop portal cache.

For clipboard support, override Firefox permissions once from Desktop Mode:

```sh
flatpak override --user --socket=wayland --socket=x11 --talk-name=org.freedesktop.portal.Desktop org.mozilla.firefox
```

If `nxm://` links still route to an old Vortex handler after registering Decky Mod Manager, clear Firefox's stale portal association:

```sh
flatpak permission-remove desktop-used-apps x-scheme-handler/nxm org.mozilla.firefox
systemctl --user restart xdg-desktop-portal xdg-desktop-portal-kde
```

## Tentative Test Game

Initial MVP target:

```text
Stardew Valley
Steam app ID: 413150
Nexus domain: stardewvalley
```

Avoid initial deployment tests on games where Vortex/manual mod state is detected.

## License

GPL-3.0-or-later.
