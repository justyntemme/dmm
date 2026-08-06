# X4: Foundations Extension Notes

## Identity

- Steam AppID: `392160`
- DMM extension ID: `x4foundations`
- Nexus domain: `x4foundations`

## Verified Sources

- Vortex X4: Foundations game extension: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-x4foundations/src/index.js`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Game-root `extensions` installer is represented.
- `content.xml` metadata extraction and output-root handling are implemented.
- Dynamic Proton Documents target root is represented for `Documents/Egosoft/X4/{steamUserId}/extensions`.
- Steam Workshop coexistence/actions are declared.
- Game version discovery reads `version.dat`.

## Beta Gaps

- Live Nexus archive validation is required.
- Game-root vs Documents target selection needs UX validation.
- Workshop coexistence with X4's in-game extension manager needs review.

## Validation Targets

- Archive with root `content.xml`.
- Archive with nested `extensions/<id>/content.xml`.
- Documents-root mod if needed by the game.
