# Blasphemous Extension Notes

## Identity

- Steam AppID: `774361`
- DMM extension ID: `blasphemous`
- Nexus domain: `blasphemous`

## Verified Sources

- Nexus API game list verified the `blasphemous` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no game-specific Blasphemous handler was found.
- Vortex shared BepInEx extension source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex`
- Representative Nexus install instructions identify BepInEx 5.4.x as the required runtime and place plugin DLLs under `BepInEx/plugins`.
- BepInEx native Unix Steam launch documentation requires `run_bepinex.sh` to be executable and launched through Steam.

## Current DMM Capability

- DMM detects native Linux installs through `Blasphemous.x86_64` and `Blasphemous_Data/globalgamemanagers`, and Windows/Proton installs through `Blasphemous.exe`.
- DMM supports Vortex-compatible BepInEx runtime packages, BepInEx root/config packages, ConfigurationManager archives, and loose BepInEx plugin DLL archives.
- Native Linux runtime packages preserve executable mode on `run_bepinex.sh`.
- Enabled BepInEx plugin mods require the extension-owned BepInEx launch tool so Steam launches through `run_bepinex.sh`.
- Unknown archive layouts remain blocked by an extension-owned catch-all until a specific install rule is verified.

## Beta Gaps

- Live-test a Blasphemous BepInEx runtime package and at least one plugin archive on the Steam Deck.
- Confirm Decky/Steam launch-option application launches Blasphemous through `run_bepinex.sh`.
- Add non-BepInEx install rules only after source or representative archive review proves target roots and rollback behavior.
