# Kenshi Extension Notes

## Identity

- Steam AppID: `233860`
- DMM extension ID: `kenshi`
- Nexus domain: `kenshi`

## Verified Sources

- Vortex Kenshi game extension: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-kenshi/src/index.js`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- `.mod` archive planning roots at the folder containing the `.mod` file.
- Deployment target is `mods/<modName>/...`.
- Vortex-declared tools are registered.
- Game version discovery reads `currentVersion.txt`.
- Steam Workshop coexistence/actions are declared and live Workshop item sync has been validated.

## Beta Gaps

- Live Nexus archive validation is required.
- Workshop enable/disable/unsubscribe needs safe manual validation.
- Load-order semantics for mixed Nexus plus Workshop mods need review.

## Validation Targets

- A simple Nexus `.mod` archive.
- A Nexus archive that includes nested folders.
- Existing Steam Workshop load order with DMM-owned Nexus files.
