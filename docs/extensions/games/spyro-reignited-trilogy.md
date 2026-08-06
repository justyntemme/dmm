# Spyro Reignited Trilogy Extension Notes

## Identity

- Steam AppID: `996580`
- DMM extension ID: `spyroreignitedtrilogy`
- Nexus domain: `spyroreignitedtrilogy`

## Verified Sources

- Vortex Spyro game extension: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-spyroreignitedtrilogy/src/index.ts`
- Vortex Spyro load-order extension source: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-spyroreignitedtrilogy/src/loadOrder.ts`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- `.pak` archive planning roots at the folder containing the first `.pak`.
- Deployment target is `falcon/content/paks/~mods`.
- FOMOD archives are excluded.
- Unreal pak load-order prefix hook is represented.

## Beta Gaps

- Live Nexus archive validation is required.
- Pak load-order UI needs validation with multiple mods.
- FOMOD exclusion should be tested with a real unsupported archive.

## Validation Targets

- Simple pak archive.
- Multiple pak mods requiring order.
- Unsupported FOMOD archive.
