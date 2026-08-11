# Half-Life Extension Notes

## Identity

- Steam AppID: `70`
- DMM extension ID: `halflife`
- Nexus domain: `halflife`

## Verified Sources

- Nexus API game list verified the `halflife` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Half-Life handler was found.
- Representative Nexus instructions verify copying subtitle/mod folders into the Half-Life root and launching the resulting mod from Steam.
- Representative Nexus instructions verify loose BSP map files belong under `valve/maps`.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM supports direct `valve` folder replacement/content archives.
- DMM supports loose `.bsp` map archives by staging them under `valve/maps`.
- DMM supports standalone GoldSrc mod folders with `liblist.gam` by staging them as game-root mod folders and declaring a dynamic primary launch tool that sets Half-Life's `-game "<folder>"` argument when exactly one standalone mod is enabled.
- Executable patchers/tools and unclassified archives remain blocked until a specific extension-owned rule can transform them safely.

## Beta Gaps

- Live-test one `valve` replacement archive and one loose BSP map archive.
- Live-test one standalone GoldSrc mod folder and confirm Steam launch options are updated through the Decky launch-tool bridge.
