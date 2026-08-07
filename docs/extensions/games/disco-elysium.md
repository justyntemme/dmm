# Disco Elysium Extension Notes

## Identity

- Steam AppID: `632470`
- DMM extension ID: `discoelysium`
- Nexus domain: `discoelysium`
- Vortex manifest package: Nexus site mod `1643`, file `7265`, version `0.1.4`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Disco Elysium Vortex extension page: `https://www.nexusmods.com/site/mods/1643`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Disco Elysium`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for `disco.exe`, `GameAssembly.dll`, and `disco_Data`.
- Blocks archive installs because the manifest describes BepInEx setup and data-folder normalization that must be source-verified before DMM writes files.

## Beta Gaps

- Inspect the Vortex package and representative archives.
- Add extension-owned rules for BepInEx payloads, data folder normalization, root replacements, and fallback/manual cases.
