# Hollow Knight Extension Notes

## Identity

- Steam AppID: `367520`
- DMM extension ID: `hollowknight`
- Nexus domain: `hollowknight`
- Vortex manifest package: Nexus site mod `376`, file `7365`, version `2.1.1`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Hollow Knight Vortex extension page: `https://www.nexusmods.com/site/mods/376`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/Hollow Knight`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the executable, Unity data folder, and managed assembly folder.
- Blocks archive installs because the manifest describes automatic BepInEx setup, managed DLL replacement, asset placement, and fallback behavior that must be source-verified.

## Beta Gaps

- Inspect the current Vortex extension package and representative Nexus archives.
- Add extension-owned BepInEx, managed assembly, Unity asset, and plugin-folder installers.
