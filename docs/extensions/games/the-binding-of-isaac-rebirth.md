# The Binding of Isaac: Rebirth Extension Notes

## Identity

- Steam AppID: `250900`
- DMM extension ID: `thebindingofisaacrebirth`
- Nexus domain: `thebindingofisaacrebirth`
- Vortex manifest package: Nexus site mod `516`, file `4127`, version `1.0.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- The Binding of Isaac Vortex extension package `1.0.0`: `https://www.nexusmods.com/site/mods/516`
- Vortex package source file: `index.js` from Nexus site mod `516`, file `4127`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/The Binding of Isaac Rebirth`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the verified Vortex extension package.
- Checks for `isaac-ng.exe`.
- Supports Vortex's default `modPath: "mods"` archive-root deployment while preserving the archive's top-level mod folder.

## Beta Gaps

- Live-test representative Nexus Isaac archives.
- Add narrower installers only if source review of specific Isaac archive classes proves the default Vortex `mods` path is insufficient.
