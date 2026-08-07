# Metal Gear Solid - Master Collection Extension Notes

## Identity

- Steam AppID: `2131630`
- DMM extension ID: `metalgearsolidmc`
- Nexus domain: `metalgearsolidmc`

## Verified Sources

- Vortex central extension manifest entry: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/726`
- Verified Vortex extension package file: `https://www.nexusmods.com/site/mods/726?tab=files&file_id=2523`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/MGS1`

## Current DMM Capability

- Root archive deployment with common wrapper stripping, matching the verified Vortex extension's `mergeMods: true`, `modPath: "."`, and relative mod path.
- Required-file diagnostics for `METAL GEAR SOLID.exe`.

## Beta Gaps

- Needs live archive validation with a safe Nexus mod.
- No load-order, plugin activation, or external tool behavior was declared by the verified Vortex extension.
