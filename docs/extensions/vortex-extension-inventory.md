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
- DMM counterparts identified in the current pass: 16
  - Framework/shared counterparts: 7
  - Game counterparts: 9

## Framework And Shared Extensions

- [ ] `changelog-dashlet`
- [x] `common-interpreters` - DMM counterpart: `internal/extensions/commoninterpreters` framework extension. Current parity is registered interpreter metadata for `.jar`, `.py`, `.vbs`, `.cmd`, and `.bat`; runtime execution remains gated until a converted extension needs interpreter launching.
- [ ] `documentation`
- [ ] `extension-dashlet`
- [ ] `feedback`
- [ ] `fnis-integration`
- [ ] `gamebryo-archive-check`
- [x] `gamebryo-archive-invalidation` - DMM counterpart: shared `internal/extensions/gamebryo` archive-invalidation handler.
- [ ] `gamebryo-archive-support`
- [ ] `gamebryo-bsa-support`
- [ ] `gamebryo-plugin-indexlock`
- [x] `gamebryo-plugin-management` - DMM counterpart: shared `internal/extensions/gamebryo` plugin activation capability. Full Bethesda load-order/sorting parity remains tracked separately.
- [ ] `gamebryo-savegame-management`
- [ ] `gamebryo-test-settings`
- [x] `gameinfo-steam` - DMM counterpart: core Steam library/app manifest discovery. This is core platform capability in DMM, not a separate game extension.
- [ ] `gamestore-gog`
- [ ] `gamestore-origin`
- [ ] `gamestore-uplay`
- [ ] `gamestore-xbox`
- [ ] `gameversion-hash`
- [ ] `issue-tracker`
- [ ] `local-gamesettings`
- [ ] `meta-editor`
- [ ] `mo-import`
- [ ] `mod-content`
- [ ] `mod-dependency-manager`
- [ ] `mod-highlight`
- [ ] `mod-report`
- [x] `modtype-bepinex` - DMM counterpart: shared `internal/extensions/bepinex` Unity/BepInEx installer/runtime capability.
- [ ] `modtype-dazip`
- [ ] `modtype-dinput`
- [ ] `modtype-enb`
- [ ] `modtype-gedosato`
- [ ] `modtype-umm`
- [ ] `morrowind-plugin-management`
- [ ] `mtframework-arc-support`
- [ ] `new-file-monitor`
- [ ] `nmm-import-tool`
- [ ] `open-directory`
- [ ] `quickbms-support`
- [x] `script-extender-error-check` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender runtime requirement capability. Full Vortex parity still needs source review per game/tool.
- [x] `script-extender-installer` - DMM counterpart: shared `internal/extensions/gamebryo` script-extender installer capability.
- [ ] `test-gameversion`
- [ ] `test-setup`
- [ ] `theme-switcher`
- [ ] `titlebar-launcher`

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
