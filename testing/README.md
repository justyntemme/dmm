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

The script builds the Go backend, Svelte web UI, and Decky sidebar plugin, copies both package formats to the Deck, then installs only the `decky-mod-manager` plugin directory with sudo.

You may be prompted for the Steam Deck sudo/root password.

## Deck-Side Package Install

If the package and script are already in `~/.testing` on the Deck:

```bash
~/.testing/install_decky_plugin_from_package.sh
```

To install the ZIP instead of the tarball:

```bash
PACKAGE=~/.testing/decky-mod-manager.zip ~/.testing/install_decky_plugin_from_package.sh
```
