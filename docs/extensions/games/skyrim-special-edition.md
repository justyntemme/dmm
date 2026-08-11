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
- Gamebryo plugin activation generation is represented, including LOOT metadata and profile-local LOOT rules.
- The backend exposes LOOT refresh/sort/userlist endpoints for Gamebryo plugin activation profiles.
- FNIS integration settings, patch-list parsing, installed/version diagnostics, automatic generator launch, wait-for-exit, and generated profile output deployment are source-backed through the reusable DMM FNIS helper.
- BodySlide is represented as a source-backed supported tool, matching the verified Vortex extension behavior.

## Beta Gaps

- Live Skyrim SE validation is required.
- Packaged LOOT sorter validation against a real Skyrim SE plugin set is still required.
- Plugin dependency validation is present through the shared Gamebryo diagnostics, but needs live validation against representative Skyrim SE missing-master and blueprint-master fixtures.
- Archive invalidation needs live verification.
- BodySlide has no source-backed automatic generated-output workflow in the verified Vortex game extension; do not invent one unless a source-backed extension requires it.

## Validation Targets

- SKSE64 archive.
- Simple texture/mesh mod.
- ESP/ESM/ESL plugin mod.
- FOMOD mod with conditional choices.
