# Half-Life 2 Extension Notes

## Identity

- Steam AppIDs: `220` Half-Life 2, `380` Episode One, `420` Episode Two
- DMM extension ID: `halflife2`
- Nexus domain: `halflife2`
- Vortex manifest package: Nexus site mod `80`, file `516`, version `1.1.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Half-Life 2 Vortex extension page: `https://www.nexusmods.com/site/mods/80`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Half-Life 2`

## Current DMM Capability

- DMM declares the verified Half-Life 2 Nexus domain for Half-Life 2 and its two episodes.
- DMM checks for the shared Linux Source executable/script and the `hl2`, `episodic`, and `ep2` `gameinfo.txt` files.
- Archive installs are intentionally blocked because the Vortex extension package/source has not been inspected and Source-engine mods can use multiple incompatible layouts.

## Beta Gaps

- Inspect the actual Vortex Half-Life 2 extension package before adding installer rules.
- Classify representative Nexus archives for base `hl2`, `episodic`, `ep2`, `custom`, VPK, sourcemods, and external-tool flows.
- Decide whether Lost Coast (`340`) belongs in this extension after source/package review; it shares the Steam install dir but was not explicitly called out by the verified Vortex manifest description.
- Add only extension-owned installers that map verified layouts to safe DMM-owned deployment roots.
