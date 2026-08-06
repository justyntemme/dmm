# RimWorld Extension Notes

## Identity

- Steam AppID: `294100`
- DMM extension ID: `rimworld`
- Nexus domain: `rimworld`

## Verified Sources

- Vortex RimWorld game extension: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-rimworld/src/index.js`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- One-`About.xml` archive planning is implemented.
- Package-ID-derived mod folder naming under `Mods` is represented.
- Filtered deployable files are implemented.
- Game version discovery reads `version.txt` with Deck case-insensitive lookup for `Version.txt`.
- Steam Workshop coexistence/actions are declared.

## Beta Gaps

- Multi-mod bundles are intentionally blocked for review.
- Live Nexus archive validation is required.
- RimWorld mod list/load-order integration needs parity review.
- Workshop enable/disable/unsubscribe needs safe manual validation.

## Validation Targets

- Single `About.xml` Nexus archive.
- Archive with multiple `About.xml` files to confirm blocking.
- Workshop coexistence with DMM-owned Nexus mods.
