# Mirror's Edge Extension Notes

## Identity

- Steam AppID: `17410`
- DMM extension ID: `mirrorsedge`
- Nexus domain: `mirrorsedge`

## Verified Sources

- Nexus API game list verified the `mirrorsedge` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Mirror's Edge handler was found.
- Representative Nexus instructions verify placing `.upk` replacements into `TdGame/CookedPC/Characters`.
- Community guide verifies a safer user-Documents `Published/CookedPC` mod-menu flow, which is intentionally separate from the game-root replacement path.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM supports game-root `TdGame/CookedPC` replacement archives.
- Archives rooted at `TdGame/CookedPC`, `CookedPC`, or known CookedPC top-level folders are normalized under `TdGame/CookedPC`.
- DMM supports user-Documents `Published/CookedPC` mod-menu archives through the `mirrorsedge-published-cookedpc-root` Proton Documents target root.
- Executable tools and unclassified payloads remain blocked until a specific extension-owned rule can place them safely.

## Beta Gaps

- Live-test one Nexus `TdGame/CookedPC` package replacement archive.
- Live-test one user-Documents `Published/CookedPC` mod-menu archive and verify the Proton Documents path on Deck.
