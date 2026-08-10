# Skyrim Special Edition Extension Notes

## Identity

- Steam AppID: `489830`
- DMM extension ID: `skyrimse`
- Nexus domain: `skyrimspecialedition`

## Verified Sources

- Vortex Skyrim Special Edition game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-skyrimse/src/index.js`
- Vortex Gamebryo plugin activation support: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-plugin-management/src/util/gameSupport.ts`
- Vortex Gamebryo archive invalidation support: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts`
- Vortex script extender installer: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/script-extender-installer/src/installer.ts`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Data-root archive planning is implemented.
- FOMOD installer-choice support is declared for Skyrim data-root installs.
- SKSE64 script-extender archive inference is extension-owned.
- SKSE64 primary launch-tool metadata is registered.
- SSEEdit, Wrye Bash, FNIS, BodySlide, and Creation Kit tool metadata are registered.
- BodySlide x64 is preferred when present, with BodySlide.exe kept as the fallback executable, matching the Vortex extension source.
- Gamebryo plugin activation generation is represented.
- FNIS integration settings, patch-list parsing, and the FNIS installed/version diagnostic are source-backed through the reusable DMM FNIS helper.

## Beta Gaps

- Live Skyrim SE validation is required.
- LOOT-style sorting is not implemented.
- Plugin dependency validation is not complete.
- Archive invalidation needs live verification.
- BodySlide/FNIS generated output workflows are not complete. FNIS still needs the generic Decky wait-for-tool-exit and generated profile-mod runtime before DMM can mirror Vortex's automatic generator deploy path.

## Validation Targets

- SKSE64 archive.
- Simple texture/mesh mod.
- ESP/ESM/ESL plugin mod.
- FOMOD mod with conditional choices.
