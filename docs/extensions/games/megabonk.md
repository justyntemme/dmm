# Megabonk Extension Notes

## Identity

- Steam AppID: `3405340`
- DMM extension ID: `megabonk`
- Nexus domain: `megabonk`
- Vortex manifest package: Nexus site mod `1495`, file `7663`, version `0.1.3`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Megabonk Vortex extension page: `https://www.nexusmods.com/site/mods/1495`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/Megabonk`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the native Linux executable, `GameAssembly.so`, and Unity data folder.
- Blocks archive installs because the manifest describes a BepInEx-or-MelonLoader choice plus loader-specific target folders.

## Beta Gaps

- Inspect the Vortex package and representative archives.
- Add generic loader-choice installer support before enabling Megabonk installs.
