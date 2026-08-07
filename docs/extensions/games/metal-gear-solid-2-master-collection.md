# Metal Gear Solid 2 Master Collection Extension Notes

## Identity

- Steam AppID: `2131640`
- DMM extension ID: `metalgearsolid2mc`
- Nexus domain: `metalgearsolid2mc`

## Verified Sources

- Vortex extension source copied from the Steam Deck Vortex plugin cache:
  - `/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Metal Gear Solid 2 - Sons of Liberty - Master Collection Vortex Extension v1.0.0/index.js`
- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/727`
- Live Steam Deck install path verification:
  - Game folder: `/home/deck/.local/share/Steam/steamapps/common/MGS2`
  - Executables: `launcher.exe`, `METAL GEAR SOLID2.exe`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Vortex root `modPath: "."` behavior is represented as a game-root archive installer.
- Vortex `mergeMods: true` behavior is represented by stripping the common archive wrapper folder before staging to the game root.
- The Vortex-required files are exposed as an extension-owned runtime diagnostic.

## Beta Gaps

- Live Nexus archive validation is required.
- The Vortex extension does not define richer install handlers, tools, or load-order behavior; if real archives need special ordering beyond profile priority, verify from representative mods before extending DMM.
- Metal Gear Solid 1 and 3 Master Collection extensions should be added separately from their own verified extension sources instead of assuming identical behavior.

## Validation Targets

- A small texture or fix archive from the `metalgearsolid2mc` Nexus domain.
- A wrapped archive whose contents should stage to the game root after common-root stripping.
- A conflicting pair of texture/archive mods to validate profile priority and rollback behavior.
