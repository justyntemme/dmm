# Metal Gear & Metal Gear 2: Solid Snake - Master Collection Extension Notes

## Identity

- Steam AppID: `2131680`
- DMM extension ID: `metalgearandmetalgear2mc`
- Nexus domain: `metalgearandmetalgear2mc`

## Verified Sources

- Vortex central extension manifest entry: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/725`
- Verified Vortex extension package file: `https://www.nexusmods.com/site/mods/725?tab=files&file_id=2522`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/MG and MG2`

## Current DMM Capability

- Root archive deployment with common wrapper stripping, matching the verified Vortex extension's `mergeMods: true`, `modPath: "."`, and relative mod path.
- Required-file diagnostics for `launcher.exe` and `METAL GEAR.exe`.

## Beta Gaps

- Needs live archive validation with a safe Nexus mod.
- No load-order, plugin activation, or external tool behavior was declared by the verified Vortex extension.
