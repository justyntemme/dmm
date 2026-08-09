# Vortex Extension API Gap Analysis

Source: local Vortex clone `/tmp/dmm-vortex`
Remote: `ssh://git@github.com/Nexus-Mods/Vortex.git`
Source commit: `c57894eb71af8234b58a6bd15ae5ab543eccac3a`
Collected: 2026-08-08
Last refreshed: 2026-08-09

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
- Vortex `supportedTools` are modeled separately from DMM primary/wrapper launch tools, so external tools such as FO4Edit, Wrye Bash, Creation Kit, Hammer, and Witcher Script Merger no longer pollute launch-option decisions. Ready tools with a concrete executable can now queue a Decky-owned Steam launch action; acquisition, tool-specific environment passing, and automated patch/merge execution remain separate capabilities.
- Vortex `requiresLauncher` can be represented as source-backed launcher requirement metadata for store-specific launcher facts. Runtime application is still a separate capability.
- Extension-declared UI/state registration surfaces now exist for source-backed metadata: actions, action checks, control wrappers, dialogs, dashlets, main pages, table attributes, load-order pages, profile features, profile files, reducers, persistors, history stacks, start hooks, settings, tests, todos, state migrations, health checks, attribute extractors, and generic game-info providers.
- Every bundled Vortex game extension now has a source-backed DMM counterpart. Rich MVP targets remain dedicated first-party Go game extensions. `internal/extensions/vortexgamecatalog` is currently empty and remains only as a source-reviewed shim for future generated metadata if a verified Vortex entry cannot yet be promoted to a dedicated extension.
- Extension capability summaries exposed through `/api/extensions` and persisted non-behavioral snapshots.

This is enough for the current Stardew Valley vertical slice and several partial source-backed extensions. It is not enough to claim full Vortex extension-framework parity.

## Mechanical API Matrix

Refreshed from source with `rg` against `/tmp/dmm-vortex/extensions` on 2026-08-09.

| Vortex surface | Source-backed use | DMM surface | Runtime status | Next gap |
| --- | ---: | --- | --- | --- |
| `context.registerGame` | 80 | `RegisterGame` | Partial runtime | Setup/prep execution, richer multi-store discovery, and same-Steam-app logical game selection outside captured installs. |
| `context.registerInstaller` | 70 | `RegisterInstaller` | Partial runtime | More reusable installer helpers for source-backed `testSupportedContent`, wrapper normalization, component choice rules, and packed/archive engines. |
| `context.registerModType` | 40 | `RegisterModType` | Partial runtime | Shared mod-type helpers still missing ENB, GeDoSaTo runtime, DInput confirmation UI, and UMM patch execution. |
| `context.registerAction` | 51 | `RegisterExtensionAction` | Metadata-only | Generic executable extension action contract and UI renderer. |
| `context.registerTest` | 28 | `RegisterExtensionTest` | Metadata-only | Runnable diagnostic/test hooks and trigger wiring. |
| `context.registerReducer` | 27 | `RegisterStateReducer` | Metadata-only | Extension state store/reducer runtime or first-party typed state replacement. |
| `context.registerMigration` | 18 | `RegisterStateMigration` | Partial runtime | DMM executes extension-declared safe purge commands once per game/migration. Non-purge migration commands, adoption migrations, and profile/settings-state migrations remain pending. |
| `context.registerDialog` | 12 | `RegisterExtensionDialog` | Metadata-only | Extension dialog schema/runtime. |
| `context.registerAPI` | 10 | `RegisterExtensionAPI` plus shared Go helpers | Partial runtime | Explicit dependency graph/import contract for cross-extension APIs; QuickBMS and Gamebryo plugin APIs remain blocked. |
| `context.registerTableAttribute` | 9 | `RegisterExtensionTableAttribute` | Metadata-only | Generic extension columns/attributes for phone and Deck surfaces. |
| `context.registerLoadOrder` | 9 | `RegisterLoadOrder` | Partial runtime | Generic load-order models, conflict/rule UI, LOOT-backed Gamebryo sorting engine, and extension-specific load-order page UX. |
| `context.registerGameStub` | 9 | `VortexStub` game registration | Metadata-only | Keep source-backed but unsupported until each game has verified install/runtime support. |
| `context.registerSettings` | 8 | `RegisterExtensionSetting` | Metadata-only | Extension-owned settings schema and persistence. |
| `context.registerDashlet` | 7 | `RegisterExtensionDashlet` | Metadata-only | Product decision whether DMM keeps dashlet concept or maps to Action Center/diagnostics cards. |
| `context.registerMainPage` | 7 | `RegisterExtensionMainPage` | Metadata-only | Generic extension page runtime, likely phone-first only. |
| `context.registerMerge` | 5 | `RegisterMerge` plus shared helpers | Partial runtime | Broader merge helpers beyond current XML/MTL and DAZIP AddIns support. |
| `context.registerInterpreter` | 5 | `RegisterInterpreter` | Metadata-only | Safe external interpreter/tool execution contract. |
| `context.registerArchiveType` | 4 | `RegisterArchiveType` | Metadata-only/blocked | BSA/BA2, ARC, and QuickBMS list/extract/write engines. |
| `context.registerProfileFeature` | 4 | `RegisterProfileFeature` | Metadata-only | Profile file/savegame/settings feature execution. |
| `context.optional.registerCollectionFeature` | 3 | `RegisterCollectionFeature` | Metadata-only | Collection import/postprocess runtime. |
| `context.registerPersistor` | 3 | `RegisterStatePersistor` | Metadata-only | Extension-owned persisted state runtime. |
| `context.registerLoadOrderPage` | 3 | `RegisterExtensionLoadOrderPage` | Metadata-only | Load-order page renderer/editor. |
| `context.registerAttributeExtractor` | 2 | `RegisterAttributeExtractor` | Metadata-only | Runnable metadata extraction hooks outside install planning. |
| `context.registerGameInfoProvider` | 2 | `RegisterGameInfoProvider` | Metadata-only | Runtime polling/caching for game info providers. |
| `context.registerProfileFile` | 2 | `RegisterProfileFile` | Metadata-only | Profile file backup/switching support. |
| `context.registerActionCheck` | 2 | `RegisterExtensionActionCheck` | Metadata-only | Generic action validation hooks. |
| `context.registerControlWrapper` | 1 | `RegisterExtensionControlWrapper` | Metadata-only | UI wrapper/adornment equivalent, likely mapped to DMM row badges/actions. |
| `context.registerHistoryStack` | 1 | `RegisterHistoryStack` | Metadata-only | Extension history/undo state runtime. |
| `context.registerHealthCheck` | 1 | `RegisterHealthCheck` | Metadata-only | Runnable health checks surfaced through diagnostics. |
| `context.registerStartHook` | 1 | `RegisterStartHook` | Metadata-only | Startup hook scheduler with explicit safety boundaries. |

Event and request/response gaps from actual source:

| Vortex event/API | Source-backed use | DMM status | Required DMM work |
| --- | --- | --- | --- |
| `will-deploy`, `did-deploy`, `will-purge`, `did-purge` | deploy lifecycle, new-file monitor, Gamebryo, Witcher, BG3, BepInEx, FNIS | Partial runtime | Keep extending input data; add snapshot/update behavior where Vortex expects mutable installed mod folders. |
| `will-remove-mods`, `did-remove-mod`, `did-remove-profile` | new-file monitor, savegame/profile utilities, Witcher | Partial runtime | Add complete new-file snapshot checks before removals and richer profile/savegame context. |
| `did-install-mod`, `mod-enabled`, `mods-enabled`, `profile-will-change`, `profile-did-change` | BepInEx, Stardew, Witcher, Gamebryo, savegames | Partial runtime | Ensure all user flows emit them consistently, including bulk operations and Deck-only actions. |
| `added-files` / `removed-files` | `new-file-monitor`, BattleTech | Partial runtime | DMM now snapshots extension-owned target roots, detects new unmanaged files before deployment, lets extensions adopt them, and persists adopted manifest entries. Removed-file events and multi-owner user resolution remain missing. |
| `gamemode-activated` | UMM, Gamebryo, BepInEx, Stardew, BG3, QuickBMS, plugin checks | Declared only | Add DMM active-game event source and runnable diagnostics/hooks. |
| `will-install-dependencies` | dependency manager | Missing runtime | Add dependency-install lifecycle before auto acquisition/runtime setup. |
| `check-mods-version` | BG3 | Missing runtime | Add game/extension-driven update compatibility checks. |
| `update-conflicts-and-rules` | dependency manager | Missing runtime | Add dependency/rule conflict recomputation event. |
| `deploy-single-mod` | Stardew config mod, Witcher, FNIS, dependency manager, script extenders | Partial runtime | DMM now supports Vortex's source-verified semantics where the third argument is an optional `enable` flag: `false` removes only that installed mod's current managed files, while the default activates only that mod's manifest mappings. The command preserves profile enable flags and records a new current deployment manifest. Broad command-bus dispatch remains separate work. |
| `purge-mods-in-path` | DAZIP, game migrations, Sims, Spyro, Blade & Sorcery | Partial runtime | DMM now has a scoped purge primitive that removes only latest-manifest-owned files under an absolute path with optional mod-type filtering, runs purge lifecycle hooks, records a new current deployment manifest, and executes extension-declared purge state-migration commands once per game/migration. Non-purge migration commands remain separate work. |
| `browse-for-download`, `nexus-download` | UMM, script extender installer | Missing request/response bus | Route through DMM catalog/captured-download flow with visible user consent. |
| `discover-tools` | script extender installer | Typed DMM primitive | Reports extension-declared game-root tools and DMM-managed tool/script-extender payloads through `/api/games/{appID}/tools`; ready executable tools can queue Decky-owned launch actions through `/api/games/{appID}/tools/{toolID}/launch`. Acquisition, environment passthrough, and automated patch execution remain separate capabilities. |
| `bake-settings` | local game settings | Missing request/response bus | Add profile settings bake runtime. |
| `unfulfilled-rules` | dependency manager | Partial runtime | DMM evaluates extension-extracted metadata dependencies for enabled profile mods, including required/recommended severity, minimum versions, disabled-mod handling, and logical-name aliases. Still missing Vortex's generic `emitAndAwait` override point, install-dependencies flow, rule editor, conflict rules, and cycle handling. |

## Current Vortex Source Audit

Refreshed against `/tmp/dmm-vortex/extensions/games` on 2026-08-09.

- Vortex game extension entry points found: 87.
- Remaining DMM catalog placeholders in `internal/extensions/vortexgamecatalog`: 0.
- Remaining placeholders are not allowed to count as full parity. If a future placeholder is introduced, it must either be promoted into a dedicated DMM game extension or replaced by a documented non-applicable decision.

Remaining placeholder groups from direct source calls:

- Dedicated source-backed `registerGame` ports with remaining blocked runtime gaps: `game-prisonarchitect` maps Vortex LocalAppData mod deployment to Proton LocalAppData and blocks native-Linux mod-path verification; `game-nehrim` preserves Vortex's Nehrim-to-Oblivion facts but blocks install until DMM has a cross-app Steam root resolver.
- Documents/AppData `registerGame` entries promoted with shared DMM target-root support: `game-grimrock`, `game-sims3`, and `game-teso`.
- Classic Gamebryo `registerGame` entries promoted with shared DMM Gamebryo support: `game-fallout3`, `game-oblivion`, and `game-skyrim`.
- Static game-root `registerGame` entries already promoted: `game-darksouls`, `game-grimdawn`, `game-shadowrunreturns`, `game-starbound`, and `game-stateofdecay`.
- Source-backed `registerGameStub` support-mod entries: none remain in the catalog. `game-cyberpunk2077`, `game-dmc5`, `game-mount-and-blade2`, `game-palworld`, `game-re2remake`, `game-re3remake`, `game-starfield`, `game-subnautica`, and `game-subnauticabelowzero` now have dedicated DMM extension packages that preserve Vortex support-mod metadata without claiming installer support.
- Shared BepInEx dependency work: `game-untitledgoose` now has a dedicated source-backed DMM extension with BepInEx installer/runtime metadata.
- DAZIP game entries promoted with a source-backed DMM `dazip` helper: `game-dragonage` and `game-dragonage2`. DMM supports Vortex `dazipOuter` nested `.dazip` extraction, `dazipInner` planning for extracted DAZIP contents, Dragon Age Origins AddIns.xml generation during `will-deploy`, DA2 game-root addins deployment, and the historical DA2 purge migration through extension-declared state-migration commands.
- UMM game entries promoted with a source-backed DMM `umm` helper: `game-gardenpaws`, `game-oni`, `game-pathfinderkingmaker`, and `game-pathfinderwrathoftherighteous`. DMM supports Vortex's Mods-folder deployment, `umm-installer` tool archive staging through a reusable `tool-only` deployment mode, discovery of staged/declared tool payloads, and Decky-owned launch requests for ready executable tools. Unity Mod Manager auto-download, environment-aware launch setup, and patch execution remain blocked.
- Lifecycle and event-bus work: `game-battletech` now uses reusable DMM `added-files` adoption runtime for Vortex's single-owner generated-file flow. `game-untitledgoose` records Vortex's migration/auto-download paths as blocked metadata until DMM has generic purge-migration and auto-acquisition runtimes.
- Merge work: Wolcen XML/MTL payload merging is now source-backed through `internal/extensions/xmlmerge`; Dragon Age Origins AddIns.xml generation is source-backed through the DAZIP helper.

Observed source-backed blocker details:

- `game-battletech/src/index.js` listens to `added-files` and copies single-owner generated files back into that mod's staging folder before removing the unmanaged game file. DMM ports the normal Documents mods installer, version parser, and this single-owner new-file adoption flow through reusable snapshot/adoption runtime plus BattleTech extension logic.
- `game-conanexiles/src/index.js` registers a load-order page and writes `ConanSandbox/Mods/modlist.txt` with staged `.pak` paths in user order. DMM now ports this through `internal/extensions/conanexiles` and the reusable `internal/extensions/loadorderfile` helper.
- `game-divinityoriginalsin2/src/index.js` registers Original and Definitive Edition against Steam app `435150`, writes mods to per-edition Documents folders, and shows a notification after newly deployed `.pak` files. DMM now ports this through `internal/extensions/divinityoriginalsin2`, with source-domain-aware install planning, multi-extension target-root resolution for the shared Steam app, per-edition Proton Documents roots, and the source-backed `.pak` deploy reminder.
- `game-dragonage/src/index.js` requires `modtype-dazip`, registers a DAZIP merge, and merges `manifest.xml` AddIn items into `Settings/AddIns.xml`. DMM now implements the managed DAZIP outer/inner installer flow and AddIns.xml generation path.
- `game-wolcen/src/index.js` registers an XML/MTL merge over the `Game` folder. DMM now ports this through `internal/extensions/wolcen` and the reusable `internal/extensions/xmlmerge` helper, which rewrites XML/MTL mappings during `will-deploy` into generated merged outputs.
- `game-pathfinderkingmaker/src/index.js`, `game-pathfinderwrathoftherighteous/src/index.ts`, `game-gardenpaws/src/index.js`, and `game-oni/src/index.js` require `modtype-umm`. DMM now has a typed source-backed UMM helper, Mods-folder deployment, source-backed Unity Mod Manager tool archive staging, and the generic Decky extension-tool launch queue for these games, but still needs Unity Mod Manager auto-download, environment-aware setup, and patch runtime before claiming full UMM runtime parity.
- `game-untitledgoose/src/index.ts` uses BepInEx setup, an Epic launcher resolver, `bepinexAddGame({ autoDownloadBepInEx: true })`, and a migration. DMM now ports this through `internal/extensions/untitledgoose`, including BepInEx installers/runtime requirements, Epic launcher metadata, setup metadata, the historical purge migration, and blocked to-dos for Epic discovery and BepInEx auto-download.

Implementation priority from this audit:

1. Keep any future `registerGame`/`registerGameStub` placeholders temporary; promote them into dedicated source-backed DMM extension packages before claiming parity.
2. Continue using the generic generated load-order file helper for future games that write ordered text manifests.
3. Add non-purge migration command runtimes for source-backed migrations that rewrite load-order state, adopt unmanaged files, or migrate profile/settings data.
4. Add Unity Mod Manager external-tool download/discovery/patch runtime for the already ported UMM-dependent Unity games.
5. Extend the reusable XML/MTL merge helper as new source-backed merge shapes appear, and add source-backed patch-existing/setup runtime for games that modify existing user/game files outside the current deploy mapping model.
6. Extend new-file monitoring beyond the current BattleTech single-owner `added-files` adoption runtime with removed-file reporting and user resolution for multi-owner candidates.

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
- DMM now supports source-domain-aware install planning and multi-extension target-root lookup for games where Vortex registers multiple logical game IDs under one Steam app, such as Divinity: Original Sin 2. Remaining work is broader UI/runtime selection for same-app variants outside the captured-install path when a future extension needs separate event/tool/version behavior by logical game ID.

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
- Source-backed helpers now exist for Vortex `dazip` outer/inner archive planning/AddIns generation, UMM game opt-in plus tool archive staging metadata, and the shared Vortex `dinput` installer/mod type. DInput archives now route files beside the extension-declared game executable, matching Vortex's executable-relative deployment behavior. ENB and GeDoSaTo remain blocked metadata, plus UMM external tool execution.
- Remaining gaps are mostly breadth: more Vortex helper shapes must be represented as reusable SDK helpers rather than copied per game.
- Needed helper APIs include source-backed versions of Vortex `testSupportedContent`, advanced `mergeMods` path transforms beyond the current XML/MTL helper, broader wrapper-root normalization variants, and richer component-choice rules.
- Needed shared mod-type helpers include DInput unsafe-DLL confirmation UI, ENB game-root deployment, GeDoSaTo texture targeting, and Unity Mod Manager external-tool download/launch/patch execution.

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
- Has a first runtime execution point for `added-files` before deployment planning, modeled on Vortex `new-file-monitor` plus BattleTech adoption.
- Still missing execution points for `removed-files`, `gamemode-activated`, `will-install-dependencies`, `check-mods-version`, and `update-conflicts-and-rules`.
- Missing a generic request/response bus equivalent to Vortex `emitAndAwait` for extension-to-core operations such as `browse-for-download` and `bake-settings`. `deploy-single-mod`, `purge-mods-in-path`, `discover-tools`, and purge state-migration execution have typed DMM runtime primitives, but broad arbitrary command-bus dispatch and non-purge migration execution are still pending.

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
- Source-backed metadata exists for Vortex `quickbms-support` APIs. Vortex `gameversion-hash`'s `getHashVersion` behavior now has a reusable DMM helper: converted game extensions can declare `hashFiles`/`hashDirPath`, DMM computes the Vortex-style chained MD5 hash, and it maps through the Vortex backend hash map when reachable. The Vortex UI/actions for editing hash-map entries remain metadata-only.
- Vortex `modtype-umm`'s `ummAddGame` API now maps to a typed DMM `umm` helper for converted first-party Go extensions. DMM also supports the source-backed `umm-installer` archive shape as a managed `tool-only` staging payload. Tool discovery reports staged DMM-managed tool payloads and extension-declared game-root tools through `/api/games/{appID}/tools`; ready executable tools can be launched by Decky through queued extension-tool actions. Runtime Unity Mod Manager auto-download, environment-aware launch setup, and patch execution are still blocked.
- Source-backed metadata exists for shared-system APIs/events used by FNIS, local game settings, dependency management, new-file monitoring, and Vortex test helpers: `deploy-single-mod`, `purge-mods-in-path`, `browse-for-download`, `discover-tools`, `bake-settings`, `unfulfilled-rules`, `registerGameInfoProvider`, and new-file adoption. `deploy-single-mod`, `purge-mods-in-path`, `discover-tools`, Decky extension-tool launch actions, profile metadata dependency evaluation, and purge state-migration execution are backed by implemented typed runtime primitives; the others remain metadata-only or blocked unless separately noted.
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

- Has load-order summaries, profile-scoped file winner overrides, profile-scoped Gamebryo plugin row enable/order state, Vortex-format Gamebryo `plugins.txt`/`loadorder.txt` generation, extension-declared LOOT game/masterlist IDs, DMM-owned LOOT masterlist/prelude cache paths, profile-local LOOT `userlist.yaml` paths under `loot/{game}/profiles/{profileID}/userlist.yaml`, a refresh endpoint for Vortex's `v0.29` masterlist/prelude URLs, DMM-owned LOOT userlist read/write endpoints, profile-copy seeding for LOOT user rules, a `dmm-loot-sorter.v1` JSON helper contract for real libloot sorting, a sort endpoint that persists profile-scoped mutable plugin order and applies the profile, Advanced UI for plugin rules/group assignments/group order rules, Unreal sortable PAK helper, Witcher `mods.settings` generation, and several generated load-order file helpers.
- Missing the bundled `dmm-loot-sorter` libloot helper binary, live validation against real Bethesda plugin sets, dependency conflict graph evaluation, cycle handling, index locks, full Vortex group editor parity, local/global LOOT rule toggle UX if we decide to expose it, custom load-order pages, and broader extension-declared load-order UI models.

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
5. Extend the multi-logical-game-per-app resolver if a future extension needs logical-game-specific event, tool, version, or UI selection beyond the captured-install and target-root paths now implemented.
6. Only mark an extension as parity-complete when source-reviewed behavior is implemented, tested lightly, and exposed in `/api/extensions` with enough detail to audit support.

## Guardrails

- Do not add game-specific branches to generic server/storage/deploy/installplan code to close a single extension gap.
- Do not mark metadata-only or blocked extensions as installer-supported.
- Do not add a Vortex API name as a no-op just to check off parity. If a capability is registered but not executable yet, label it as descriptive metadata or blocked support in the extension summary.
- When an engine is required, implement the reusable engine first, then have the extension declare it.
