# Decky Mod Manager

Decky Mod Manager is a Steam Deck-first mod manager for Nexus/Vortex-compatible mods.

The MVP is a Decky plugin that starts a bundled Go backend and shows a URL for a phone or tablet browser. The backend owns mod-management logic; the Svelte + Vite web UI displays state, collects choices, and calls REST/SSE APIs.

## Current Status

MVP vertical slice in active testing:

- Decky plugin starts/stops a bundled Go backend and shows the phone/tablet URL.
- User-level `nxm://` registration captures Nexus Mod Manager Download links from the Deck browser.
- Svelte + Vite phone/tablet UI shows games, install requests, staged mods, profiles, deploy preview, jobs, and settings.
- SQLite persistence covers games, profiles, jobs, pending imports, staged mods, downloads, checksums, and deployment manifests.
- Nexus API key configuration, URL parsing, download-link resolution, and archive download are implemented.
- Import URL parsing goes through a catalog resolver boundary so future upstreams can plug in without changing the HTTP import handlers; Nexus remains the only MVP download provider.
- Pending install requests and active pending-import downloads/extractions can be canceled from the phone/tablet UI.
- Jobs persist structured source/game metadata for reliable game-scoped request and activity filtering after backend restarts.
- Download approval is required by default. The Decky plugin Settings tab exposes "Auto-accept download requests" for faster Deck-only flows.
- The Decky plugin shows Gaming Mode notifications for Nexus install request and download job transitions while it is loaded.
- Stardew Valley (`413150`) is the first supported deploy target.
- Install planning uses Vortex-modeled metadata specs: the current Stardew slice handles manifest-based mods, root-folder `Content/` archives, and SMAPI installer archives with Linux embedded-payload extraction.
- Installer selection, mod type deployment roots, metadata extraction, and deployment eligibility are separate spec-owned concerns. Staged manifests preserve Vortex-style planner evidence plus manifest attributes such as logical file names, unique IDs, versions, content-pack targets, and dependencies.
- Repeated downloads/restaging of the same Nexus file are de-duplicated in the staged plugin list.
- Staged mods can be removed from the Plugins pane without deleting the cached download.
- Older raw-staged records without install-plan target mappings are shown as `needs_recovery` and skipped by deployment; use Recover Downloads to restage supported archives with the current planner, or Remove the staged row.
- ZIP extraction is handled in-process with path-traversal checks.
- Extensionless Nexus CDN archive paths are detected by file signature.
- Failed downloads that reached DMM-managed storage can be recovered from the game Plugins pane.
- Staged mod display names come from install-plan metadata when the selected installer declares manifest-derived naming, otherwise DMM falls back to the Nexus archive name.
- Game diagnostics and the mobile Review tab surface handler-derived runtime requirements from enabled mod metadata. For example, staged mods with a Stardew SMAPI mod type are reported separately from whether SMAPI itself is present to load them.
- The Review tab can also report missing required Stardew framework/dependency mods derived from staged manifest metadata.
- 7z/RAR extraction is supported through external helper tools.
- FOMOD archives are detected and currently fail with a clear unsupported-installer message; interactive FOMOD choice UI is still pending.
- Deployment uses a Vortex-style staging/manifest model with symlink deployment, conflict detection, profile-aware keep/add/replace/remove planning, verification, repair, purge, and apply-time rollback for DMM-owned files.
- Profile-scoped mod priority can be changed from the Plugins pane; lower priority numbers win duplicate target conflicts.
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

Steam frontend JavaScript/plugin load errors:

```text
/home/deck/.local/share/Steam/logs/webhelper_js.txt
```

Backend app data:

```text
/home/deck/.local/share/decky-mod-manager/
```

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
