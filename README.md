# Decky Mod Manager

Decky Mod Manager is a Steam Deck-first mod manager for Nexus/Vortex-compatible mods.

The MVP is a Decky plugin that starts a bundled Go backend and shows a URL for a phone or tablet browser. The backend owns mod-management logic; the Svelte + Vite web UI displays state, collects choices, and calls REST/SSE APIs.

## Current Status

Early scaffold:

- Go backend health/status APIs
- LAN-only request middleware
- Steam library/game discovery
- Archive dependency detection
- Job/SSE skeleton
- Nexus URL parser skeleton
- Svelte + Vite management UI shell
- Decky plugin wrapper skeleton
- SQLite persistence for discovered games and default profiles
- Archive inspection safety checks
- Deployment plan/apply/purge primitives tested in temp directories
- Nexus REST client skeleton

See [GUIDELINES.md](GUIDELINES.md) for product and architecture decisions.

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
```

Steam frontend JavaScript/plugin load errors:

```text
/home/deck/.local/share/Steam/logs/webhelper_js.txt
```

Backend app data:

```text
/home/deck/.local/share/decky-mod-manager/
```

## Tentative Test Game

Initial clean candidate:

```text
METAL GEAR SOLID V: THE PHANTOM PAIN
Steam app ID: 287700
Nexus domain: metalgearsolidvtpp
```

Avoid initial deployment tests on games where Vortex/manual mod state is detected.

## License

GPL-3.0-or-later.
