# The Witcher 3 Extension Notes

## Identity

- Steam AppID: `292030`
- Steam AppID DX entry: `499450`
- DMM extension ID: `witcher3`
- Nexus domain: `witcher3`

## Verified Sources

- Vortex Witcher 3 game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/index.ts`
- Vortex Witcher 3 installers: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/installers.ts`
- Vortex Witcher 3 common merge/load-order constants: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/common.ts`
- Vortex Witcher 3 lifecycle and load-order hooks: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/eventHandlers.ts`

## Current DMM Capability

- Nexus domain and Steam AppIDs are registered.
- Menu mod root, mixed mod/DLC, top-level mod, content-only, and DLC installer shapes are represented.
- Script Merger is registered as a launch tool and blocked as a mod archive.
- Basic managed `mods.settings` generation exists through an extension deploy hook.

## Beta Gaps

- Advanced menu XML merge is incomplete.
- Script Merger setup and prompts are incomplete.
- Manual load-order preservation and full load-order UI semantics are incomplete.
- Collection/profile data behavior needs parity review.
- Live archive validation is required.

## Validation Targets

- Plain `mods/modName` archive.
- `dlc` archive.
- Mixed `mods` plus `dlc` archive.
- Menu XML mod that requires merge behavior.
