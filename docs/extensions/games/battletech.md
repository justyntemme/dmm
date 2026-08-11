# BattleTech

## Verified Sources

- Vortex source: `/tmp/dmm-vortex/extensions/games/game-battletech/src/index.js`
- Vortex registers Steam app `637090`, Nexus game ID `battletech`, executable `BattleTech.exe`, and required files `BattleTech.exe` plus `BattleTechLauncher.exe`.
- Vortex deploys mods to Documents `My Games/BattleTech/mods`.
- Vortex reads game version from `BattleTech_Data/StreamingAssets/version.json` field `ProductVersion`.
- Vortex listens for `added-files` and, when a newly generated file has exactly one candidate owner, copies it back into that mod's installed folder and removes the unmanaged game-folder copy.

## DMM Capability

- DMM extension: `internal/extensions/battletech`
- Steam app: `637090`
- Nexus domain: `battletech`
- Deployment target: Steam Deck Proton Documents root for `My Games/BattleTech/mods`.
- Installer: source-backed archive-root deployment into the BattleTech Documents mod root.
- Version provider: source-backed `ProductVersion` parser.
- Lifecycle: source-backed `added-files` adoption through DMM's reusable snapshot/diff runtime. The game extension copies single-owner generated files into the owning mod's staging folder, removes the unmanaged game copy, and DMM persists the adopted file into the installed mod manifest before deployment planning continues.

## Remaining Gaps

- Live Steam Deck validation against an actual BattleTech mod that generates runtime files.
- Removed-file event emission is implemented in the shared new-file monitor runtime. Source review found BattleTech only consumes `added-files`, so there is no BattleTech-specific removed-file handler to port.
- Multi-owner generated-file resolution is surfaced through the shared candidate-owner event path. BattleTech intentionally adopts only single-owner generated files, matching the verified Vortex extension behavior.
