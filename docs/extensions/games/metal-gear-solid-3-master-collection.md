# Metal Gear Solid 3 Master Collection Extension Notes

## Identity

- Steam AppID: `2131650`
- DMM extension ID: `metalgearsolid3mc`
- Nexus domain: `metalgearsolid3mc`

## Verified Sources

- Vortex extension source copied from the Steam Deck Vortex plugin cache:
  - `/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Metal Gear Solid 3 - Snake Eater - Master Collection Vortex Extension v1.0.0/index.js`
- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/728`
- Live Steam Deck path check:
  - `/home/deck/.local/share/Steam/steamapps/common/MGS3` exists, but no `appmanifest_2131650.acf` was present and the folder did not contain `launcher.exe` or `METAL GEAR SOLID3.exe` during this pass.

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Vortex root `modPath: "."` behavior is represented as a game-root archive installer.
- Vortex `mergeMods: true` behavior is represented by stripping the common archive wrapper folder before staging to the game root.
- The Vortex-required files are exposed as an extension-owned runtime diagnostic.

## Beta Gaps

- The current Deck install appears stale or incomplete; reinstall the game before live validation.
- Live Nexus archive validation is required.
- The Vortex extension does not define richer install handlers, tools, or load-order behavior; if real archives need special ordering beyond profile priority, verify from representative mods before extending DMM.
- Metal Gear Solid 1 Master Collection should be added separately from its own verified extension source instead of assuming identical behavior.

## Validation Targets

- A clean Steam install with `appmanifest_2131650.acf`.
- A small texture or fix archive from the `metalgearsolid3mc` Nexus domain.
- A wrapped archive whose contents should stage to the game root after common-root stripping.
