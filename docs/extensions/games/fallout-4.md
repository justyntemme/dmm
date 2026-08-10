# Fallout 4 Extension Notes

## Identity

- Steam AppID: `377160`
- DMM extension ID: `fallout4`
- Nexus domain: `fallout4`

## Verified Sources

- Vortex Fallout 4 game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-fallout4/src/index.js`
- Vortex Gamebryo plugin activation support: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-plugin-management/src/util/gameSupport.ts`
- Vortex Gamebryo archive invalidation support: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts`
- Vortex script extender installer: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/script-extender-installer/src/installer.ts`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Data-root archive planning is implemented.
- FOMOD installer-choice support is declared for Fallout 4 data-root installs.
- F4SE script-extender archive inference is extension-owned.
- F4SE primary launch-tool metadata is registered.
- FO4Edit, Wrye Bash, and BodySlide tool metadata are registered.
- BodySlide x64 is preferred when present, with BodySlide.exe kept as the fallback executable, matching the Vortex extension source.
- Gamebryo plugin activation generation is represented.
- Steam Workshop coexistence/actions are declared.

## Beta Gaps

- Live Fallout 4 archive validation is still required.
- LOOT-style sorting is not implemented.
- Plugin dependency validation is not complete.
- Archive invalidation needs live verification against Proton paths.
- Workshop enable/disable/unsubscribe needs safe manual validation.

## Validation Targets

- F4SE installer archive.
- Simple `Data` texture/mesh mod.
- ESP/ESM plugin mod.
- FOMOD mod with a small option set.
- Existing Steam Workshop item coexistence.
