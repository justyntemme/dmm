# Citizen Sleeper Extension Notes

## Identity

- Steam AppID: `1578650`
- DMM extension ID: `citizensleeper`
- Nexus domain: `citizensleeper`
- Vortex manifest package: Nexus site mod `444`, file `1656`, version `1.0.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Citizen Sleeper Vortex extension package `1.0.0`: `https://www.nexusmods.com/site/mods/444`
- Vortex package source file: `index.js` from Nexus site mod `444`, file `1656`
- Vortex shared BepInEx extension source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Citizen Sleeper`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the verified Vortex extension package.
- Checks for `Citizen Sleeper.exe`.
- Supports BepInEx runtime packages at the game root and plugin archives under `BepInEx/plugins`, matching Vortex's `queryModPath`.
- Preserves plugin archive wrapper folders because BepInEx plugin mods commonly ship as folder-scoped packages.

## Beta Gaps

- Automatic BepInEx helper download is not implemented yet; users can install BepInEx through the normal DMM capture/import pipeline.
- The Vortex package references `BepInEx.cfg`, but the downloaded package did not contain that file; DMM does not synthesize it until the intended config content is source-verified.
- Live-test representative Nexus Citizen Sleeper archives.
