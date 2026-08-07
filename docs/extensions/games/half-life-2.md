# Half-Life 2 Extension Notes

## Identity

- Steam AppID: `220` Half-Life 2
- DMM extension ID: `halflife2`
- Nexus domain: `halflife2`
- Vortex manifest package: Nexus site mod `80`, file `516`, version `1.1.0`
- Vortex package game ID: `half-life2`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Half-Life 2 Vortex extension package v1.1.0: `https://www.nexusmods.com/site/mods/80?tab=files`
- Nexus API domain verification: `https://api.nexusmods.com/v1/games/halflife2.json`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Half-Life 2`

## Current DMM Capability

- DMM declares the verified Half-Life 2 Nexus domain `halflife2`.
- DMM registers the source-verified Steam AppID `220`. Lost Coast and the episodes are no longer registered through this extension because the inspected Vortex package only registers AppID `220`.
- DMM checks for `hl2/gameinfo.txt` plus a native Linux or Windows executable marker.
- DMM installs `.vpk` archives through the source-verified Vortex installer shape into `hl2/custom`.
- FOMOD archives are skipped so the shared installer-choice pipeline can own them if a future Source-engine extension declares support.

## Beta Gaps

- Validate representative Nexus Half-Life 2 VPK archives through the browser `nxm://` capture path.
- Add separate source-backed handling for Lost Coast and the episodes only after their Vortex/official behavior is verified.
- Keep sourcemods, root-folder replacements, custom folder bundles, and external tool flows blocked until source or representative archive behavior is verified.
