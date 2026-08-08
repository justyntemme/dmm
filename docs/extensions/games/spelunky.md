# Spelunky Extension Notes

## Identity

- Steam AppID: `239350`
- DMM extension ID: `spelunky`
- Nexus domain: `spelunky`

## Verified Sources

- Nexus API game list verified the `spelunky` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Spelunky handler was found.
- Representative Nexus instructions verify direct `Data` folder replacement for controller texture packages.
- Representative Nexus instructions verify `Localization` and `Textures` replacement under the game `Data` folder.
- Community modding notes identify Spelunky HD mods as file replacements inside `Data`, with raw texture source edits requiring Spelunktool/Patchlunky.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM supports direct `Data` folder replacement archives.
- Archives containing loose `Localization`, `Music`, or `Textures` folders are normalized under `Data`.
- DMM deploys game-consumed replacement files such as `.pct`, `.ogg`, `.wad`, and `.wix`.
- Raw texture source edits, Patchlunky `.plm` packages, Spelunktool workflows, executable tools, and unclassified archives remain blocked.

## Beta Gaps

- Live-test one Nexus `Data` replacement archive.
- Add Patchlunky/Spelunktool support only after a generic external-tool/transform capability exists.
