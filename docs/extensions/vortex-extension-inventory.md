# Vortex Extension Inventory

Source: local clone `/tmp/dmm-vortex`
Remote: `ssh://git@github.com/Nexus-Mods/Vortex.git`
Source commit: `c57894eb71af8234b58a6bd15ae5ab543eccac3a`
Collected: 2026-08-08

This is the parity backlog for DMM extensions and shared extension capabilities. Each Vortex extension below gets a DMM counterpart: either a game extension, a shared extension-framework capability, a provider/store capability, a mod-type capability, or an explicit documented decision that the Vortex extension is not applicable to DMM.

Status legend:

- `[x]` means a DMM counterpart already exists and a duplicate should not be created.
- `[ ]` means no DMM counterpart has been identified yet.
- `Counterpart exists` does not mean full Vortex parity is complete; verify source parity before marking implementation work complete elsewhere.

Rules:

- Verify each Vortex extension from source before implementing parity behavior.
- Keep game-specific behavior inside the DMM game extension.
- Add generic extension-framework APIs when multiple games or mod types need the same kind of capability.
- Do not mark an entry complete only because a placeholder DMM extension exists.

Counts:

- Framework/shared Vortex extensions: 46
- Game Vortex extensions: 86
- Total Vortex extension entries: 132
- DMM counterparts identified in the current pass: 132
  - Framework/shared counterparts: 46
  - Game counterparts: 86

## Framework And Shared Extensions

- [x] `changelog-dashlet` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed dashlet/state metadata marked blocked until DMM has a generic extension UI/state runtime.
- [x] `common-interpreters` - DMM counterpart: `internal/extensions/commoninterpreters` framework extension. Current parity is registered interpreter metadata for `.jar`, `.py`, `.vbs`, `.cmd`, and `.bat`; runtime execution remains gated until a converted extension needs interpreter launching.
- [x] `documentation` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed main-page/action/todo metadata marked blocked until DMM has a generic extension UI runtime.
- [x] `extension-dashlet` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed dashlet metadata marked blocked until DMM has a generic extension UI runtime.
- [x] `feedback` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed feedback main-page/dialog/action/state metadata marked blocked; public feedback submission is not part of Steam Deck MVP.
- [x] `fnis-integration` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed FNIS settings/action/test/todo metadata and blocked `deploy-single-mod` dependency until DMM has an external-tool/deploy-single-mod runtime.
- [x] `gamebryo-archive-check` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed Gamebryo archive compatibility test metadata marked blocked until DMM has BSA/BA2 inspection parity.
- [x] `gamebryo-archive-invalidation` - DMM counterpart: shared `internal/extensions/gamebryo` archive-invalidation handler.
- [x] `gamebryo-archive-support` - DMM counterpart: `internal/extensions/gamebryoarchive` framework extension. Current parity is source-backed BA2/BSA archive type metadata with blocked runtime status until native list/extract/write engines exist.
- [x] `gamebryo-bsa-support` - DMM counterpart: `internal/extensions/gamebryoarchive` framework extension. Current parity is source-backed BSA archive type metadata with blocked runtime status until the native BSA engine exists.
- [x] `gamebryo-plugin-indexlock` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed plugin index-lock state/table metadata marked blocked until DMM has Gamebryo plugin table/rule runtime parity.
- [x] `gamebryo-plugin-management` - DMM counterpart: shared `internal/extensions/gamebryo` plugin activation capability. Full Bethesda load-order/sorting parity remains tracked separately.
- [x] `gamebryo-savegame-management` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed savegame state/action/main-page/profile-feature metadata marked blocked until DMM implements savegame profile management.
- [x] `gamebryo-test-settings` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed Oblivion/Skyrim settings test metadata marked blocked until DMM has Gamebryo settings validation.
- [x] `gameinfo-steam` - DMM counterpart: core Steam library/app manifest discovery. This is core platform capability in DMM, not a separate game extension.
- [x] `gamestore-gog` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows GOG Galaxy registry/client integration.
- [x] `gamestore-origin` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Origin/EA manifest and protocol integration.
- [x] `gamestore-uplay` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Ubisoft registry/protocol integration.
- [x] `gamestore-xbox` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Xbox app registry and shell launch integration.
- [x] `gameversion-hash` - DMM counterpart: `internal/extensions/gameversionhash` framework extension. Current parity is source-backed provider/API metadata marked blocked until DMM supports extension-declared hash inputs and the Vortex backend hash map resolver.
- [x] `issue-tracker` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed issue dashlet/dialog/state metadata marked blocked until DMM has a generic issue/reporting runtime.
- [x] `local-gamesettings` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed profile-feature/test metadata and blocked `bake-settings` API until DMM implements profile-local game settings.
- [x] `meta-editor` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed metadata editor dialog/action/state metadata marked blocked until DMM has a generic metadata edit runtime.
- [x] `mo-import` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed MO import dialog/action metadata marked blocked; actual MO import remains a future migration feature.
- [x] `mod-content` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod-content table/action metadata marked blocked until DMM has generic extension table attributes/actions.
- [x] `mod-dependency-manager` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed dependency state/table/action/dialog/settings/start-hook metadata marked blocked until DMM implements dependency/rule graph runtime and UI.
- [x] `mod-highlight` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod highlight table/action/state metadata marked blocked until DMM has generic table attributes and UI state.
- [x] `mod-report` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod report action metadata marked blocked until DMM has a reporting/export flow.
- [x] `modtype-bepinex` - DMM counterpart: shared `internal/extensions/bepinex` Unity/BepInEx installer/runtime capability.
- [x] `modtype-dazip` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed DAZIP mod type and installer metadata marked blocked until nested DAZIP extraction/submodule planning and Addins.xml registration exist.
- [x] `modtype-dinput` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed DInput mod type and installer metadata marked blocked until executable-relative DLL deployment and unsafe DLL confirmation exist.
- [x] `modtype-enb` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed ENB mod type metadata marked blocked until game-root deployment and unsafe DLL confirmation are implemented; Vortex's ENB installer registration is currently commented out upstream.
- [x] `modtype-gedosato` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed GeDoSaTo mod type and installer metadata marked blocked until external tool discovery and texture-folder targeting exist.
- [x] `modtype-umm` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed Unity Mod Manager mod type, installer, and `ummAddGame` API metadata marked blocked until DMM has a typed UMM helper/API and tool discovery flow.
- [x] `morrowind-plugin-management` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed Morrowind plugin main-page metadata marked blocked until DMM implements Morrowind plugin activation/load-order runtime.
- [x] `mtframework-arc-support` - DMM counterpart: `internal/extensions/mtframeworkarc` framework extension. Current parity is source-backed ARC archive type metadata with blocked runtime status until a Deck-safe ARCtool bridge or native ARC engine exists.
- [x] `new-file-monitor` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed lifecycle/new-file-adoption metadata; DMM lifecycle events exist but managed adoption UI/runtime remains blocked.
- [x] `nmm-import-tool` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed NMM import dialog/action/todo/state metadata marked blocked; actual NMM import remains a future migration feature.
- [x] `open-directory` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed open-folder action metadata marked blocked until DMM has a Deck-safe open-directory action.
- [x] `quickbms-support` - DMM counterpart: `internal/extensions/quickbmssupport` framework extension. Current parity is source-backed QuickBMS API metadata for register/list/extract/write/reimport with blocked runtime status until the executable bridge exists.
- [x] `script-extender-error-check` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender runtime requirement capability. Full Vortex parity still needs source review per game/tool.
- [x] `script-extender-installer` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender installer capability.
- [x] `test-gameversion` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed game-version test and `registerGameInfoProvider` metadata marked blocked until generic game-info provider runtime exists.
- [x] `test-setup` - DMM counterpart: `internal/extensions/vortexsharedsystems` framework extension. Current parity is source-backed setup test metadata marked blocked until DMM has generic setup/uninstall-entry test runtime.
- [x] `theme-switcher` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed theme settings/state metadata marked blocked; custom theming is not part of Steam Deck MVP.
- [x] `titlebar-launcher` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed titlebar launcher settings/state metadata marked blocked; DMM launch controls are Decky-native.

## Game Extensions

Source-backed catalog coverage:

- Remaining Vortex game entries are represented by `internal/extensions/vortexgamecatalog` unless a richer dedicated first-party DMM extension is listed.
- The catalog preserves verified Vortex game IDs, Nexus domains, Steam app IDs when Vortex declares them, `registerGameStub` support-mod metadata, source-backed `registerGame` metadata when ported, external `supportedTools` metadata when ported, and source URLs pinned to the scanned Vortex commit.
- Catalog entries do not claim executable install parity. When the Vortex source registers custom installers, mod types, load order, lifecycle behavior, or other game-specific runtime logic that DMM has not ported yet, the DMM extension exposes blocked metadata so the UI/API can report the gap instead of silently installing unsafely.

- [x] `game-7daystodie` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-ahatintime` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-baldursgate3` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-battletech` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-bladeandsorcery` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-bloodstainedritualofthenight` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-breakingwheel` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-codevein` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-conanexiles` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-cyberpunk2077` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-daggerfallunity` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-darkestdungeon` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-darksouls` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-darksouls2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-dawnofman` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-divinityoriginalsin2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-dmc5` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-dragonage` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-dragonage2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-dragons-dogma` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-elex` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-enderal` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-factorio` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-fallout3` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-fallout4` - DMM counterpart: `internal/extensions/fallout4`.
- [x] `game-fallout4vr` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-falloutnv` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-galciv3` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-gardenpaws` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-greedfall` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-grimdawn` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-grimrock` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-kenshi` - DMM counterpart: `internal/extensions/kenshi`.
- [x] `game-kerbalspaceprogram` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-kingdomcome-deliverance` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-masterchiefcollection` - DMM counterpart: `internal/extensions/masterchiefcollection`.
- [x] `game-microsoftflightsimulator` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-monster-hunter-world` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-morrowind` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-mount-and-blade` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-mount-and-blade2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-nehrim` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-neverwinter-nights` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-neverwinter-nights2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-nomanssky` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-oblivion` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-oni` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-palworld` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-pathfinderkingmaker` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-pathfinderwrathoftherighteous` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-pillarsofeternity2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-prisonarchitect` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-re2remake` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-re3remake` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-rimworld` - DMM counterpart: `internal/extensions/rimworld`.
- [x] `game-sekiro` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-shadowrunreturns` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-sims3` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-sims4` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-skyrim` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-skyrimse` - DMM counterpart: `internal/extensions/skyrimse`.
- [x] `game-skyrimvr` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-spyroreignitedtrilogy` - DMM counterpart: `internal/extensions/spyroreignitedtrilogy`.
- [x] `game-starbound` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-stardewvalley` - DMM counterpart: `internal/extensions/stardewvalley`.
- [x] `game-starfield` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-stateofdecay` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-subnautica` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-subnauticabelowzero` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-survivingmars` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-sw-kotor` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-teamfortress2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-teso` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-torchlight2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-totalwarthreekingdoms` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-untitledgoose` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-vtmbloodlines` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-warthunder` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-witcher` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-witcher2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-witcher3` - DMM counterpart: `internal/extensions/witcher3`.
- [x] `game-wolcen` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-worldoftanks` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-x4foundations` - DMM counterpart: `internal/extensions/x4foundations`.
- [x] `game-xcom2` - DMM counterpart: `internal/extensions/vortexgamecatalog` source-backed catalog entry. Any Vortex custom installer, mod-type, load-order, or lifecycle surfaces remain blocked until ported into DMM extension capabilities.
- [x] `game-xrebirth` - DMM counterpart: `internal/extensions/xrebirth`.
