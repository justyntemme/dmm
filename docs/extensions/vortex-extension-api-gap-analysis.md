# Vortex Extension API Gap Analysis

Source: local Vortex clone `/tmp/dmm-vortex`
Remote: `ssh://git@github.com/Nexus-Mods/Vortex.git`
Source commit: `c57894eb71af8234b58a6bd15ae5ab543eccac3a`
Collected: 2026-08-08

This document compares extension APIs that Vortex extensions actually call against the current DMM first-party Go extension SDK. It is intentionally source-backed; do not use memory or one downloaded archive as the authority for parity work.

## Scan Summary

The source scan covered 581 TypeScript/JavaScript extension files under `/tmp/dmm-vortex/extensions`.

High-volume Vortex calls:

- `context.registerGame`: 80 call sites.
- `context.registerInstaller`: 70 call sites.
- `context.registerAction`: 51 call sites.
- `context.registerModType`: 40 call sites.
- `context.api.onAsync`: 38 call sites.
- `context.registerTest`: 28 call sites.
- `context.registerReducer`: 27 call sites.
- `context.registerMigration`: 18 call sites.
- `context.registerDialog`: 12 call sites.
- `context.registerAPI`: 10 call sites.
- `context.registerTableAttribute`: 9 call sites.
- `context.registerLoadOrder`: 9 call sites.
- `context.registerGameStub`: 9 call sites.
- `context.registerSettings`: 8 call sites.
- `context.registerDashlet`: 7 call sites.
- `context.registerMainPage`: 7 call sites.
- `context.registerMerge`: 5 call sites.
- `context.registerInterpreter`: 5 call sites.
- `context.registerArchiveType`: 4 call sites.
- `context.registerProfileFeature`: 4 call sites.
- `context.registerGameStore`: 4 call sites.
- `context.optional.registerCollectionFeature`: 3 call sites.
- `context.registerPersistor`: 3 call sites.
- `context.registerLoadOrderPage`: 3 call sites.
- `context.registerAttributeExtractor`: 2 call sites.
- `context.registerGameInfoProvider`: 2 call sites.
- `context.registerProfileFile`: 2 call sites.
- `context.registerActionCheck`: 2 call sites.
- `context.registerControlWrapper`: 1 call site.
- `context.registerHistoryStack`: 1 call site.
- `context.registerHealthCheck`: 1 call site.
- `context.registerStartHook`: 1 call site.

Frequent Vortex event names:

- `onAsync`: `will-deploy`, `did-deploy`, `will-purge`, `did-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `will-enable-mods`, `added-files`, `download-script-extender`, `update-conflicts-and-rules`, `apply-settings`.
- `events.on`: `gamemode-activated`, `did-install-mod`, `profile-will-change`, `profile-did-change`, `mod-enabled`, `mods-enabled`, `will-install-dependencies`, `autosort-plugins`, `collection-postprocess-complete`, `set-plugin-list`, `did-update-masterlist`, `mod-content-changed`.
- `emitAndAwait`: `purge-mods-in-path`, `deploy-single-mod`, `browse-for-download`, `nexus-download`, `discover-tools`, `bake-settings`, `unfulfilled-rules`.
- Literal `context.registerAPI` names found in source: `lootSortAsync`, `isBlueprintPlugin`, `ummAddGame`, `bepinexAddGame`, `getHashVersion`, `qbmsRegisterGame`, `qbmsList`, `qbmsExtract`, `qbmsWrite`, and `qbmsReimport`.
- Direct `context.api.*` usage is dominated by Redux state reads/writes (`store`, `getState`), event bus use (`events`, `onAsync`, `onStateChange`, `emitAndAwait`), notifications/dialogs (`sendNotification`, `showErrorNotification`, `showDialog`), stylesheets, metadata lookup/save, directory selection, and `addMetaServer`.

## Current DMM SDK Coverage

DMM currently has first-party Go extension registration for:

- Games, Steam app IDs, Nexus domains, Vortex game IDs, source references, and Steam Workshop action declarations.
- Install platforms, target roots, mod types, install planners, custom installer match/build hooks, installer choices, FOMOD-like choice persistence, and source-aware captured installs.
- Runtime requirements, runtime metadata dependencies, launch tools, dynamic launch inputs/arguments, and game version providers.
- Plugin activation, unmanaged markers, conflict ignores, deploy ignores, packed archive mutation declarations, merge/load-order summaries, and lifecycle event handlers.
- Deploy lifecycle execution for `will-deploy`, `did-deploy`, `will-purge`, `did-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `did-install-mod`, `will-enable-mods`, `mod-enabled`, `mods-enabled`, `profile-will-change`, and `profile-did-change` in the current product path.
- Extension-declared archive type, game store, game version provider, and extension API metadata can now report `ready`, `metadata`, or `blocked` status in `/api/extensions` without pretending that unimplemented engines or platform integrations are executable.
- Extension-declared Vortex `registerGame` metadata now covers executable path, required files, static or dynamic query-mod-path signals, merge mode, cleanup requirement, stop patterns, compatible download domains, and environment key/value metadata.
- Vortex `supportedTools` are modeled separately from DMM primary/wrapper launch tools, so external tools such as FO4Edit, Wrye Bash, Creation Kit, Hammer, and Witcher Script Merger no longer pollute launch-option decisions.
- Vortex `requiresLauncher` can be represented as source-backed launcher requirement metadata for store-specific launcher facts. Runtime application is still a separate capability.
- Extension-declared UI/state registration surfaces now exist for source-backed metadata: actions, action checks, control wrappers, dialogs, dashlets, main pages, table attributes, load-order pages, profile files, reducers, persistors, history stacks, start hooks, settings, tests, todos, state migrations, health checks, attribute extractors, and generic game-info providers.
- Every bundled Vortex game extension now has a source-backed DMM counterpart. Rich MVP targets remain dedicated first-party Go game extensions, while `internal/extensions/vortexgamecatalog` covers the remaining Vortex games as metadata/research-blocked entries with verified game IDs, Nexus domains, Steam app IDs where Vortex declares them, source links, and blocked installer/mod-type/load-order capability summaries.
- Extension capability summaries exposed through `/api/extensions` and persisted non-behavioral snapshots.

This is enough for the current Stardew Valley vertical slice and several partial source-backed extensions. It is not enough to claim full Vortex extension-framework parity.

## MVP-Critical Gaps

### 1. Game Registration Parity

Vortex source examples:

- `games/game-fallout4/src/index.js`
- `games/game-skyrimse/src/index.js`
- `games/game-witcher3/src/index.ts`
- `games/game-baldursgate3/src/index.tsx`
- `games/game-xrebirth/src/index.ts`

Vortex game registrations commonly declare `queryModPath`, `requiredFiles`, `supportedTools`, `requiresLauncher`, `setup`, `getGameVersion`, `mergeMods`, `details.stopPatterns`, `environment`, logos, and platform/store-specific executable behavior.

DMM status:

- Supports Steam app IDs, Nexus domains, Vortex game IDs, target roots, primary/wrapper launch tools, external supported tools, game version providers, installer specs, stop folders in installer-choice specs, and deployment defaults.
- `GameRegistration` now has source-backed fields for Vortex executable path, required files, query-mod-path metadata, merge mode, cleanup requirement, stop patterns, compatible download domains, and environment metadata.
- Source-backed Vortex game catalog entries can now represent Vortex `registerGameStub` separately from normal `registerGame` entries that have no verified Steam app ID, so DMM does not invent Steam ownership for Windows/Epic/registry-only Vortex discovery paths.
- Fallout 4, Skyrim SE, Witcher 3, X Rebirth, Baldur's Gate 3, Factorio, Skyrim VR, and Team Fortress 2 now expose verified `registerGame` metadata or supported-tool metadata where the scanned Vortex source has static facts.
- Remaining gap: no first-class setup/prep action runtime equivalent to Vortex `setup`, though setup can be advertised through `GameSetups`/blocked source metadata.
- Remaining gap: `requiresLauncher` is represented as metadata, but store-specific launcher/runtime application is not yet executed outside existing Decky Steam launch-option requests.
- Generic installers can now use source-backed `details.stopPatterns` matching, with case-insensitive evaluation matching Vortex helper behavior.
- Remaining gap: no explicit multi-logical-game-per-Steam-app resolver. Vortex uses this for cases such as XCOM 2/War of the Chosen and Divinity: Original Sin 2 variants, while DMM currently maps one app ID to one active extension.

Priority: P0 for more game conversions.

### 2. Installer And Mod-Type Parity

Vortex source examples:

- `games/game-stardewvalley/src/index.ts`
- `games/game-fallout4/src/index.js`
- `games/game-skyrimse/src/index.js`
- `games/game-witcher3/src/installers.ts`
- `games/game-baldursgate3/src/installers.ts`
- `games/game-xrebirth/src/installers.ts`

DMM status:

- Supports declarative and custom Go installer rules, mod types, archive-root installs, common-root stripping, generated files, target policies, metadata extractors, custom builders, FOMOD, and generic component choices.
- Supports source-backed Vortex `declareInstallers`-style table matchers for file suffixes, regex patterns, game stop patterns, common-root stripping, and archive-root copy planning.
- X Rebirth is promoted from catalog metadata into `internal/extensions/xrebirth`, covering its Vortex `content.xml` custom installer plus table-declared savegame, shader injector, utility, drop-in, save-patch, and documentation installers.
- Elex is promoted from catalog metadata into `internal/extensions/elex`, covering the Vortex `.pak` installer, `data/packed` mod root, required executable metadata, and Vortex's FOMOD exclusion.
- Torchlight II is promoted from catalog metadata into `internal/extensions/torchlight2`, covering Vortex's `.mod` installer, documents mods target root, Linux `ModLauncher.bin.x86` executable metadata, and Steam launcher requirement metadata.
- Galactic Civilizations III is promoted from catalog metadata into `internal/extensions/galacticcivilizations3`, covering Vortex's documents-root mod path selection, broad archive copy installer, `.faction` routing, Crusade mod-type metadata, and in-game enable reminder.
- A Hat in Time is promoted from catalog metadata into `internal/extensions/ahatintime`, covering Vortex `modinfo.ini` archive root handling and supported Modding Tools metadata.
- GreedFall is promoted from catalog metadata into `internal/extensions/greedfall`, covering Vortex datalocal wrapper stripping, FOMOD exclusion, executable version metadata, and a did-deploy hook that refreshes managed target timestamps after deployment.
- Surviving Mars is promoted from catalog metadata into `internal/extensions/survivingmars`, covering Vortex `modcontent.hpk` archive root handling and the AppData mods target root adapted to Steam Proton compatdata on Deck.
- Daggerfall Unity is promoted from catalog metadata into `internal/extensions/daggerfallunity`, covering Vortex `.dfmod` handling, Windows/no-platform payload filtering, and `DaggerfallUnity_Data/StreamingAssets` deployment.
- Sekiro is promoted from catalog metadata into `internal/extensions/sekiro`, covering Vortex loose `.partsbnd.dcx` handling, root `parts` package handling, and Mod Engine presence diagnostics.
- Dawn of Man is promoted from catalog metadata into `internal/extensions/dawnofman`, covering Vortex scenario `.scn.xml` installs into Documents scenarios, UMM-style `Info.json` installs into game `Mods`, and blocked UMM setup parity metadata.
- Team Fortress 2 is promoted from catalog metadata into `internal/extensions/teamfortress2`, covering Vortex `.vpk` archive handling into `tf/custom`, Hammer supported-tool metadata, and `tf/steam.inf` ClientVersion discovery.
- Bloodstained: Ritual of the Night is promoted from catalog metadata into `internal/extensions/bloodstainedritualofthenight`, covering Vortex `.pak` archive handling, the `BloodstainedRotN/Content/Paks/~mods` target root, and the alphabetic load-order prefix rewrite Vortex uses because the game loads paks by folder name. Vortex unmanaged-file import and migration surfaces remain blocked metadata.
- Code Vein is promoted from catalog metadata into `internal/extensions/codevein`, covering Vortex `.pak` archive handling, the `CodeVein/content/paks/~mods` target root, executable version metadata, and the alphabetic load-order prefix rewrite Vortex uses because the game loads paks by folder name. Vortex unmanaged-file import and migration surfaces remain blocked metadata.
- Mount & Blade is promoted from catalog metadata into `internal/extensions/mountandblade`, covering the three Vortex-registered variants, `module.ini` module-package installs, supported loose override file routing into each game's native module, and native module version discovery.
- Star Wars: Knights of the Old Republic and KOTOR II are promoted from catalog metadata into `internal/extensions/kotor`, covering Vortex game-root folder installs, default override-folder installs, KOTOR II Steam launcher metadata, and explicit blocked TSLPatcher utility/mod installers.
- Neverwinter Nights is promoted from catalog metadata into `internal/extensions/neverwinter`, covering classic NWN and NWN: Enhanced Edition loose extension routing, override-folder preservation, structured Neverwinter folder archives, and the EE Documents mod root. Neverwinter Nights 2 is covered in the same package for Vortex `.mod` module archives and Documents module/override target roots.
- Factorio, No Man's Sky, The Witcher, and The Witcher 2 are promoted from catalog metadata into `internal/extensions/factorio`, `internal/extensions/nomanssky`, and `internal/extensions/witcherlegacy`. This covers Vortex-verified external/default mod roots, user-content mod-type routing, No Man's Sky `.pak` and `.dll` mod-type routing, Steam executable metadata, and setup metadata. No Man's Sky's setup rename and historical migration remain metadata-only until DMM has executable setup/migration runtime.
- Total War: Three Kingdoms is promoted from catalog metadata into `internal/extensions/totalwarthreekingdoms`, covering Vortex `.pack` folder-copy handling, `data` deployment, Assembly Kit supported tools, launcher metadata, setup metadata, and a deploy reminder.
- War Thunder and World of Tanks are promoted from catalog metadata into `internal/extensions/warthunder` and `internal/extensions/worldoftanks`. This covers War Thunder skin/audio mod-type routing, World of Tanks `version.xml`-derived `res_mods/<version>` targeting, and default archive-root deployment. War Thunder's setup-time `config.blk` audio toggle remains blocked until executable setup/patch-existing support exists.
- Darkest Dungeon is promoted from catalog metadata into `internal/extensions/darkestdungeon`, covering Vortex `project.xml` installers, generated `project.xml` installers for no-project archives, game-directory structure matching through the generic custom-planner `GamePath` input, hero portrait routing, setup metadata, and store launcher metadata.
- Dragon's Dogma is promoted from catalog metadata into `internal/extensions/dragonsdogma`, covering Vortex nativePC archive routing and the Vortex invalid-archive confirmation installer. Selective MT Framework ARC merging and historical staged-mod migration remain blocked metadata until DMM has the shared ARC engine/migration runtime.
- Blade & Sorcery is promoted from catalog metadata into `internal/extensions/bladeandsorcery`, covering Vortex official `manifest.json` installer routing, engine-injection routing through `dinput`, obsolete MulleDK19 `mod.json` blocking, supported VR tools, setup metadata, and source-backed load-order/migration/version-validation blocked metadata.
- Monster Hunter: World is promoted from catalog metadata into `internal/extensions/monsterhunterworld`, covering Vortex `nativePC` archive stripping, ReShade `.ini` deployment to the game root with the same ReShade warning, Stracker's Loader root-file routing, setup metadata for `nativePC`, and HunterPie/SmartHunter/MHW Transmog supported-tool metadata.
- Fallout: New Vegas, Fallout 4 VR, and Skyrim VR are promoted from catalog metadata into `internal/extensions/falloutnv`, `internal/extensions/fallout4vr`, and `internal/extensions/skyrimvr`. This covers Vortex `Data` root installers, script-extender installer/launch-tool metadata for NVSE/F4SEVR/SKSEVR, Gamebryo plugin activation metadata, archive invalidation target metadata, Fallout NV 4GB patch `dinput` routing, VR ESL-enabler routing, conflict-ignore metadata where Vortex declares it, and source-backed supported tools. The remaining verified gap is dynamic ESL plugin-support toggling from enabled `eslEnabler` mod metadata, which needs a reusable plugin-activation condition API.
- Source-backed metadata now exists for shared Vortex mod types `dazip`, `dinput`, `enb`, `gedosato`, and `umm`, plus their registered installers where Vortex has them. They are marked `blocked` until the reusable DMM helpers exist.
- Remaining gaps are mostly breadth: more Vortex helper shapes must be represented as reusable SDK helpers rather than copied per game.
- Needed helper APIs include source-backed versions of Vortex `testSupportedContent`, advanced `mergeMods` path transforms, broader wrapper-root normalization variants, and richer component-choice rules.
- Needed shared mod-type helpers include nested DAZIP extraction/submodule planning, executable-relative DLL deployment with unsafe-file confirmation, ENB game-root deployment, GeDoSaTo external tool discovery and texture targeting, and Unity Mod Manager game opt-in/tool discovery.

Priority: P0.

### 3. Archive Type Engines

Vortex source examples:

- `mtframework-arc-support/src/index.ts` registers `arc`.
- `gamebryo-bsa-support/src/index.ts` registers `bsa`.
- `gamebryo-archive-support/src/index.ts` registers `ba2` and `bsa`.
- `quickbms-support/src/index.ts` exposes `qbmsList`, `qbmsExtract`, `qbmsWrite`, and `qbmsReimport`.

DMM status:

- Has generic ZIP/7z/RAR extraction through the download/install pipeline.
- Has MGSV-specific QAR/FPK primitives exposed through extension-owned packed archive mutation.
- Has a generic extension-declared archive type registry and source-backed framework metadata for Vortex `gamebryo-archive-support`, `gamebryo-bsa-support`, `mtframework-arc-support`, and `quickbms-support`.
- BSA, BA2, ARC, and QuickBMS are intentionally exposed as `blocked` capabilities until DMM implements native Deck-safe list/extract/write/reimport engines or verified external-tool bridges.

Priority: P0 for Bethesda/MT Framework/QuickBMS games.

### 4. Lifecycle Event Parity

Vortex source examples:

- `new-file-monitor/src/index.ts`
- `gamebryo-plugin-management/src/index.ts`
- `mod-dependency-manager/src/index.tsx`
- `games/game-stardewvalley/src/runtime/registerRuntimeEvents.ts`
- `games/game-witcher3/src/index.ts`

DMM status:

- Supports extension handlers and generated deploy mappings for deploy/purge paths already wired in core.
- First-class event constants and execution points now exist for `will-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `did-install-mod`, `will-enable-mods`, `mod-enabled`, `mods-enabled`, `profile-will-change`, and `profile-did-change`.
- Still missing execution points for `added-files`, `gamemode-activated`, `will-install-dependencies`, `check-mods-version`, and `update-conflicts-and-rules`.
- Missing a generic request/response bus equivalent to Vortex `emitAndAwait` for extension-to-core operations such as `deploy-single-mod`, `purge-mods-in-path`, `browse-for-download`, `discover-tools`, and `bake-settings`.

Priority: P0 for parity and safe profile transitions.

### 5. Extension UI Actions, Settings, Diagnostics, And Tests

Vortex source examples:

- `gamebryo-plugin-management/src/index.ts`
- `mod-dependency-manager/src/index.tsx`
- `gamebryo-savegame-management/src/index.ts`
- `fnis-integration/src/index.ts`
- `games/game-baldursgate3/src/index.tsx`
- `games/game-stardewvalley/src/registration/registerUi.ts`

DMM status:

- Has Action Center jobs, Decky notifications, runtime action notices, settings panes, debug logs, and backend diagnostics.
- Has source-backed metadata declarations for Vortex UI/state registration surfaces, including `registerAction`, `registerSettings`, `registerTest`, `registerHealthCheck`, `registerDialog`, `registerMainPage`, `registerDashlet`, `registerTableAttribute`, `registerLoadOrderPage`, `registerProfileFile`, `registerReducer`, `registerPersistor`, and `registerStartHook`.
- Missing the generic runtime renderer/executor for those declarations. Converted extensions can advertise blocked UI/state capabilities now, but DMM cannot yet render arbitrary extension dialogs/pages/table attributes or execute arbitrary extension actions.

Priority: P1. Critical for polish and advanced parity, but less urgent than install/deploy correctness.

### 6. Cross-Extension API And State

Vortex source examples:

- `quickbms-support/src/index.ts` registers QuickBMS APIs.
- `modtype-bepinex/src/index.ts` registers `bepinexAddGame`.
- `modtype-umm/src/index.ts` registers `ummAddGame`.
- `gamebryo-plugin-management/src/index.ts` registers LOOT/plugin APIs.
- `gameversion-hash/src/index.ts` registers `getHashVersion`.

DMM status:

- First-party Go extensions can share normal Go packages, but there is no explicit registered extension API namespace, dependency graph, or import contract.
- Source-backed metadata exists for Vortex `quickbms-support` APIs and `gameversion-hash`'s `getHashVersion` API, but both are blocked until executable/runtime behavior is implemented.
- Source-backed metadata exists for Vortex `modtype-umm`'s `ummAddGame` API, but it is blocked until DMM has a typed Unity Mod Manager helper/API that converted game extensions can call.
- Source-backed metadata exists for shared-system APIs/events used by FNIS, local game settings, dependency management, new-file monitoring, and Vortex test helpers: `deploy-single-mod`, `purge-mods-in-path`, `browse-for-download`, `discover-tools`, `bake-settings`, `unfulfilled-rules`, `registerGameInfoProvider`, and new-file adoption.
- Missing runtime implementations for extension-owned persistent state/persistor/migration behavior equivalent to Vortex `registerReducer`, `registerPersistor`, and `registerMigration`.

Priority: P1 for full Vortex parity; P0 only when a converted MVP game needs cross-extension behavior.

### 7. Game Store Discovery

Vortex source examples:

- `gamestore-gog/src/index.ts`
- `gamestore-origin/src/index.ts`
- `gamestore-uplay/src/index.ts`
- `gamestore-xbox/src/index.ts`
- `gameinfo-steam/src/index.ts`

DMM status:

- Steam discovery is implemented in core for the Steam Deck MVP.
- Source-backed framework metadata exists for GOG, Origin/EA, Ubisoft/Uplay, and Xbox game-library discovery, marked blocked because the verified Vortex implementations depend on Windows clients, registry state, or Xbox shell commands.
- This should remain outside the Steam Deck MVP path unless the user explicitly broadens discovery beyond Steam Deck Steam libraries.

Priority: P2 for MVP, P0 after Steam-only MVP.

### 8. Load Order, Merge, Dependencies, And Rule Editors

Vortex source examples:

- `gamebryo-plugin-management/src/index.ts`
- `mod-dependency-manager/src/index.tsx`
- `games/game-witcher3/src/index.ts`
- `games/game-baldursgate3/src/loadOrder.ts`
- `games/game-codevein/src/index.ts`
- `games/game-spyroreignitedtrilogy/src/index.ts`
- `games/game-xcom2/src/index.js`

DMM status:

- Has load-order summaries, file winner overrides, basic plugin activation generation, Unreal sortable PAK helper, Witcher `mods.settings` generation, and some Gamebryo plugin activation support.
- Missing LOOT/masterlist/userlist behavior, rule graph editing, dependency conflict graph, cycles, group editors, index locks, custom load-order pages, and extension-declared load-order UI models.

Priority: P0 for Bethesda parity, P1 for broader user-facing polish.

### 9. Imports, File Monitoring, And Savegame/Profile Utilities

Vortex source examples:

- `new-file-monitor/src/index.ts`
- `nmm-import-tool/src/index.ts`
- `mo-import/src/index.ts`
- `gamebryo-savegame-management/src/index.ts`
- `local-gamesettings/src/index.ts`

DMM status:

- Has DMM-owned VFS, staging, deployment manifests, rollback, and local archive import.
- Has source-backed blocked metadata for new-file adoption/monitoring, Vortex/NMM/MO import UI entries, savegame profile features, and local game settings baking.
- Missing the actual runtime implementations for those features.

Priority: P2 for MVP unless needed to safely adopt dirty Vortex installs.

## Implementation Order

1. Add runtime execution only for the subset needed by current MVP games, starting with lifecycle event parity around profile/mod install, enable, remove, purge, and deploy operations.
2. Implement generic planner helpers for source-backed `testSupportedContent`, broader wrapper-root normalization, component-choice rules, and merge-mode path transforms.
3. Convert shared Vortex framework extensions into DMM shared helpers before converting more game extensions: common interpreters, archive engines, Gamebryo archive/plugin helpers, QuickBMS, BepInEx, UMM, dependency/rule primitives, and game-version helpers.
4. Promote catalog entries into dedicated game extensions when a target game becomes MVP-critical, implementing the missing reusable SDK/runtime capability first and keeping every game-specific rule inside that game extension package.
5. Add a multi-logical-game-per-app resolver before converting Vortex extensions that register multiple selectable game IDs against one Steam app.
6. Only mark an extension as parity-complete when source-reviewed behavior is implemented, tested lightly, and exposed in `/api/extensions` with enough detail to audit support.

## Guardrails

- Do not add game-specific branches to generic server/storage/deploy/installplan code to close a single extension gap.
- Do not mark metadata-only or blocked extensions as installer-supported.
- Do not add a Vortex API name as a no-op just to check off parity. If a capability is registered but not executable yet, label it as descriptive metadata or blocked support in the extension summary.
- When an engine is required, implement the reusable engine first, then have the extension declare it.
