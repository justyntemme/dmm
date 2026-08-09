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
- `context.registerHealthCheck`: 1 call site.
- `context.registerStartHook`: 1 call site.

Frequent Vortex event names:

- `onAsync`: `will-deploy`, `did-deploy`, `will-purge`, `did-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `will-enable-mods`, `added-files`, `download-script-extender`, `update-conflicts-and-rules`, `apply-settings`.
- `events.on`: `gamemode-activated`, `did-install-mod`, `profile-will-change`, `profile-did-change`, `mod-enabled`, `mods-enabled`, `will-install-dependencies`, `autosort-plugins`, `collection-postprocess-complete`, `set-plugin-list`, `did-update-masterlist`, `mod-content-changed`.
- `emitAndAwait`: `purge-mods-in-path`, `deploy-single-mod`, `browse-for-download`, `nexus-download`, `discover-tools`, `bake-settings`, `unfulfilled-rules`.

## Current DMM SDK Coverage

DMM currently has first-party Go extension registration for:

- Games, Steam app IDs, Nexus domains, Vortex game IDs, source references, and Steam Workshop action declarations.
- Install platforms, target roots, mod types, install planners, custom installer match/build hooks, installer choices, FOMOD-like choice persistence, and source-aware captured installs.
- Runtime requirements, runtime metadata dependencies, launch tools, dynamic launch inputs/arguments, and game version providers.
- Plugin activation, unmanaged markers, conflict ignores, deploy ignores, packed archive mutation declarations, merge/load-order summaries, and lifecycle event handlers.
- Deploy lifecycle execution for `will-deploy`, `did-deploy`, `will-purge`, `did-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `did-install-mod`, `will-enable-mods`, `mod-enabled`, `mods-enabled`, `profile-will-change`, and `profile-did-change` in the current product path.
- Extension-declared archive type, game store, game version provider, and extension API metadata can now report `ready`, `metadata`, or `blocked` status in `/api/extensions` without pretending that unimplemented engines or platform integrations are executable.
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

- Supports Steam app IDs, Nexus domains, Vortex game IDs, target roots, tools, game version providers, installer specs, stop folders in installer-choice specs, and deployment defaults.
- Partial gap: no first-class setup/prep action contract equivalent to Vortex `setup`.
- Partial gap: no explicit `requiredFiles`/`environment` declaration on `GameRegistration` for game discovery diagnostics.
- Partial gap: no first-class `requiresLauncher` contract; DMM models the actionable subset through extension-declared launch tools and Decky launch-option requests.
- Partial gap: no generic game details field for stop patterns shared across installers and diagnostics.

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
- Remaining gaps are mostly breadth: more Vortex helper shapes must be represented as reusable SDK helpers rather than copied per game.
- Needed helper APIs include source-backed versions of Vortex-style `queryModPath` defaults, `stopPatterns` matching, `testSupportedContent`, `mergeMods` path transforms, wrapper-root normalization, and component-choice rules.

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
- Missing extension-declared UI action schemas equivalent to `registerAction`.
- Missing extension-declared settings schemas equivalent to `registerSettings`.
- Missing extension-declared tests/health checks equivalent to `registerTest` and `registerHealthCheck`.
- Missing extension-declared dialogs/main pages/table attributes/profile features that can be rendered without custom frontend code.

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
- Missing extension-owned persistent state/persistor/migration contracts equivalent to `registerReducer`, `registerPersistor`, and `registerMigration`.

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
- Missing new-file adoption/monitoring, Vortex/NMM/MO import, savegame profile features, and local game settings baking.

Priority: P2 for MVP unless needed to safely adopt dirty Vortex installs.

## Implementation Order

1. Add missing extension SDK registration types that are safe to model now without claiming behavioral parity: archive types, interpreters, game stores, setup actions, UI actions/settings/tests, profile/collection features, state migrations/persistors, extension APIs, and health checks.
2. Wire new SDK registrations into DMM capability summaries and persisted extension snapshots so every extension can advertise exact support without core game branches.
3. Add runtime execution only for the subset needed by current MVP games, starting with lifecycle event parity around profile/mod install, enable, remove, purge, and deploy operations.
4. Convert shared Vortex framework extensions into DMM shared helpers before converting more game extensions: common interpreters, archive engines, Gamebryo archive/plugin helpers, QuickBMS, BepInEx, UMM, dependency/rule primitives, and game-version helpers.
5. Convert target game extensions against the updated SDK and keep every game-specific rule in that game extension package.
6. Only mark an extension as parity-complete when source-reviewed behavior is implemented, tested lightly, and exposed in `/api/extensions` with enough detail to audit support.

## Guardrails

- Do not add game-specific branches to generic server/storage/deploy/installplan code to close a single extension gap.
- Do not mark metadata-only or blocked extensions as installer-supported.
- Do not add a Vortex API name as a no-op just to check off parity. If a capability is registered but not executable yet, label it as descriptive metadata or blocked support in the extension summary.
- When an engine is required, implement the reusable engine first, then have the extension declare it.
