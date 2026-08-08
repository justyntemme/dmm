# Potion Craft: Alchemist Simulator Extension Notes

## Identity

- Steam AppID: `1210320`
- DMM extension ID: `potioncraft`
- Nexus domain: `potioncraftalchemistsimulator`

## Verified Sources

- Nexus API game list verified the `potioncraftalchemistsimulator` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Vortex has no bundled Potion Craft game handler, so DMM uses the source-reviewed shared Vortex `modtype-bepinex` behavior.
- Representative Nexus install instructions show BepInEx plugin archives targeting `Potion Craft/BepInEx/plugins`.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM detects the Steam Deck Windows/Proton install through `Potion Craft.exe`, `UnityPlayer.dll`, and `Potion Craft_Data/globalgamemanagers`.
- DMM supports Vortex-compatible BepInEx runtime packages, BepInEx root/config packages, ConfigurationManager archives, and loose BepInEx plugin DLL archives.
- Unknown archive layouts remain blocked by an extension-owned catch-all until a specific install rule is verified.

## Beta Gaps

- Live-test a Potion Craft Nexus BepInEx runtime package and at least one plugin archive on the Steam Deck.
- Add non-BepInEx install rules only after source or representative archive review proves the target roots.
