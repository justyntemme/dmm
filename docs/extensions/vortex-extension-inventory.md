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
- DMM counterparts identified in the current pass: 44
  - Framework/shared counterparts: 35
  - Game counterparts: 9

## Framework And Shared Extensions

- [x] `changelog-dashlet` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed dashlet/state metadata marked blocked until DMM has a generic extension UI/state runtime.
- [x] `common-interpreters` - DMM counterpart: `internal/extensions/commoninterpreters` framework extension. Current parity is registered interpreter metadata for `.jar`, `.py`, `.vbs`, `.cmd`, and `.bat`; runtime execution remains gated until a converted extension needs interpreter launching.
- [x] `documentation` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed main-page/action/todo metadata marked blocked until DMM has a generic extension UI runtime.
- [x] `extension-dashlet` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed dashlet metadata marked blocked until DMM has a generic extension UI runtime.
- [x] `feedback` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed feedback main-page/dialog/action/state metadata marked blocked; public feedback submission is not part of Steam Deck MVP.
- [ ] `fnis-integration`
- [ ] `gamebryo-archive-check`
- [x] `gamebryo-archive-invalidation` - DMM counterpart: shared `internal/extensions/gamebryo` archive-invalidation handler.
- [x] `gamebryo-archive-support` - DMM counterpart: `internal/extensions/gamebryoarchive` framework extension. Current parity is source-backed BA2/BSA archive type metadata with blocked runtime status until native list/extract/write engines exist.
- [x] `gamebryo-bsa-support` - DMM counterpart: `internal/extensions/gamebryoarchive` framework extension. Current parity is source-backed BSA archive type metadata with blocked runtime status until the native BSA engine exists.
- [ ] `gamebryo-plugin-indexlock`
- [x] `gamebryo-plugin-management` - DMM counterpart: shared `internal/extensions/gamebryo` plugin activation capability. Full Bethesda load-order/sorting parity remains tracked separately.
- [ ] `gamebryo-savegame-management`
- [ ] `gamebryo-test-settings`
- [x] `gameinfo-steam` - DMM counterpart: core Steam library/app manifest discovery. This is core platform capability in DMM, not a separate game extension.
- [x] `gamestore-gog` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows GOG Galaxy registry/client integration.
- [x] `gamestore-origin` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Origin/EA manifest and protocol integration.
- [x] `gamestore-uplay` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Ubisoft registry/protocol integration.
- [x] `gamestore-xbox` - DMM counterpart: `internal/extensions/gamestores` framework extension. Current parity is source-backed metadata marked blocked because Vortex uses Windows Xbox app registry and shell launch integration.
- [x] `gameversion-hash` - DMM counterpart: `internal/extensions/gameversionhash` framework extension. Current parity is source-backed provider/API metadata marked blocked until DMM supports extension-declared hash inputs and the Vortex backend hash map resolver.
- [x] `issue-tracker` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed issue dashlet/dialog/state metadata marked blocked until DMM has a generic issue/reporting runtime.
- [ ] `local-gamesettings`
- [x] `meta-editor` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed metadata editor dialog/action/state metadata marked blocked until DMM has a generic metadata edit runtime.
- [x] `mo-import` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed MO import dialog/action metadata marked blocked; actual MO import remains a future migration feature.
- [x] `mod-content` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod-content table/action metadata marked blocked until DMM has generic extension table attributes/actions.
- [ ] `mod-dependency-manager`
- [x] `mod-highlight` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod highlight table/action/state metadata marked blocked until DMM has generic table attributes and UI state.
- [x] `mod-report` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed mod report action metadata marked blocked until DMM has a reporting/export flow.
- [x] `modtype-bepinex` - DMM counterpart: shared `internal/extensions/bepinex` Unity/BepInEx installer/runtime capability.
- [x] `modtype-dazip` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed DAZIP mod type and installer metadata marked blocked until nested DAZIP extraction/submodule planning and Addins.xml registration exist.
- [x] `modtype-dinput` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed DInput mod type and installer metadata marked blocked until executable-relative DLL deployment and unsafe DLL confirmation exist.
- [x] `modtype-enb` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed ENB mod type metadata marked blocked until game-root deployment and unsafe DLL confirmation are implemented; Vortex's ENB installer registration is currently commented out upstream.
- [x] `modtype-gedosato` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed GeDoSaTo mod type and installer metadata marked blocked until external tool discovery and texture-folder targeting exist.
- [x] `modtype-umm` - DMM counterpart: `internal/extensions/sharedmodtypes` framework extension. Current parity is source-backed Unity Mod Manager mod type, installer, and `ummAddGame` API metadata marked blocked until DMM has a typed UMM helper/API and tool discovery flow.
- [ ] `morrowind-plugin-management`
- [x] `mtframework-arc-support` - DMM counterpart: `internal/extensions/mtframeworkarc` framework extension. Current parity is source-backed ARC archive type metadata with blocked runtime status until a Deck-safe ARCtool bridge or native ARC engine exists.
- [ ] `new-file-monitor`
- [x] `nmm-import-tool` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed NMM import dialog/action/todo/state metadata marked blocked; actual NMM import remains a future migration feature.
- [x] `open-directory` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed open-folder action metadata marked blocked until DMM has a Deck-safe open-directory action.
- [x] `quickbms-support` - DMM counterpart: `internal/extensions/quickbmssupport` framework extension. Current parity is source-backed QuickBMS API metadata for register/list/extract/write/reimport with blocked runtime status until the executable bridge exists.
- [x] `script-extender-error-check` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender runtime requirement capability. Full Vortex parity still needs source review per game/tool.
- [x] `script-extender-installer` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender installer capability.
- [ ] `test-gameversion`
- [ ] `test-setup`
- [x] `theme-switcher` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed theme settings/state metadata marked blocked; custom theming is not part of Steam Deck MVP.
- [x] `titlebar-launcher` - DMM counterpart: `internal/extensions/vortexuisurfaces` framework extension. Current parity is source-backed titlebar launcher settings/state metadata marked blocked; DMM launch controls are Decky-native.

## Game Extensions

- [ ] `game-7daystodie`
- [ ] `game-ahatintime`
- [ ] `game-baldursgate3`
- [ ] `game-battletech`
- [ ] `game-bladeandsorcery`
- [ ] `game-bloodstainedritualofthenight`
- [ ] `game-breakingwheel`
- [ ] `game-codevein`
- [ ] `game-conanexiles`
- [ ] `game-cyberpunk2077`
- [ ] `game-daggerfallunity`
- [ ] `game-darkestdungeon`
- [ ] `game-darksouls`
- [ ] `game-darksouls2`
- [ ] `game-dawnofman`
- [ ] `game-divinityoriginalsin2`
- [ ] `game-dmc5`
- [ ] `game-dragonage`
- [ ] `game-dragonage2`
- [ ] `game-dragons-dogma`
- [ ] `game-elex`
- [ ] `game-enderal`
- [ ] `game-factorio`
- [ ] `game-fallout3`
- [x] `game-fallout4` - DMM counterpart: `internal/extensions/fallout4`.
- [ ] `game-fallout4vr`
- [ ] `game-falloutnv`
- [ ] `game-galciv3`
- [ ] `game-gardenpaws`
- [ ] `game-greedfall`
- [ ] `game-grimdawn`
- [ ] `game-grimrock`
- [x] `game-kenshi` - DMM counterpart: `internal/extensions/kenshi`.
- [ ] `game-kerbalspaceprogram`
- [ ] `game-kingdomcome-deliverance`
- [x] `game-masterchiefcollection` - DMM counterpart: `internal/extensions/masterchiefcollection`.
- [ ] `game-microsoftflightsimulator`
- [ ] `game-monster-hunter-world`
- [ ] `game-morrowind`
- [ ] `game-mount-and-blade`
- [ ] `game-mount-and-blade2`
- [ ] `game-nehrim`
- [ ] `game-neverwinter-nights`
- [ ] `game-neverwinter-nights2`
- [ ] `game-nomanssky`
- [ ] `game-oblivion`
- [ ] `game-oni`
- [ ] `game-palworld`
- [ ] `game-pathfinderkingmaker`
- [ ] `game-pathfinderwrathoftherighteous`
- [ ] `game-pillarsofeternity2`
- [ ] `game-prisonarchitect`
- [ ] `game-re2remake`
- [ ] `game-re3remake`
- [x] `game-rimworld` - DMM counterpart: `internal/extensions/rimworld`.
- [ ] `game-sekiro`
- [ ] `game-shadowrunreturns`
- [ ] `game-sims3`
- [ ] `game-sims4`
- [ ] `game-skyrim`
- [x] `game-skyrimse` - DMM counterpart: `internal/extensions/skyrimse`.
- [ ] `game-skyrimvr`
- [x] `game-spyroreignitedtrilogy` - DMM counterpart: `internal/extensions/spyroreignitedtrilogy`.
- [ ] `game-starbound`
- [x] `game-stardewvalley` - DMM counterpart: `internal/extensions/stardewvalley`.
- [ ] `game-starfield`
- [ ] `game-stateofdecay`
- [ ] `game-subnautica`
- [ ] `game-subnauticabelowzero`
- [ ] `game-survivingmars`
- [ ] `game-sw-kotor`
- [ ] `game-teamfortress2`
- [ ] `game-teso`
- [ ] `game-torchlight2`
- [ ] `game-totalwarthreekingdoms`
- [ ] `game-untitledgoose`
- [ ] `game-vtmbloodlines`
- [ ] `game-warthunder`
- [ ] `game-witcher`
- [ ] `game-witcher2`
- [x] `game-witcher3` - DMM counterpart: `internal/extensions/witcher3`.
- [ ] `game-wolcen`
- [ ] `game-worldoftanks`
- [x] `game-x4foundations` - DMM counterpart: `internal/extensions/x4foundations`.
- [ ] `game-xcom2`
- [ ] `game-xrebirth`
