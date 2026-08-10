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
- Runtime requirements, runtime metadata dependencies, extension-declared runtime acquisition sources, acquisition modes/instructions, launch tools, dynamic launch inputs/arguments, and game version providers.
- Plugin activation, unmanaged markers, conflict ignores, deploy ignores, packed archive mutation declarations, merge/load-order summaries, explicit duplicate-target conflict detection, profile-scoped file-winner overrides, and lifecycle event handlers.
- Deploy lifecycle execution for `will-deploy`, `did-deploy`, `will-purge`, `did-purge`, `will-remove-mods`, `did-remove-mod`, `did-remove-profile`, `did-install-mod`, `will-enable-mods`, `mod-enabled`, `mods-enabled`, `profile-will-change`, `profile-did-change`, `check-mods-version`, and `update-conflicts-and-rules` in the current product path.
- Extension-declared archive type, game store, game version provider, and extension API metadata can now report `ready`, `metadata`, or `blocked` status in `/api/extensions` without pretending that unimplemented engines or platform integrations are executable. DMM now has a native source-backed BA2/BSA list/read/extract runtime for Gamebryo archives and a native BSA create/write runtime for uncompressed v103/v104/v105 archives; BA2 write remains pending.
- Extension-declared Vortex `registerGame` metadata now covers executable path, preferred executable variants selected from files present under the game root and optional game-path predicates, required files, static or dynamic query-mod-path signals, merge mode, cleanup requirement, stop patterns, compatible download domains, and environment key/value metadata.
- Extension-declared `setup`/prep behavior now has a typed runtime for source-backed actions: require path, ensure directory, ensure file, safe `rename-if-exists` marker transitions, and regex `patch-text` edits for existing text files. Setup actions can target the game root or an extension-declared target root, report read-only state in game diagnostics, run explicitly through `POST /api/games/{appID}/setup`, and auto-run before manual/profile/auto-enable deployment planning. Rich setup functions that compute generated config templates from current user state or perform source-specific state migrations still need dedicated typed action primitives.
- Vortex `supportedTools` are modeled separately from DMM primary/wrapper launch tools, so external tools such as FO4Edit, Wrye Bash, Creation Kit, Hammer, and Witcher Script Merger no longer pollute launch-option decisions. Ready tools with a concrete executable can now queue a Decky-owned Steam launch action. Supported tools may declare preferred executable variants that are selected when their required files exist under the game root, matching source-verified Vortex tools such as BodySlide x64 with a BodySlide.exe fallback. Supported tools may also declare source-backed acquisition sources that route through the shared captured-install pipeline or through a Decky browser handoff with extension-provided instructions; extension actions can now target those tool acquisitions through a typed `acquire-tool` action without adding game-specific server code. Tool-specific environment passing and automated patch/merge execution remain separate capabilities.
- Vortex `requiresLauncher` can be represented as source-backed launcher requirement metadata for store-specific launcher facts. Steam launcher requirements matching the discovered Steam app now appear as satisfied diagnostics; non-Steam launcher application remains a separate capability.
- Extension-declared UI/state registration surfaces now exist for source-backed metadata: actions, action checks, control wrappers, dialogs, dashlets, main pages, table attributes, load-order pages, reducers, persistors, history stacks, settings, tests, todos, state migrations, health checks, attribute extractors, and start hooks. Mod health checks can now run through a typed hook fed by DMM staged manifests and surface warnings in game diagnostics. The one source-backed Vortex `registerStartHook` use is now represented by a typed startup hook that checks discovered games for unresolved duplicate managed file targets and queues a normal Action Center notice when a file winner is required. Profile features now also have per-profile boolean state storage/API, copy-on-profile-create behavior, extension-bound validation, and phone/tablet advanced toggles. Profile files now have extension-declared base/path metadata, a profile read model, Steam Deck Proton LocalAppData/Documents path resolution, and Vortex-style profile-switch copy/backup/restore execution for extension-declared `local_game_settings` files. DMM now models Vortex `bake-settings` as a profile-switch lifecycle event with profile-scoped enabled mods sorted by priority/name, so extensions can generate profile-local settings around the file swap without core game-specific code. Source-backed `gamebryo-test-settings` coverage now includes Oblivion `Oblivion.ini` missing-font detection, Vortex-equivalent automatic Oblivion font repair through extension-owned diagnostic repair hooks, and Skyrim-family `fontconfig.txt` validation against `Skyrim - Interface.bsa`. Extension settings now have a typed JSON/string/path/bool/number schema, extension-bound validation, HTTP API, change events, and ready settings can be global or profile-scoped. Profile-scoped settings are stored per profile, copied when profiles are cloned, merged into profile-aware extension hook inputs, and exposed through profile routes so Vortex reducer-style profile state can drive runtime behavior without global leakage. Applying those settings remains capability-specific work. Generic game-info providers now have executable provider hooks and a `/api/games/{appID}/info` read model for priority-ordered detail rows.
- Every bundled Vortex game extension now has a source-backed DMM counterpart. Rich MVP targets remain dedicated first-party Go game extensions. `internal/extensions/vortexgamecatalog` is currently empty and remains only as a source-reviewed shim for future generated metadata if a verified Vortex entry cannot yet be promoted to a dedicated extension.
- Extension capability summaries exposed through `/api/extensions` and persisted non-behavioral snapshots.

This is enough for the current Stardew Valley vertical slice and several partial source-backed extensions. It is not enough to claim full Vortex extension-framework parity.

## Mechanical API Matrix

Refreshed from source with `rg` against `/tmp/dmm-vortex/extensions` on 2026-08-09.

| Vortex surface | Source-backed use | DMM surface | Runtime status | Next gap |
| --- | ---: | --- | --- | --- |
| `context.registerGame` | 80 | `RegisterGame` | Partial runtime | Richer multi-store discovery, same-Steam-app logical game selection outside captured installs, and setup actions beyond require/ensure/file/rename/text-patch primitives. |
| `context.registerInstaller` | 70 | `RegisterInstaller` | Partial runtime | More reusable installer helpers for source-backed `testSupportedContent`, wrapper normalization, component choice rules, and packed/archive engines. |
| `context.registerModType` | 40 | `RegisterModType` | Partial runtime | Shared mod-type helpers still missing any source-backed ENB runtime. DInput trust confirmation, GeDoSaTo texture targeting, and UMM tool/runtime acquisition are implemented through generic primitives. |
| `context.registerAction` | 51 | `RegisterExtensionAction` | Partial runtime | Typed open-directory actions now queue backend-resolved paths for Decky-owned execution, and typed acquire-tool actions route extension-declared supported-tool packages through the shared captured-install pipeline. Ready typed actions are exposed through the selected-game Decky surface and phone/tablet Review view. Remaining gaps are generic action UI schemas beyond those typed actions, action checks, and arbitrary extension callbacks. |
| `context.registerTest` | 28 | `RegisterExtensionTest` | Partial runtime | DMM can run extension-owned typed diagnostic tests from game diagnostics with game/profile/mod context and surface warnings. `gamemode-activated` tests run when Decky reports the active game and queue Action Center notices for non-passing results. Other triggers remain pending. |
| `context.registerReducer` | 27 | `RegisterStateReducer` plus first-party typed state replacements | Partial runtime | Some Vortex reducer fields now map to typed DMM runtime state instead of generic reducer blobs; for example BG3 `autoExportLoadOrder` is a ready typed extension setting with Vortex's default `true`. Remaining reducer-backed fields need either a concrete DMM runtime replacement or an extension-owned state store/persistor runtime. |
| `context.registerMigration` | 18 | `RegisterStateMigration` | Partial runtime | DMM executes extension-declared safe purge commands, manifest `set-mod-type` retag/retarget commands, and active-profile redeploy commands once per game/migration, gated by the previous persisted extension snapshot so historical migrations do not run on fresh installs. Adoption migrations and profile/settings-state migrations remain pending. |
| `context.registerDialog` | 12 | `RegisterExtensionDialog` | Metadata-only | Extension dialog schema/runtime. |
| `context.registerAPI` | 10 | `RegisterExtensionAPI` plus shared Go helpers | Partial runtime | Explicit dependency graph/import contract for cross-extension APIs; QuickBMS, Gamebryo plugin APIs, UMM helpers, and game-version hash helpers now have typed first-party Go namespaces or helpers where Vortex uses cross-extension APIs. Broader arbitrary extension API dispatch remains pending. |
| `context.registerTableAttribute` | 9 | `RegisterExtensionTableAttribute` | Metadata-only | Generic extension columns/attributes for phone and Deck surfaces. |
| `context.registerLoadOrder` | 9 | `RegisterLoadOrder` | Partial runtime | Generic load-order models, conflict/rule UI, LOOT-backed Gamebryo sorting engine, and extension-specific load-order page UX. |
| `context.registerGameStub` | 9 | `VortexStub` game registration | Metadata-only | Keep source-backed but unsupported until each game has verified install/runtime support. |
| `context.registerSettings` | 8 | `RegisterExtensionSetting` | Partial runtime | Generic value persistence/API/events, typed JSON/string/path/bool/number schema validation, default values, and phone/tablet rendering are implemented for registered ready settings. Extension tests, target-root resolvers, and deployment/lifecycle events now receive defaults plus persisted overrides. Vortex `apply-settings` behavior for Gamebryo archive invalidation is represented through extension-declared profile-file INI patches. Remaining gaps are other concrete setting effects and richer Vortex-equivalent custom setting components. |
| `context.registerDashlet` | 7 | `RegisterExtensionDashlet` | Metadata-only | Product decision whether DMM keeps dashlet concept or maps to Action Center/diagnostics cards. |
| `context.registerMainPage` | 7 | `RegisterExtensionMainPage` | Metadata-only | Generic extension page runtime, likely phone-first only. |
| `context.registerMerge` | 5 | `RegisterMerge` plus shared helpers | Partial runtime | Broader merge helpers beyond current XML/MTL and DAZIP AddIns support. |
| `context.registerInterpreter` | 5 | `RegisterInterpreter` | Partial runtime | DMM resolves extension-declared interpreter commands/arguments by file extension and platform through `/api/interpreters/resolve`; actual execution remains a Decky/user action boundary. |
| `context.registerArchiveType` | 4 | `RegisterArchiveType` plus native archive runtimes | Partial runtime | BSA/BA2 list/read/extract and BSA write/create are implemented. QuickBMS and ARCtool now have typed process bridges matching Vortex's external-helper model; Dragon's Dogma wires ARCtool through an extension-owned selective ARC merge hook. BA2 write remains pending. |
| `context.registerProfileFeature` | 4 | `RegisterProfileFeature` | Partial runtime | Per-profile boolean state persists and copies with profiles. Vortex `local_game_settings` profile-file execution is implemented for declared files; savegame feature execution and generic feature runtimes remain pending. |
| `context.optional.registerCollectionFeature` | 3 | `RegisterCollectionFeature` | Metadata-only | Collection import/postprocess runtime. |
| `context.registerPersistor` | 3 | `RegisterStatePersistor` | Metadata-only | Extension-owned persisted state runtime. |
| `context.registerLoadOrderPage` | 3 | `RegisterExtensionLoadOrderPage` | Metadata-only | Load-order page renderer/editor. |
| `context.registerAttributeExtractor` | 2 | `RegisterAttributeExtractor` | Partial runtime | Source-backed attributes can be extracted during extension-owned install planning, including Stardew SMAPI manifest metadata and Pillars II `SupportedGameVersion` ranges. A generic post-install extractor scheduler remains pending. |
| `context.registerGameInfoProvider` | 2 | `RegisterGameInfoProvider` | Partial runtime | Ready providers can run through `/api/games/{appID}/info` with source-shaped priority/tags/cache metadata, and provider detail rows now render in the phone/tablet Review view plus the selected-game Decky surface. Remaining gaps are persistent TTL caching and a Steam Web API detail provider. |
| `context.registerProfileFile` | 2 | `RegisterProfileFile` | Partial runtime | Extension-declared files resolve through `/api/profiles/{id}/files`; profile-switch backup/copy/restore runtime is implemented for files marked `sync_on_profile_switch` and gated by an extension-declared feature such as `local_game_settings`. Generic savegame/profile-file feature runtimes remain pending. |
| `context.registerActionCheck` | 2 | `RegisterExtensionActionCheck` plus typed LOOT validation | Partial runtime | Gamebryo duplicate userlist-rule validation is enforced in the LOOT userlist write path. The generic Redux-style action-check runtime remains blocked. |
| `context.registerControlWrapper` | 1 | `RegisterExtensionControlWrapper` | Metadata-only | UI wrapper/adornment equivalent, likely mapped to DMM row badges/actions. |
| `context.registerHistoryStack` | 1 | `RegisterHistoryStack` | Metadata-only | Extension history/undo state runtime. |
| `context.registerHealthCheck` | 1 | `RegisterHealthCheck` | Partial runtime | Typed mod health checks now run from installed staged manifests and surface warnings through diagnostics. Non-mod/global health checks remain metadata-only until source requires them. |
| `context.registerStartHook` | 1 | `RegisterStartHook` | Partial runtime | The source-backed dependency-manager startup conflict check now runs once after game discovery and queues/cancels Action Center notices for unresolved duplicate managed file targets. Arbitrary extension startup callbacks remain blocked by design. |

Event and request/response gaps from actual source:

| Vortex event/API | Source-backed use | DMM status | Required DMM work |
| --- | --- | --- | --- |
| `will-deploy`, `did-deploy`, `will-purge`, `did-purge` | deploy lifecycle, new-file monitor, Gamebryo, Witcher, BG3, BepInEx, FNIS | Partial runtime | Event handlers now receive staged file and metadata summaries for installed mods plus persisted extension settings. BG3 uses this with the Divine process bridge to generate `modsettings.lsx` from enabled pak metadata during deployment. DMM now has generic wait-for-tool-exit, backend-prepared tool input files, and generated profile-mod primitives for external generated-output tools. FNIS source support uses those primitives to queue `GenerateFNISForUsers.exe`, write/remove `MyPatches.txt` from profile settings, wait for the tool to exit, and record/deploy the resulting generated profile mod. Keep extending input data; add checksum/update/load-after behavior where Vortex expects mutable installed mod folders. |
| `will-remove-mods`, `did-remove-mod`, `did-remove-profile` | new-file monitor, savegame/profile utilities, Witcher | Partial runtime | Add complete new-file snapshot checks before removals and richer profile/savegame context. |
| `did-install-mod`, `mod-enabled`, `mods-enabled`, `profile-will-change`, `profile-did-change` | BepInEx, Stardew, Witcher, Gamebryo, savegames | Partial runtime | Captured installs and installer-choice/FOMOD applies share the `did-install-mod` completion path and have route-level coverage. Continue verifying bulk operations and Deck-only actions emit lifecycle events consistently. |
| `added-files` / `removed-files` | `new-file-monitor`, BattleTech | Partial runtime | DMM now snapshots extension-owned target roots, detects added and removed unmanaged files before deployment, passes Vortex-style candidate owner lists to extension handlers, lets extensions adopt selected added files, persists adopted manifest entries, and exposes removed-file deltas to extension handlers. Generic user-facing unmanaged adoption and concrete removed-file recovery handlers remain missing. |
| `gamemode-activated` | UMM, Gamebryo, BepInEx, Stardew, BG3, QuickBMS, plugin checks | Partial runtime | Decky reports the active Steam app to `/api/games/{appID}/activate`; the backend runs source-registered lifecycle handlers and queues extension-declared supported-tool or runtime acquisitions marked `auto_acquire`. Deploy actions also pause and queue missing runtime acquisitions before applying enabled mods. Runnable diagnostics/hooks for the remaining source use cases are still pending. |
| `will-install-dependencies` | dependency manager | Partial runtime | DMM can queue extension-declared runtime acquisitions through the shared captured-install pipeline from Deck activation, deploy guards, Decky UI, or phone/tablet Review. Still missing Vortex's dependency-install lifecycle, rule editor, conflict rules, and generic dependency download resolution. |
| `check-mods-version` | BG3, UMM, BepInEx, script extender installer | Partial runtime | `POST /api/games/{appID}/mods/check-updates` runs extension-owned `check-mods-version` handlers with installed mod/profile context, queues extension notices, and runs the same extension-declared tool/runtime auto-acquisition pass used by game activation. BG3 now checks Norbyte/lslib stable GitHub releases for LSLib/Divine updates through an extension-owned handler. UMM's source-backed check path is covered by the generic auto-acquire pass because Vortex's handler calls `ensureUMM`, which resolves the fixed source-backed UMM package table. Vortex's current script-extender support map gives every entry a Nexus Mods page, causing the custom handler to exit and defer to generic Nexus update metadata; DMM's provider update path covers that same boundary for Nexus-installed script extenders. Runtime acquisitions now carry version/source identity, and update checks can queue a replacement captured-install for an enabled runtime provider when the extension-declared target no longer matches. BepInEx default Nexus site mod/file identity is treated as current even when DMM's acquisition URL is the GitHub fallback asset. BepInEx force-GitHub-style acquisition now uses extension-declared GitHub latest-release asset patterns and semver constraints, matching Vortex's release-list/asset-regex lookup without adding BepInEx branches to the server. Remaining work is live validation against a safe older runtime-provider fixture and any provider-specific rule recomputation a future verified extension requires. |
| `update-conflicts-and-rules` | dependency manager | Partial runtime | `POST /api/games/{appID}/conflicts-and-rules/recompute` validates the game, runs any extension-owned handler with Vortex's `calculateOverrides` flag, republishes the normal deployment refresh event, and returns the same diagnostics/deploy-preview read model used by the UI. DMM deployment planning now treats duplicate managed file targets as unresolved until a profile file-winner is selected, except where an extension declares a source-backed conflict-ignore pattern. Rule-editor state, cycle solving, override redundancy repair, and broad dependency-manager session state remain missing. |
| `deploy-single-mod` | Stardew config mod, Witcher, FNIS, dependency manager, script extenders | Partial runtime | DMM now supports Vortex's source-verified semantics where the third argument is an optional `enable` flag: `false` removes only that installed mod's current managed files, while the default activates only that mod's manifest mappings. The command preserves profile enable flags and records a new current deployment manifest. Broad command-bus dispatch remains separate work. |
| `purge-mods-in-path` | DAZIP, game migrations, Sims, Spyro, Blade & Sorcery | Partial runtime | DMM now has a scoped purge primitive that removes only latest-manifest-owned files under an absolute path with optional mod-type filtering, runs purge lifecycle hooks, records a new current deployment manifest, and executes extension-declared purge state-migration commands once per game/migration. Manifest `set-mod-type` migration commands retag installed manifests and retarget file mappings through extension-declared mod type roots, and `deploy-profile` lets migrations reapply the active profile after state changes. Broader staged-state and profile/settings migrations remain separate work. |
| `browse-for-download`, `nexus-download` | UMM, script extender installer | Typed DMM primitive | Extension-declared tool/runtime acquisitions now carry mode, expected archive name, instructions, and either a catalog URL or Nexus source game/mod/file IDs. `/api/games/{appID}/tools/{toolID}/acquire` and `/api/games/{appID}/requirements/{requirementID}/acquire` route catalog-resolvable sources through captured installs; `/open-browser` variants publish the Decky browser handoff with bounded instructions. Extension actions can also invoke a declared supported-tool acquisition through the same captured-install path, matching BG3's source-backed Re-install LSLib/Divine toolbar action without adding BG3 code to the server. Generic interception of ordinary non-`nxm://` file downloads inside the browser is still a separate provider/browser gap. |
| `discover-tools` | script extender installer, UMM | Typed DMM primitive | Reports extension-declared game-root tools and DMM-managed tool/script-extender payloads through `/api/games/{appID}/tools`; managed tools inherit default-primary and launch-argument metadata from matching extension declarations; ready executable tools can queue Decky-owned launch actions through `/api/games/{appID}/tools/{toolID}/launch`; extension-declared acquisition sources can queue captured installs through `/api/games/{appID}/tools/{toolID}/acquire` or browser handoffs through `/open-browser`. Environment passthrough and automated patch execution remain separate capabilities. |
| `bake-settings` | local game settings | Partial runtime | DMM runs a profile-switch lifecycle event before capturing old profile settings and after installing new profile settings, passing profile-scoped enabled mods in priority order. Concrete extension handlers still need to be added where Vortex source uses the bake bus for mod-provided settings. |
| `unfulfilled-rules` | dependency manager | Partial runtime | DMM evaluates extension-extracted metadata dependencies for enabled profile mods, including required/recommended severity, minimum versions, disabled-mod handling, logical-name aliases, and extension-provided handlers that can mark an unfulfilled rule as handled. Duplicate managed file targets are surfaced as blocking conflicts until resolved through profile file-winner overrides or extension conflict-ignore metadata. Still missing Vortex's install-dependencies flow, rule editor, conflict-rule graph, and cycle handling. |

## Current Vortex Source Audit

Refreshed against `/tmp/dmm-vortex/extensions/games` on 2026-08-09.

- Vortex game extension entry points found: 87.
- Remaining DMM catalog placeholders in `internal/extensions/vortexgamecatalog`: 0.
- Remaining placeholders are not allowed to count as full parity. If a future placeholder is introduced, it must either be promoted into a dedicated DMM game extension or replaced by a documented non-applicable decision.

Remaining placeholder groups from direct source calls:

- Dedicated source-backed `registerGame` ports with remaining blocked runtime gaps: `game-prisonarchitect` maps Vortex LocalAppData mod deployment to Proton LocalAppData and blocks native-Linux mod-path verification. `game-nehrim` now resolves Vortex's Nehrim-to-Oblivion cross-app install root through Steam app manifests and enables the `data` installer with copy deployment.
- Documents/AppData `registerGame` entries promoted with shared DMM target-root support: `game-grimrock`, `game-sims3`, and `game-teso`.
- Classic Gamebryo `registerGame` entries promoted with shared DMM Gamebryo support: `game-enderal`, `game-fallout3`, `game-oblivion`, and `game-skyrim`.
- Static game-root `registerGame` entries already promoted: `game-darksouls`, `game-grimdawn`, `game-shadowrunreturns`, `game-starbound`, and `game-stateofdecay`.
- Source-backed `registerGameStub` support-mod entries: none remain in the catalog. `game-cyberpunk2077`, `game-dmc5`, `game-mount-and-blade2`, `game-palworld`, `game-re2remake`, `game-re3remake`, `game-starfield`, `game-subnautica`, and `game-subnauticabelowzero` now have dedicated DMM extension packages that preserve Vortex support-mod metadata without claiming installer support.
- Shared BepInEx dependency work: `game-untitledgoose` now has a dedicated source-backed DMM extension with BepInEx installer/runtime metadata and extension-declared runtime acquisition for the Vortex default BepInEx package. Converted BepInEx game packages now also declare source-backed acquisition where Vortex opts into it: Citizen Sleeper uses the default BepInEx package, Hollow Knight uses the pinned `5.4.23.5` GitHub release asset plus a Vortex-style latest-release asset pattern/constraint, and Disco Elysium uses the verified bleeding-edge IL2CPP direct archive from `builds.bepinex.dev`. Update checks now compare enabled managed runtime-provider mods against the extension-declared acquisition source/resolved file identity and queue replacement through the normal captured-install update pipeline when the target changes.
- DAZIP game entries promoted with a source-backed DMM `dazip` helper: `game-dragonage` and `game-dragonage2`. DMM supports Vortex `dazipOuter` nested `.dazip` extraction, `dazipInner` planning for extracted DAZIP contents, Dragon Age Origins AddIns.xml generation during `will-deploy`, DA2 game-root addins deployment, and the historical DA2 purge migration through extension-declared state-migration commands.
- UMM game entries promoted with a source-backed DMM `umm` helper: `game-dawnofman`, `game-gardenpaws`, `game-oni`, `game-pathfinderkingmaker`, and `game-pathfinderwrathoftherighteous`. DMM supports Vortex's Mods-folder deployment where the game uses the shared UMM installer, source-specific UMM-style installers where the game extension declares them, `umm-installer` tool archive staging through a reusable `tool-only` deployment mode, source-backed UMM package acquisition, active-game auto-acquire triggers, runtime requirements satisfied by enabled managed UMM provider mods, discovery of staged/declared tool payloads, default-primary metadata for managed UMM tools, and Decky-owned launch requests for ready executable tools. Source review did not find a Vortex-owned noninteractive patch operation; Vortex installs/locates UMM and leaves patch/configuration to UMM's own UI.
- Lifecycle and event-bus work: `game-battletech` now uses reusable DMM `added-files` adoption runtime for Vortex's single-owner generated-file flow. `game-untitledgoose` records Vortex's migration path and Epic discovery limitations as blocked metadata while BepInEx auto-download is now modeled through generic runtime acquisition.
- Merge work: Wolcen XML/MTL payload merging is now source-backed through `internal/extensions/xmlmerge`; Dragon Age Origins AddIns.xml generation is source-backed through the DAZIP helper.

Observed source-backed blocker details:

- `game-battletech/src/index.js` listens to `added-files` and copies single-owner generated files back into that mod's staging folder before removing the unmanaged game file. DMM ports the normal Documents mods installer, version parser, and this single-owner new-file adoption flow through reusable snapshot/adoption runtime plus BattleTech extension logic.
- `game-xrebirth/src/diagnostic.ts` registers three mod health checks over Vortex installed-mod file output and attributes. DMM now runs equivalent extension-owned checks against staged manifest files, mod types, and install metadata, then exposes warnings in diagnostics.
- `game-conanexiles/src/index.js` registers a load-order page and writes `ConanSandbox/Mods/modlist.txt` with staged `.pak` paths in user order. DMM now ports this through `internal/extensions/conanexiles` and the reusable `internal/extensions/loadorderfile` helper.
- `game-divinityoriginalsin2/src/index.js` registers Original and Definitive Edition against Steam app `435150`, writes mods to per-edition Documents folders, and shows a notification after newly deployed `.pak` files. DMM now ports this through `internal/extensions/divinityoriginalsin2`, with source-domain-aware install planning, multi-extension target-root resolution for the shared Steam app, per-edition Proton Documents roots, and the source-backed `.pak` deploy reminder.
- `game-dragonage/src/index.js` requires `modtype-dazip`, registers a DAZIP merge, and merges `manifest.xml` AddIn items into `Settings/AddIns.xml`. DMM now implements the managed DAZIP outer/inner installer flow and AddIns.xml generation path.
- `game-wolcen/src/index.js` registers an XML/MTL merge over the `Game` folder. DMM now ports this through `internal/extensions/wolcen` and the reusable `internal/extensions/xmlmerge` helper, which rewrites XML/MTL mappings during `will-deploy` into generated merged outputs.
- `game-pathfinderkingmaker/src/index.js`, `game-pathfinderwrathoftherighteous/src/index.ts`, `game-gardenpaws/src/index.js`, and `game-oni/src/index.js` require `modtype-umm`. DMM now has a typed source-backed UMM helper, Mods-folder deployment, source-backed Unity Mod Manager tool archive staging, extension-declared acquisition for Vortex's UMM 0.24.2 Nexus/GitHub package, active-game auto-acquire triggers for `autoDownloadUMM`, managed default-primary tool discovery, runtime requirement/provider evaluation, and the generic Decky extension-tool launch queue for these games. Source review shows Vortex's remaining configuration step is inside UMM's own UI, not a Vortex extension patch command.
- `game-untitledgoose/src/index.ts` uses BepInEx setup, an Epic launcher resolver, `bepinexAddGame({ autoDownloadBepInEx: true })`, and a migration. DMM now ports this through `internal/extensions/untitledgoose`, including BepInEx installers/runtime requirements, source-backed BepInEx runtime acquisition through the captured-install pipeline, Epic launcher metadata, source-backed BepInEx folder and `BepInEx.cfg` setup actions, and the historical purge migration metadata.
- `game-7daystodie/src/index.tsx` prompts for a User Data Folder, writes launcher settings, stores profile prefix offsets, and deploys modlets under that selected folder's `Mods` directory. DMM now gives target-root resolvers and launch-option requirements access to persisted extension setting values, exposes a ready typed path setting for the 7DTD User Data Folder in phone/tablet Extension Settings, routes 7DTD modlet deployment through an extension-declared target root that resolves either the selected UDF `Mods` folder or Vortex's game-root fallback, declares a generic setting-driven Steam launch argument requirement for `-UserDataFolder="..."`, and stores the Vortex-equivalent profile prefix offset as a profile-scoped extension setting consumed by the load-order deploy hook as `makePrefix(idx + offset)`. DMM does not copy Vortex's `launchersettings.json` because that file is Vortex launcher app state; the Deck equivalent is the existing Decky-owned Steam launch-options API.

Implementation priority from this audit:

1. Keep any future `registerGame`/`registerGameStub` placeholders temporary; promote them into dedicated source-backed DMM extension packages before claiming parity.
2. Continue using the generic generated load-order file helper for future games that write ordered text manifests.
3. Add more migration command runtimes for source-backed migrations that rewrite load-order state, adopt unmanaged files, or migrate profile/settings data. Manifest `set-mod-type` retagging/retargeting, active-profile redeploy, and old-extension-version gating are implemented; No Man's Sky now uses those primitives for its Vortex 1.0.1 purge-retag-redeploy migration.
4. Live-validate the UMM flow on Deck for one UMM-dependent Unity game: auto-acquire or manually acquire UMM, install it as a managed tool provider, launch it through the Decky-owned extension-tool action, and confirm DMM reports the runtime requirement correctly. Do not invent a noninteractive patch runtime unless a source-verified UMM command contract is found.
5. Extend the reusable XML/MTL merge helper as new source-backed merge shapes appear, and add source-backed patch-existing/setup runtime for games that modify existing user/game files outside the current deploy mapping model.
6. Extend new-file monitoring beyond extension-selected `added-files` adoption with concrete removed-file recovery handlers and a generic user-facing unmanaged adoption wizard.

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
- Setup/prep now has a typed runtime for the safe Vortex subset represented as extension-declared require/ensure path/file actions, safe marker rename actions, and regex text-file patches. BG3 source-backed setup now creates the Public profile `modsettings.lsx` from Vortex's v8 default template without overwriting an existing user file. Remaining setup gaps are richer source-specific action primitives for generated config templates from current user state and historical setup migrations.
- `requiresLauncher` is represented as source-backed launcher requirement metadata. Matching Steam requirements are evaluated as satisfied diagnostics for Steam Deck installs; Epic/Xbox/GOG launcher application remains metadata-only until DMM has a verified non-Steam launcher boundary.
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
- Torchlight II is promoted from catalog metadata into `internal/extensions/torchlight2`, covering Vortex's `.mod` installer, documents mods target root, Linux `ModLauncher.bin.x86` executable metadata, and source-backed Steam launcher requirement diagnostics.
- Galactic Civilizations III is promoted from catalog metadata into `internal/extensions/galacticcivilizations3`, covering Vortex's documents-root mod path selection, broad archive copy installer, `.faction` routing, Crusade mod-type metadata, and in-game enable reminder.
- A Hat in Time is promoted from catalog metadata into `internal/extensions/ahatintime`, covering Vortex `modinfo.ini` archive root handling and supported Modding Tools metadata.
- GreedFall is promoted from catalog metadata into `internal/extensions/greedfall`, covering Vortex datalocal wrapper stripping, FOMOD exclusion, executable version metadata, and a did-deploy hook that refreshes managed target timestamps after deployment.
- Surviving Mars is promoted from catalog metadata into `internal/extensions/survivingmars`, covering Vortex `modcontent.hpk` archive root handling and the AppData mods target root adapted to Steam Proton compatdata on Deck.
- Daggerfall Unity is promoted from catalog metadata into `internal/extensions/daggerfallunity`, covering Vortex `.dfmod` handling, Windows/no-platform payload filtering, and `DaggerfallUnity_Data/StreamingAssets` deployment.
- Sekiro is promoted from catalog metadata into `internal/extensions/sekiro`, covering Vortex loose `.partsbnd.dcx` handling, root `parts` package handling, and Mod Engine presence diagnostics.
- Dawn of Man is promoted from catalog metadata into `internal/extensions/dawnofman`, covering Vortex scenario `.scn.xml` installs into Documents scenarios, UMM-style `Info.json` installs into game `Mods`, setup metadata for the game `Mods` folder and Documents scenario folder, shared UMM tool archive staging/acquisition metadata, managed UMM tool discovery/default-primary launch metadata, and a runtime requirement for UMM-backed mods.
- Team Fortress 2 is promoted from catalog metadata into `internal/extensions/teamfortress2`, covering Vortex `.vpk` archive handling into `tf/custom`, Hammer supported-tool metadata, and `tf/steam.inf` ClientVersion discovery.
- Bloodstained: Ritual of the Night is promoted from catalog metadata into `internal/extensions/bloodstainedritualofthenight`, covering Vortex `.pak` archive handling, the `BloodstainedRotN/Content/Paks/~mods` target root, and the alphabetic load-order prefix rewrite Vortex uses because the game loads paks by folder name. Vortex unmanaged-file import and migration surfaces remain blocked metadata.
- Code Vein is promoted from catalog metadata into `internal/extensions/codevein`, covering Vortex `.pak` archive handling, the `CodeVein/content/paks/~mods` target root, executable version metadata, and the alphabetic load-order prefix rewrite Vortex uses because the game loads paks by folder name. Vortex unmanaged-file import and migration surfaces remain blocked metadata.
- Mount & Blade is promoted from catalog metadata into `internal/extensions/mountandblade`, covering the three Vortex-registered variants, `module.ini` module-package installs, supported loose override file routing into each game's native module, and native module version discovery.
- Star Wars: Knights of the Old Republic and KOTOR II are promoted from catalog metadata into `internal/extensions/kotor`, covering Vortex game-root folder installs, default override-folder installs, KOTOR II Steam launcher metadata, and explicit blocked TSLPatcher utility/mod installers.
- Neverwinter Nights is promoted from catalog metadata into `internal/extensions/neverwinter`, covering classic NWN and NWN: Enhanced Edition loose extension routing, override-folder preservation, structured Neverwinter folder archives, and the EE Documents mod root. Neverwinter Nights 2 is covered in the same package for Vortex `.mod` module archives and Documents module/override target roots.
- Factorio, No Man's Sky, The Witcher, and The Witcher 2 are promoted from catalog metadata into `internal/extensions/factorio`, `internal/extensions/nomanssky`, and `internal/extensions/witcherlegacy`. This covers Vortex-verified external/default mod roots, user-content mod-type routing, No Man's Sky `.pak` and `.dll` mod-type routing, Steam executable metadata, No Man's Sky `DISABLEMODS.TXT` marker rename setup, setup metadata, and No Man's Sky's Vortex 1.0.1 purge-retag-redeploy migration through generic extension-declared migration commands.
- Total War: Three Kingdoms is promoted from catalog metadata into `internal/extensions/totalwarthreekingdoms`, covering Vortex `.pack` folder-copy handling, `data` deployment, Assembly Kit supported tools, launcher metadata, setup metadata, and a deploy reminder.
- War Thunder and World of Tanks are promoted from catalog metadata into `internal/extensions/warthunder` and `internal/extensions/worldoftanks`. This covers War Thunder skin/audio mod-type routing, the Vortex `config.blk` audio toggle through the reusable text patch deploy hook, World of Tanks `version.xml`-derived `res_mods/<version>` targeting, and default archive-root deployment.
- Darkest Dungeon is promoted from catalog metadata into `internal/extensions/darkestdungeon`, covering Vortex `project.xml` installers, generated `project.xml` installers for no-project archives, game-directory structure matching through the generic custom-planner `GamePath` input, hero portrait routing, setup metadata, and store launcher metadata.
- Dragon's Dogma is promoted from catalog metadata into `internal/extensions/dragonsdogma`, covering Vortex nativePC archive routing, the Vortex invalid-archive confirmation installer, and selective MT Framework ARC merging for `game_main.arc` and `title.arc` through an extension-owned restore-aware `will-deploy` hook backed by the shared ARCtool bridge. Historical staged-mod migration remains blocked metadata until DMM has the shared migration runtime.
- Blade & Sorcery is promoted from catalog metadata into `internal/extensions/bladeandsorcery`, covering Vortex official `manifest.json` installer routing, engine-injection routing through `dinput`, obsolete MulleDK19 `mod.json` blocking, supported VR tools, setup metadata, generated `StreamingAssets/Mods/loadorder.json` through the reusable JSON load-order hook, and a typed open-directory action for the official mods folder. Source-backed drag/drop load-order UI, migration, and version-validation behavior remains blocked metadata.
- Monster Hunter: World is promoted from catalog metadata into `internal/extensions/monsterhunterworld`, covering Vortex `nativePC` archive stripping, ReShade `.ini` deployment to the game root with the same ReShade warning, Stracker's Loader root-file routing, setup metadata for `nativePC`, and HunterPie/SmartHunter/MHW Transmog supported-tool metadata.
- Fallout: New Vegas, Fallout 4 VR, and Skyrim VR are promoted from catalog metadata into `internal/extensions/falloutnv`, `internal/extensions/fallout4vr`, and `internal/extensions/skyrimvr`. This covers Vortex `Data` root installers, script-extender installer/launch-tool metadata for NVSE/F4SEVR/SKSEVR, Gamebryo plugin activation metadata, archive invalidation target metadata, Fallout NV 4GB patch `dinput` routing, VR ESL-enabler routing, conflict-ignore metadata where Vortex declares it, and source-backed supported tools. Fallout 4 VR and Skyrim VR now use a reusable plugin-activation metadata condition so `.esl` support is enabled only when the active profile has an enabled installed mod with Vortex `eslEnabler=true` metadata, matching the verified Vortex extension behavior. Remaining work is live validation on a real VR install with the ESL enabler and one light plugin.
- Source-backed helpers now exist for Vortex `dazip` outer/inner archive planning/AddIns generation, UMM game opt-in plus tool archive staging/acquisition/runtime-provider metadata, the shared Vortex `dinput` installer/mod type, and the Vortex GeDoSaTo all-texture installer/texture-root behavior. DInput archives now route files beside the extension-declared game executable and require a generic unsafe-DLL installer-choice confirmation before planning, matching Vortex's trust prompt. Dark Souls II opts into GeDoSaTo texture deployment with `textures/DarkSoulsII` target-root mapping. ENB remains metadata because the upstream Vortex installer registration is commented out.
- Witcher 3 now implements Vortex-style menu `.part.txt` settings merges as extension-owned restore-aware `patch-existing` mappings over the Proton Documents settings files. Witcher 3 config-matrix XML merges are also extension-owned: DMM strips the raw config XML mappings, reads the native or `.vortex_backup` base, merges `UserConfig.Group` entries by `id`, replaces matching `VisibleVars.Var` entries by `id`, appends missing vars/groups, and returns a generated restore-aware `patch-existing` mapping. Script Merger setup is now extension-owned too: DMM declares the tool acquisition from `IDCs/WitcherScriptMerger`, verifies the downloaded archive and extracted executable against Vortex's pinned MD5 values, stages matching archives as `tool-only`, records managed tool metadata, rewrites `WitcherScriptMerger.exe.config` after install, and exposes an acquire/reinstall action through the generic tool-acquisition path. Remaining Witcher gaps are Script Merger live execution validation, manual load-order semantics, and collection/profile merged-data import/export.
- Remaining gaps are mostly breadth: more Vortex helper shapes must be represented as reusable SDK helpers rather than copied per game.
- Needed helper APIs include source-backed versions of Vortex `testSupportedContent`, advanced `mergeMods` path transforms beyond the current XML/MTL helper, broader wrapper-root normalization variants, and richer component-choice rules.
- Needed shared mod-type helpers include any future source-backed ENB runtime path and any future source-backed Unity Mod Manager command contract if UMM exposes one beyond its own UI.

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
- Has native BA2/BSA list/read/extract support backed by Vortex's Gamebryo archive fixtures. Has native BSA create/write support for uncompressed v103/v104/v105 archives with round-trip reader tests. Has a first-party QuickBMS API namespace and typed process bridge that mirrors Vortex's register/list/extract/write/reimport flags, wildcard filter file, log file, timeout, registration gating, and list parsing contract. Has a typed ARCtool process bridge that mirrors Vortex's Dragon's Dogma defaults, list verbose-file parsing, extract temporary-rename behavior, and create sidecar order-file behavior; Dragon's Dogma now uses it to generate restore-aware merged ARC mappings before the normal DMM deploy planner runs. BA2 write remains pending.

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
- `gamemode-activated`, `will-install-dependencies`, `check-mods-version`, `removed-files`, and `update-conflicts-and-rules` have typed partial runtimes but not broad Vortex request/response parity.
- Missing a broad arbitrary request/response bus equivalent to Vortex `emitAndAwait`. `browse-for-download` and `nexus-download` now map to typed DMM acquisition primitives, while `deploy-single-mod`, `purge-mods-in-path`, `discover-tools`, `bake-settings`, purge state-migration execution, manifest `set-mod-type` migration execution, and migration active-profile redeploy have typed DMM runtime primitives. Broad arbitrary command-bus dispatch, extension-specific download dialogs outside declared acquisitions, and state/profile migration execution are still pending.

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
- Has source-backed metadata declarations for Vortex UI/state registration surfaces, including `registerAction`, `registerSettings`, `registerTest`, `registerHealthCheck`, `registerDialog`, `registerMainPage`, `registerDashlet`, `registerTableAttribute`, `registerLoadOrderPage`, `registerProfileFile`, `registerReducer`, and `registerPersistor`. `registerStartHook` is no longer tracked as a generic UI/state blocker because the only verified source-backed use is the dependency-manager startup conflict check represented by a typed runtime. `registerTest` now has a typed diagnostics runtime for extension-owned checks, `gamemode-activated` tests run through the Decky active-game report path, and Gamebryo `apply-settings` INI edits are represented as extension-declared profile-file patches. Other test triggers and arbitrary UI/state callbacks remain pending.
- Missing the generic runtime renderer/executor for those declarations. Converted extensions can advertise blocked UI/state capabilities now, and ready typed actions can be listed and executed from selected-game UI surfaces, but DMM cannot yet render arbitrary extension dialogs/pages/table attributes or execute arbitrary extension callbacks.

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
- Vortex `quickbms-support` APIs now map to a typed first-party DMM namespace backed by `internal/quickbms`, with register/list/extract/write/reimport semantics and explicit per-game opt-in. Vortex `gameversion-hash`'s `getHashVersion` behavior now has a reusable DMM helper: converted game extensions can declare `hashFiles`/`hashDirPath`, DMM computes the Vortex-style chained MD5 hash, and it maps through the Vortex backend hash map when reachable. Source review shows the Vortex hash-map entry editor actions are guarded by `DEBUG_MODE`, so they are desktop developer tooling rather than DMM runtime behavior.
- Vortex `modtype-umm`'s `ummAddGame` API now maps to a typed DMM `umm` helper for converted first-party Go extensions. DMM also supports the source-backed `umm-installer` archive shape as a managed `tool-only` staging payload and exposes Vortex's UMM 0.24.2 package as extension-declared acquisition on both the supported tool and the runtime requirement. The acquisition records Vortex's GitHub browse instructions while DMM resolves the source-verified GitHub release asset directly through the captured-install pipeline. Decky reports active games to the backend, and runtime requirements marked `auto_acquire` queue the same captured-install pipeline when missing. The update-check route also runs the same auto-acquire pass, matching Vortex `modtype-umm`'s `check-mods-version` handler, which calls `ensureUMM` rather than a separate remote update feed. Tool discovery reports staged DMM-managed tool payloads and extension-declared game-root tools through `/api/games/{appID}/tools`; managed UMM tools inherit Vortex-style default-primary metadata; ready executable tools, including installed managed tool archives, can be launched by Decky through queued extension-tool actions. Those tool actions can now opt into waiting for Steam app lifetime exit notifications before completion, which is required for generated-output tools such as FNIS. Source review shows the UMM patch/configuration step remains inside UMM's own UI.
- Source-backed metadata exists for shared-system APIs/events used by FNIS, local game settings, dependency management, new-file monitoring, and Vortex test helpers: `deploy-single-mod`, `purge-mods-in-path`, `browse-for-download`, `nexus-download`, `discover-tools`, `bake-settings`, `unfulfilled-rules`, `registerGameInfoProvider`, new-file adoption, and removed-file reporting. `browse-for-download`, `nexus-download`, `deploy-single-mod`, `purge-mods-in-path`, `discover-tools`, `bake-settings`, `unfulfilled-rules` dependency override handlers, Decky extension-tool launch actions, profile metadata dependency evaluation, removed-file reporting, purge state-migration execution, manifest `set-mod-type` retag/retarget migration execution, migration active-profile redeploy, wait-for-tool-exit actions, backend-prepared tool input files, and generated profile-mod recording/deploy are backed by implemented typed runtime primitives. FNIS now has a reusable source-backed helper for supported-game opt-in, settings, patch-list parsing, installed/version diagnostics, automatic generator launch, profile patch input writing, and generated profile output deployment. The others remain metadata-only or blocked unless separately noted.
- Missing runtime implementations for extension-owned persistent state/persistor and broader migration behavior equivalent to Vortex `registerReducer`, `registerPersistor`, and state-rewriting `registerMigration` handlers.

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

- Has load-order summaries, profile-scoped file winner overrides, profile-scoped Gamebryo plugin row enable/order/index-lock state, Vortex-format Gamebryo `plugins.txt`/`loadorder.txt` generation, extension-declared LOOT game/masterlist IDs, source-backed incompatible BSA/BA2 archive-version diagnostics for enabled managed Gamebryo plugins, source-backed Oblivion INI missing-font diagnostics plus automatic repair, source-backed Skyrim-family font diagnostics from `Skyrim - Interface.bsa`, DMM-owned LOOT masterlist/prelude cache paths, profile-local LOOT `userlist.yaml` paths under `loot/{game}/profiles/{profileID}/userlist.yaml`, a refresh endpoint for Vortex's `v0.29` masterlist/prelude URLs, DMM-owned LOOT userlist read/write endpoints, profile-copy seeding for LOOT user rules, a `dmm-loot-sorter.v1` JSON helper contract and Rust helper source for real libloot sorting, a sort endpoint that persists profile-scoped mutable plugin order and applies the profile, Advanced UI for plugin rules/group assignments/group order rules, Unreal sortable PAK helper, Witcher `mods.settings` generation, Witcher menu `.part.txt` settings merge, Witcher config-matrix XML merge, BG3 Divine-backed `modsettings.lsx` generation from enabled pak `meta.lsx`, and several generated load-order file helpers.
- Missing live validation of the packaged `dmm-loot-sorter` helper against real Bethesda plugin sets, dependency conflict graph evaluation beyond duplicate file targets, cycle handling, full Vortex group editor parity, local/global LOOT rule toggle UX if we decide to expose it, bespoke load-order pages beyond the generic profile-order editor, BG3 load-order import/migration UI, and BA2 write/repack support. Source-backed Gamebryo index-lock state is now represented in profile plugin activation state and applied during generated load-order output, but the Vortex-style table/detail editor for those locks is still pending UI work.

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
- Has source-backed runtime for new-file adoption/monitoring, plus blocked metadata for Vortex/NMM/MO import UI entries and savegame profile features. Profile-local settings file sync itself is implemented through game extensions that declare the Vortex `local_game_settings` files, and `bake-settings` has a profile-switch lifecycle runtime. Concrete extension handlers for mod-provided settings baking remain to be added where source review requires them.
- Missing the actual runtime implementations for the remaining import/savegame/profile utility features.

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
