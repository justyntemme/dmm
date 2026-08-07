# Disco Elysium Extension Notes

## Identity

- Steam AppID: `632470`
- DMM extension ID: `discoelysium`
- Nexus domain: `discoelysium`
- Vortex manifest package: Nexus site mod `1643`, file `7265`, version `0.1.4`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Disco Elysium Vortex extension page and package `0.1.4`: `https://www.nexusmods.com/site/mods/1643`
- Vortex package source file: `index.js` from Nexus site mod `1643`, file `7265`
- Vortex shared BepInEx extension source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Disco Elysium`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the verified Vortex extension package.
- Checks for `disco.exe`, `GameAssembly.dll`, and `disco_Data`.
- Supports Vortex-backed archive installers for root game/data-folder mods, BepInEx runtime packages, BepInEx plugin/config roots, BepInEx Configuration Manager archives, `GameAssembly.dll` replacements, and Unity asset/resource files.
- Canonicalizes Vortex's Windows-tolerant `BepinEx` path spelling to `BepInEx` for Steam Deck filesystem safety.
- Blocks only the Vortex fallback/manual archive shape until a more specific extension-owned installer can classify it safely.

## Beta Gaps

- Automatic BepInEx helper download is not implemented yet; users can install the BepInEx Unity IL2CPP x64 runtime through the normal DMM capture/import pipeline.
- Live-test representative Nexus archives for each supported installer shape.
- Add generic dependency-helper tooling before mirroring Vortex's automatic BepInEx download action.
