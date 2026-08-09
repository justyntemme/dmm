# Architecture Decisions

Use this file only when an implementation path requires a meaningful architectural choice that the user has not already explicitly made. Keep entries concise enough to review later, but complete enough to explain why the code went in that direction.

## When To Add An Entry

- Add an entry when choosing between competing architecture patterns, storage models, extension boundaries, event delivery mechanisms, deployment semantics, privilege boundaries, or UI/system integration mechanisms.
- Do not add entries for routine implementation details, small bug fixes, obvious refactors, or decisions the user made directly in conversation.
- If source verification changes the decision later, replace or amend the entry rather than preserving stale decision history.

## Entry Format

### Short Decision Title

- Date:
- Area:
- Decision:
- Options considered:
- Rationale:
- Tradeoffs:
- Verification/source references:
- Follow-up:

## Decisions

### Pass Game Path Through Extension Install Planning

- Date: 2026-08-08
- Area: Extension framework and Vortex installer parity
- Decision: Add resolved `GamePath` and reserved `LibraryPath` fields to `installplan.BuildOptions` and `installplan.BuildInput`, and pass `GamePath` through the game extension registry into custom installer builders.
- Options considered: let custom installers rediscover the game folder; add game-specific server branches; expose the resolved game path through the generic extension planner boundary.
- Rationale: verified Vortex installers can inspect the discovered game directory while classifying an archive. Darkest Dungeon maps no-project archives by comparing archive folders to the real game directory structure, so the game extension needs that state without moving Darkest Dungeon logic into core.
- Tradeoffs: custom builders now receive more environment context and must avoid mutating game files during planning. This remains safer than rediscovery because the server already owns game-path discovery and can pass the exact path used for staging/deployment.
- Verification/source references: Vortex `extensions/games/game-darkestdungeon/src/index.js` uses `prepareForModding`, `_DIRECTORY_STRUCT`, and `getModsFolder()` during installer matching/building; DMM implementation in `internal/installplan`, `internal/gameext`, and `internal/extensions/darkestdungeon`.
- Follow-up: pass `LibraryPath` once a source-verified installer requires Steam-library-relative state, and keep planner-stage game-path access read-only.

### Add Framework Extensions Beside Game Extensions

- Date: 2026-08-08
- Area: Extension framework and Vortex parity
- Decision: DMM supports first-party framework extensions with `Kind: "framework"` in addition to game extensions. Framework extensions may register shared capabilities such as interpreters, archive types, game stores, extension APIs, UI actions/settings/tests, profile features, state stores, migrations, health checks, and attribute extractors without declaring a Steam app ID.
- Options considered: keep all shared behavior as unregistered Go helper packages; force shared Vortex counterparts to fake a Steam app ID; add a first-class framework extension kind.
- Rationale: Vortex has many shared extensions (`common-interpreters`, archive support, Gamebryo plugin management, QuickBMS, dependency management, game stores) that are not games but must still be audited and converted. A framework kind keeps those capabilities visible in `/api/extensions` and persisted snapshots without polluting the game list or moving shared behavior into generic core branches.
- Tradeoffs: the first pass registers capability metadata only; executable engines such as BA2/BSA/QuickBMS and interpreter launch paths still need concrete runtime implementations before any extension can rely on them. This prevents no-op parity claims while giving conversion work a stable Go SDK shape.
- Verification/source references: Vortex `common-interpreters/src/index.ts` registers `.jar`, `.vbs`, `.py`, `.cmd`, and `.bat`; DMM implementation in `internal/extensions/sdk`, `internal/gameext`, and `internal/extensions/commoninterpreters`.
- Follow-up: convert archive/game-store/framework Vortex extensions into framework packages only after each runtime engine or integration path is source-verified and explicitly represented in the SDK.

### Model Extension-Owned Dynamic Launch Arguments

- Date: 2026-08-08
- Area: Extension framework and Steam launch-tool integration
- Decision: Add typed dynamic launch arguments to the extension SDK. Extensions can declare that a launch tool needs arguments derived from enabled profile mods, starting with an `enabled-mod-root` argument source. The backend resolves those arguments from staged manifests and exposes the desired Steam launch options through the existing Decky launch-tool action path.
- Options considered: add a Quake-specific launch-option hook; generate per-game wrapper scripts from extension hooks; add generic typed dynamic launch arguments resolved from enabled profile state.
- Rationale: Quake 4 `fs_game` support needs the enabled mod folder in `+set fs_game <folder>`, but that behavior belongs in the Quake extension. A typed dynamic argument keeps the game-specific token in the extension while core only validates placeholders, reads DMM-owned staged manifests, and builds the same launch-option request shape used by static tools such as SMAPI.
- Tradeoffs: the MVP implementation supports game-root target mappings only and blocks external target-root dynamic launch roots with a clear diagnostic. It also blocks `RequireExactlyOne` launch tools until exactly one matching enabled mod can provide a launch root, which is correct for Quake 4 but may need additional generic selection semantics for future games that support multiple active roots.
- Verification/source references: Quake 4 research notes in `docs/extensions/games/quake-4.md`; implementation in `internal/extensions/quake4`, `internal/extensions/sdk`, `internal/gameext`, and `internal/server`; tests in `internal/extensions/quake4/spec_test.go` and `internal/server/server_test.go`.
- Follow-up: live-test a Quake 4 `fs_game` archive and Steam launch-option update on the Deck, then extend the same typed argument model only when a source-verified extension requires more input kinds.

### Use A Process-Scoped Pairing Token For MVP API Auth

- Date: 2026-08-08
- Area: Network security and Decky/backend boundary
- Decision: Decky generates a process-scoped API token when it starts the backend, writes it to a `0600` runtime file under Decky Mod Manager state, passes it to the Go backend as `DMM_AUTH_TOKEN`, and gives the phone/tablet app a paired URL with the token in the URL fragment. The web app stores that token locally, removes it from the visible URL, and sends it as `X-DMM-Token` for fetch calls plus a WebSocket query token for `/api/events/ws`. `/api/health` remains unauthenticated for liveness checks.
- Options considered: no auth with LAN-only filtering; persisted password/API key in `config.json`; cookie-based login; process-scoped bearer token generated by Decky.
- Rationale: LAN-only is not enough on shared networks. A persisted password adds setup friction and secret-rotation work before MVP. Cookie login requires a pairing ceremony and CSRF model we do not need yet. A process-scoped token gives immediate protection for remote API calls, works with the Decky plugin, WebSocket, and OS `nxm://` handler, and rotates naturally when the backend is restarted.
- Tradeoffs: the paired URL is effectively a bearer secret, so a leaked URL can control the API until the backend/token is rotated. A reset/session-rotation UI is still needed before MVP approval. Static web assets remain public, but state APIs are protected.
- Verification/source references: implemented in `internal/server/auth.go`, Decky bridge token generation in `decky/main.py`, `nxm://` handler token-file read in `cmd/dmm-nxm-handler/main.go`, and Svelte token pairing in `web/src/App.svelte`.
- Follow-up: add a visible Decky reset token action, polish security copy, and validate the full paired-phone + Nexus capture flow on the Steam Deck.

### Require Explicit Metadata For Mutually Exclusive Installer Choices

- Date: 2026-08-07
- Area: Extension framework and installer-choice validation
- Decision: DMM enforces installer-choice conflicts and cardinality only from explicit FOMOD metadata or extension-declared `ComponentChoiceSpec.GroupType` rules such as `SelectExactlyOne`, `SelectAtMostOne`, `SelectAtLeastOne`, and `SelectAll`.
- Options considered: infer conflicts from package names/folder names; treat every multi-component archive as all-or-nothing; require FOMOD/extension metadata to declare choice semantics.
- Rationale: filename inference is unsafe across game ecosystems and would silently make wrong install decisions. Vortex-style FOMOD installers already encode group selection rules, and DMM's extension framework can encode equivalent rules for non-FOMOD component archives without moving game-specific logic into core.
- Tradeoffs: archives that do not declare conflict metadata may still expose combinable component choices until the game extension adds verified rules. This is safer than guessing, but extension coverage must improve as representative mods reveal real mutually exclusive groups.
- Verification/source references: DMM install-plan tests now cover an extension-declared `SelectExactlyOne` component group and reject selecting two mutually exclusive variant roots while accepting exactly one.
- Follow-up: when researching game extensions, encode verified mutually exclusive component groups in the extension spec and keep core validation generic.

### Surface Extension Lifecycle Messages As Action Center Jobs

- Date: 2026-08-07
- Area: Extension framework and user-visible lifecycle actions
- Decision: Promote `did-deploy`/`did-purge` extension hook messages into deduped `extension-notice` jobs instead of leaving them as logs or creating a separate notification table.
- Options considered: keep hook messages as backend logs only; add a new extension-notification persistence model; reuse the existing job manager with a new `extension-notice` type.
- Rationale: Vortex extensions can emit post-deploy notifications with required manual work, as verified in the Ghost Recon Breakpoint extension's AnvilToolkit repack warning. DMM already has a durable job model, WebSocket events, phone/tablet Action Center, Decky job toasts, and cancel/dismiss semantics. Reusing that pipeline makes extension follow-up work visible without adding game-specific server code or another action persistence path.
- Tradeoffs: the first version is message-only. Launching an extension-declared tool such as AnvilToolkit from the notice still needs a generic extension tool-runtime action contract before DMM can safely execute it across Steam Deck/Linux/Proton contexts.
- Verification/source references: Ghost Recon Breakpoint Vortex extension package `site/mods/972?file_id=7463`, `index.js` `deployNotify`, `runAtk`, and `context.api.onAsync('did-deploy', ...)`.
- Follow-up: add a generic extension tool-runtime action shape so notices can expose safe action buttons such as `Run AnvilToolkit` without hardcoding Ghost Recon behavior in the backend.

### Store Extension Text Choices In The Existing Selection Map

- Date: 2026-08-07
- Area: Extension framework and installer-choice UI
- Decision: Add extension-owned text prompt metadata to the existing installer-choice step/group schema, and continue storing submitted values in `choices_json` as `map[groupID][]string` with the text value at index 0.
- Options considered: create a separate text-input table/schema; encode text as a fake plugin option; reuse the existing grouped selection map with explicit `Type: "Text"` metadata.
- Rationale: Vortex extensions can ask for free-text installer input, as verified in the Ghost Recon Breakpoint `.forge` folder rename dialog. The prompt belongs in the game extension, but the persistence, Action Center, phone UI, Decky modal, presets, and apply/retry semantics should stay on the one installer-choice pipeline. A typed text group avoids fake options while keeping storage and saved presets simple.
- Tradeoffs: text values do not have a separate typed storage column, so extension builders must validate and sanitize the value they consume. This keeps the core generic and prevents game-specific validation from moving into server code.
- Verification/source references: Ghost Recon Breakpoint Vortex extension package `site/mods/972?file_id=7463`, `index.js` `folderRenameDialog`, `installDataFolder`, and `installLoose`.
- Follow-up: live-test a representative Breakpoint `.data` archive and reuse this same text-group shape for future source-verified Vortex prompts before adding any new prompt primitive.

### Add Deploy Ignore As An Extension Capability

- Date: 2026-08-07
- Area: Extension framework and deployment planning
- Decision: Add `RegisterDeployIgnore` to the first-party extension SDK and pass those patterns into the generic deployment planner as `IgnoreDeployPatterns`.
- Options considered: encode Metro readme/changelog skips inside the Metro installer; treat deploy ignores as conflict ignores only; add a generic extension-owned deploy-ignore capability.
- Rationale: verified Vortex extensions can declare both `ignoreConflicts` and `ignoreDeploy`. They are different behaviors: conflict ignores suppress conflicts when a target already exists, while deploy ignores prevent matching staged files from being deployed at all. Keeping both extension-owned preserves Vortex parity without game-specific core code.
- Tradeoffs: ignored files still exist in DMM-owned staging and install metadata, but they are excluded from the active deployment plan. This is intentional for provenance and profile repair, but the UI may later need a power-user detail view that explains why a staged file is not deployed.
- Verification/source references: Metro Exodus Vortex package file `site/mods/907?file_id=8800` declares `IGNORE_CONFLICTS` and `IGNORE_DEPLOY` for `**/changelog*` and `**/readme*`.
- Follow-up: reuse the same capability for other source-verified Vortex extensions that declare `ignoreDeploy`, and add UI diagnostics if users need to inspect ignored staged files.

### Split Extension Coverage From Manage-Ready Filtering

- Date: 2026-08-07
- Area: Extension capability model and game selection UX
- Decision: Treat `supported` as "DMM has a registered extension for this Steam app" and add a separate backend-derived `coverage` classification for `installer`, `research_blocked`, `browse_only`, `workshop_only`, and `metadata_only`. The default game selector shows manage-ready games (`installer` and `workshop_only`), with explicit filters for all DMM extensions and all installed games.
- Options considered: keep one boolean `supported`; add hardcoded frontend heuristics; derive coverage in the backend from extension capabilities.
- Rationale: research-blocked Nexus manifests are useful because they expose verified domains and diagnostics, but they must not look deployable. Backend classification keeps web and Decky behavior consistent and makes extension coverage measurable in live checks.
- Tradeoffs: the default selector hides research-blocked games until the user changes the filter, so extension authors need the coverage live check or DMM Extensions view to see pending research targets.
- Verification/source references: Vortex source search for the remaining unsupported games found no official game extension matches; authenticated Nexus `games.json` returned no exact domains for the current unsupported list. Live `/api/games` coverage check passed on the Deck after deployment.
- Follow-up: add real installer coverage only after source-verified extension behavior or representative archive review, and use `REQUIRE_NO_UNSUPPORTED=1` only for release-candidate coverage gates.

### Keep ModDB Deferred Without Scraping

- Date: 2026-08-07
- Area: Remote provider architecture
- Decision: Keep ModDB as a deferred provider until a supported official API/client path is verified. Do not add HTML scraping or mirror-page parsing as an MVP provider.
- Options considered: scrape ModDB pages/mirror redirects; use third-party scraping libraries; route users to direct archive URLs; keep ModDB deferred and rely on mod.io/direct URLs where appropriate.
- Rationale: current source review found ModDB forum guidance stating there is no ModDB API and third-party libraries describe themselves as scrapers. DMM's provider policy requires verified official APIs, source, schemas, or supported client behavior for automated providers. Scraping would be brittle and likely hostile to the "move fast but don't build hacks into product paths" rule.
- Tradeoffs: users cannot browse/import ModDB pages directly in MVP. They can still use direct archive URLs where a stable archive URL is available, and mod.io remains the supported API-backed sibling provider.
- Verification/source references: ModDB forum thread "Is there a ModDB API?" says no API was available and no ETA; third-party `moddb` Python docs describe scraping; mod.io documentation exposes a proper REST API/download object.
- Follow-up: revisit ModDB only if ModDB publishes an official API or a verified first-party client contract that supports automated downloads without scraping.

### Route Bulk Imports Through Captured Installs

- Date: 2026-08-07
- Area: Collections, bulk install, and provider workflow
- Decision: MVP bulk install accepts multiple provider URLs and queues each one through the existing captured-install pipeline. Full Vortex/Nexus collection manifest replay remains a separate parity feature.
- Options considered: implement a separate bulk downloader/installer; implement full Vortex collection sessions first; route URL lists through the current captured-install code path.
- Rationale: Vortex collections include source metadata, rules, bundled files, installer choices, patches, and game-specific parsers. DMM's validated non-premium Nexus path depends on BrowserView-generated `nxm://` credentials and immediate capture/download. Bulk URL import can improve the MVP without bypassing that proven path or creating duplicate install semantics.
- Tradeoffs: this does not yet import Nexus collection pages, `.collection` archives, Vortex rule ordering, bundled files, or author-saved installer choices. Users can bulk-add known links now; collection-manifest parity still needs source-verified design work.
- Verification/source references: Vortex collection types in `/tmp/dmm-vortex/src/renderer/src/extensions/collections/types/ICollection.ts`; collection installer/session code in `/tmp/dmm-vortex/src/renderer/src/extensions/collections/makeInstall.ts` and `/tmp/dmm-vortex/src/renderer/src/extensions/collections/util/InstallDriver.ts`.
- Follow-up: add a real collection manifest reader only after verifying Nexus collection API/page behavior and mapping collection members onto DMM profile/install-candidate semantics without bypassing BrowserView credential capture.

### Pass Archive Filename Through Extension Install Planning

- Date: 2026-08-07
- Area: Extension framework and Vortex installer parity
- Decision: Add optional `ArchiveName` to `installplan.BuildOptions` and `installplan.BuildInput`, and pass it from server install flows that already know the archive path.
- Options considered: let extensions infer names from temporary extract folders; add game-specific server branches; pass source archive identity through the generic planner boundary.
- Rationale: verified Vortex installers can derive target folder names from the downloaded archive filename, such as FF7 Rebirth UE4SS script/DLL installers when `Scripts/` or `dlls/` sits at archive root. The generic planner boundary should expose that source identity so game extensions can mirror Vortex without core game-specific logic.
- Tradeoffs: existing callers may still pass an empty archive name, so extensions must keep a stable fallback. This is an extension API expansion, not a compatibility fallback path.
- Verification/source references: cached FF7 Rebirth Vortex extension package v0.4.0 uses `fileName` to derive the UE4SS mod folder in `installScripts` and `installDll`.
- Follow-up: use the same field for other verified Vortex installers that name output folders from archive filenames, and consider exposing archive metadata beyond filename only when a verified extension requires it.

### Use Planner-Owned Generic Installer Choices

- Date: 2026-08-07
- Area: Extension framework and installer UI
- Decision: Custom game-extension installers can return a typed `installplan.ChoiceRequiredError` with a FOMOD-compatible wizard JSON shape. The server records it as the same install-candidate/action-center flow used by FOMOD, and applying the candidate re-runs the extension planner with the saved selections.
- Options considered: keep non-FOMOD multi-choice archives blocked; add one-off server/UI paths per game; generalize the planner contract so extension code owns the prompt reason and option list while core owns persistence, UI delivery, staging, presets, and profile install.
- Rationale: verified Vortex extensions can prompt for non-FOMOD decisions, such as Star Wars Jedi: Survivor's multi-PAK selection. That choice is game/installer behavior and belongs in the extension, but DMM should not build a separate UI/action pipeline for every prompt type.
- Tradeoffs: the first implementation uses the existing FOMOD-compatible `steps/groups/plugins` JSON vocabulary, so the UI can render it immediately. Longer term, we may want a neutral UI schema name to avoid leaking FOMOD terminology into generic installer choices.
- Verification/source references: Vortex Jedi Survivor extension source at `https://github.com/Pickysaurus/vortex-jedi-survivor` prompts when multiple `.pak` files are present; DMM tests cover generic choice recording and selected PAK staging through the same install-candidate endpoints.
- Follow-up: live-test a real multi-choice archive in Decky and phone/tablet UI, then reuse the same planner-owned choice contract for other Vortex prompts before adding game-specific workaround code.

### Preserve Known Workshop Metadata Across Partial Steam Reads

- Date: 2026-07-31
- Area: Steam Workshop state mirroring
- Decision: Treat SteamClient Workshop sync as an observation stream with field-level quality. DMM replaces observed membership/order from the current Steam read, but preserves previously known item titles and local disabled state when the current read omits title or explicitly reports disabled state as unknown.
- Options considered: strictly replace every field on each sync; skip all syncs when `GetSubscribedWorkshopItems` returns zero; merge known user-visible metadata while still honoring current membership/order observations.
- Rationale: live Deck startup showed `GetDownloadedWorkshopItems` returning 15 Kenshi IDs while `GetSubscribedWorkshopItems` returned 0, even though a previous boot returned subscribed state. Strict replacement degraded the UI from named/manageable rows to unknown placeholder rows. Preserving only omitted fields keeps the backend durable without inventing subscription membership.
- Tradeoffs: if Steam stops exposing disabled state after a user changes it outside DMM, DMM may show the last known disabled value until a richer subscribed-state sync arrives. This is preferable to disabling all controls and losing names from a transient partial read, but it remains a live-validation gap for Workshop mutation flows.
- Verification/source references: Steam UI source calls `SteamClient.Apps.GetSubscribedWorkshopItemDetails(appID, publishedFileIDs)` after `GetSubscribedWorkshopItems`; live Deck logs on 2026-07-31 returned 15 titled detail rows for Kenshi through that method.
- Follow-up: live-test enable/disable/unsubscribe/order on a safe Workshop-heavy game and verify whether SteamClient reliably returns subscribed disabled state after mutation.

### Keep Steam Workshop Ordering Extension-Gated

- Date: 2026-07-31
- Area: Steam Workshop management and profile mod UI
- Decision: DMM shows installed/subscribed Steam Workshop items inside the same mod-management area as DMM-managed mods, with source tags and Steam-owned actions, but it does not merge Workshop items into the same profile priority sequence unless a game extension explicitly declares and implements a verified shared load-order model.
- Options considered: merge Workshop rows directly into profile mod order for every game; keep Workshop entirely separate in diagnostics; show Workshop in Mods but keep action/order semantics source-specific until verified per game.
- Rationale: Steam stores Workshop content under Steam-owned paths, and individual games decide whether Workshop items load from Steam's order APIs, native config files, separate folders, or game-specific manifests. A global merge would look convenient but risks implying behavior DMM cannot guarantee.
- Tradeoffs: users see source-specific order controls for Workshop content instead of one universal drag list across all sources. The benefit is that DMM can still disable, unsubscribe, and order Workshop items through verified Steam APIs without corrupting profile semantics for games that do not share one load-order surface.
- Verification/source references: live Decky introspection exposes `SteamClient.Apps.SetWorkshopItemsDisabledLocally`, `SubscribeWorkshopItem`, and `SetWorkshopItemsLoadOrder`; DMM extensions already declare Workshop coexistence/action support per game.
- Follow-up: when building each Workshop-heavy game extension, verify whether that game supports a true shared DMM/Workshop load-order model and encode it in the extension rather than in generic core.

### Persist Update State Through Source-Specific Providers

- Date: 2026-07-31
- Area: Provider architecture and installed-mod updates
- Decision: Installed-mod update checks route through source-specific provider capabilities keyed by normalized catalog/source ID. Nexus uses its dedicated Nexus update path, while GitHub Releases, Modrinth, Thunderstore, GameBanana, mod.io, and CurseForge implement the shared `catalog.UpdateModCatalog` capability. Direct, Local, and Steam Workshop persist explicit unsupported update state with source-specific messaging instead of being skipped or treated as Nexus edge cases.
- Options considered: keep the Nexus-only handler and special-case every other catalog in UI; add update methods to the existing URL resolver interface; add a capability interface on catalog implementations plus a server adapter that can wrap any update-capable catalog.
- Rationale: URL resolution and installed-mod update checks are related but not identical capabilities. A catalog-level optional capability keeps provider behavior with the verified provider adapter, avoids server-side provider switches, and gives the UI durable source-aware update status for both supported and unsupported providers.
- Tradeoffs: GitHub release file IDs are now encoded tag/asset identifiers instead of the previous hash-derived ID. That is an intentional pre-MVP breaking source-ID change so future update checks can resolve a specific release asset without persisting raw URLs. Steam Workshop remains Steam-owned platform content and should not be treated as a normal remote archive update path.
- Verification/source references: backend tests cover Nexus update caching, Nexus update install queueing, browser-required Nexus update failures, unsupported catalog status persistence, unsupported update-install rejection, Modrinth update resolution/queueing through a fake API, and GitHub latest-release asset update persistence through the server update endpoint. Live Deck Stardew update check on 2026-07-31 returned four current Nexus mods through the catalog update path.
- Follow-up: live-test GitHub Releases, Modrinth, Thunderstore, GameBanana, mod.io, and CurseForge update checks with safe real fixtures.

### Adopt Matching Generated Files Without Taking Restore Ownership

- Date: 2026-07-30
- Area: Deployment semantics and extension-generated files
- Decision: Add a generic `adopt-existing` deployment target policy for extension-generated profile files. The planner records an existing unmanaged target as DMM-managed only when the generated source and target checksums already match; if they differ, deployment conflicts instead of overwriting.
- Options considered: use restore-aware patch mappings; let extensions edit generated config files directly; copy generated files as ordinary mappings and accept unmanaged conflicts; add a dedicated adopt-only policy.
- Rationale: Stardew SMAPI mods generate `config.json` files after the game runs. Vortex preserves those through its Stardew config-mod feature, but DMM needs the same safety through the deployment manifest so profile disable/re-enable can keep user config without leaving staging details in the default UX. Restore-aware mappings would restore the config into the game folder when disabling a mod, which is not the profile-switching behavior we want. Direct extension writes bypass preview, rollback, purge, and manifest ownership. Adopt-only lets the extension copy the live file into a profile-owned generated staging area, then lets core own future deploy/remove behavior.
- Tradeoffs: this adopts only byte-identical targets and intentionally conflicts if the target changes between extension scan and deploy planning. Stardew still does not have Vortex's synthetic config mod UI model; DMM is using a profile-owned generated staging area instead. Future generated-file hooks must keep durable restore/profile state outside ephemeral event work directories.
- Verification/source references: Vortex Stardew config-mod source at `/tmp/dmm-vortex/extensions/games/game-stardewvalley/src/configMod`; DMM deploy planner tests now cover matching adoption and differing-target refusal; Stardew extension tests cover live config adoption, saved config redeploy, disable-time refresh, and archive-owned config exclusion.
- Follow-up: live-test Stardew by applying once after generated configs exist, disabling/re-enabling a SMAPI mod, and confirming the config is removed from the game folder while remaining available for re-enable.

### Store Profile Deployment Strategy Overrides With Profiles

- Date: 2026-07-30
- Area: Deployment settings and profile ownership
- Decision: Add profile-scoped deployment strategy overrides to SQLite-owned profile state, and resolve effective strategy as profile override, then game override, then extension default.
- Options considered: keep only the existing game-level config override; store profile overrides in JSON config keyed by profile ID; add a profile-owned database field for deployment strategy.
- Rationale: deployment strategy can affect how a profile is applied, previewed, rolled back, and repaired. Profiles, profile mods, and file conflict winners are already SQLite-owned; putting profile strategy in the same model avoids a cross-store source of truth and keeps future profile export/import coherent.
- Tradeoffs: this adds a storage migration and API surface. The game-level override remains useful as a default, but the UI must make the scope explicit so power users understand whether they are changing the selected profile or the game default.
- Verification/source references: DMM profiles and file winners already live in SQLite; deploy planning currently resolves one strategy before building mappings.
- Follow-up: expose profile override in Advanced Deployment Tools and ensure deployment preview/apply use the selected/default profile's override.

### Store File Conflict Winners On Profiles

- Date: 2026-07-30
- Area: Deployment conflict resolution
- Decision: Persist explicit duplicate-target file winners as profile-owned records keyed by absolute target path and winner installed-mod ID. The deploy planner consumes those records as overrides after the core validates the requested winner against the active managed duplicate-target preview.
- Options considered: encode conflict winners as mod priority only; store winners globally per game; store winners per profile and target path; defer persistence and keep preview-only controls.
- Rationale: conflict winners are loadout/profile behavior, not global mod metadata. Profile-scoped storage keeps the simple enable/disable flow intact while giving power users deterministic file winners when two enabled mods write the same target. Validating against the active preview prevents stale or cross-game records from influencing filesystem writes.
- Tradeoffs: target-path keys are absolute, so moving a game folder or changing a target root may leave stale overrides that are ignored until reset. This is acceptable for MVP and safer than fuzzy matching file paths across roots.
- Verification/source references: DMM deployment plans already resolve duplicate targets before apply, and profile mods already own enabled/priority state in SQLite. The implemented endpoint validates candidate installed-mod IDs from the current deployment plan before storing an override.
- Follow-up: add live UI validation with real conflicting mods, add Decky conflict summaries if needed, and revisit extension-owned load-rule semantics for games where file winners should come from plugin/load-order systems rather than manual file picks.

### Keep Deployment Strategy Overrides Game-Scoped Until Profile Storage Is Designed

- Date: 2026-07-30
- Area: Deployment settings and profile ownership
- Decision: Do not add profile-scoped deployment strategy overrides as another config map right now. Keep the existing game-scoped override as the only behavior source until profile-scoped settings are designed against the profile/database model.
- Options considered: add `profile_id -> strategy` to config; add a database table owned by profiles; defer profile-scoped overrides and keep extension/game strategy resolution unchanged.
- Rationale: deployment planning currently resolves one strategy per game from explicit game override, extension default, then symlink fallback. Profiles live in SQLite and can be rebuilt/imported; storing profile IDs in config would create a stale cross-store source of truth. A database-backed profile setting is cleaner but needs UI/API migration semantics and should be handled as its own feature.
- Tradeoffs: power users cannot choose different deployment strategies per profile yet. The payoff is avoiding a half-parallel override model that would complicate deployment previews, rollback/history interpretation, and future profile import/export.
- Verification/source references: current code stores game overrides in `config.Deploy.GameStrategies`; profiles and profile mods are database-owned; `buildGameDeployPlan` consumes `defaultDeploymentStrategy(appID)` once for the selected game.
- Follow-up: when implementing profile-scoped strategy, prefer a profile-owned database setting and update deploy settings responses to show game default, profile override, effective strategy, and reset behavior explicitly.

### Resolve Native/Proton Install Shape Through Extension Platform Metadata

- Date: 2026-07-30
- Area: Extension framework and runtime launch tools
- Decision: Game extensions can declare install platforms with relative marker files, and installers/launch-tool variants can opt into a platform ID. The core detects the active platform from extension markers, filters platform-specific installers, and resolves platform-specific launch executables/required files before publishing Decky Steam API actions.
- Options considered: hardcode Stardew Linux/Windows filename checks in the server; let duplicate SMAPI installers race and use whichever payload builds first; add extension-owned platform metadata and generic core validation/resolution.
- Rationale: Vortex's Stardew SMAPI installer has explicit platform variants, and SMAPI archives can contain both Linux and Windows payloads. The server must not know Stardew filenames, but it does need a generic way to choose the correct installer and launch target from the installed game shape.
- Tradeoffs: platform detection is marker-file based and depends on each extension declaring good markers. Unknown platform falls back to generic installer behavior, so live validation is still required for Proton installs. The payoff is a reusable contract for future native/Proton or edition-specific game handlers without adding core game-specific branches.
- Verification/source references: Vortex Stardew SMAPI platform files at `/tmp/dmm-vortex/extensions/games/game-stardewvalley/src/installers/smapi/linux.ts` and `windows.ts`; Vortex installer tests verify Windows extracts `internal/windows/install.dat` and Linux extracts `internal/linux/install.dat`.
- Follow-up: live-test with a Windows/Proton Stardew install and then extend the platform contract only if another game needs richer detection than marker files.

### Allow Workshop-Only Game Extensions

- Date: 2026-07-30
- Area: Extension capability model
- Decision: First-party game extensions may register a Steam app with Steam Workshop coexistence/actions and no Nexus domain or Nexus installer rules. Project Zomboid is the first example: it is registered only as a Workshop-managed/coexistence target.
- Options considered: require every extension to declare a Nexus domain; invent an unverified Nexus domain for Project Zomboid; allow extensions to be capability-specific when no verified Vortex/Nexus extension exists.
- Rationale: Project Zomboid is installed and Workshop-heavy, but the checked-out Vortex source does not include an official Project Zomboid game extension to clone. A Workshop-only extension lets DMM leave Workshop files alone and queue Steam Workshop enable/disable/subscribe/unsubscribe actions without pretending Nexus deployment is supported.
- Tradeoffs: Project Zomboid remains unsupported for Nexus archive installation until we verify a real upstream handler or design one deliberately. The extension still improves the user experience by avoiding false dirty-state warnings for Workshop content and exposing Workshop controls through the existing Decky-executed Steam API boundary.
- Verification/source references: Steam Deck installed app manifest snapshot lists Project Zomboid as AppID `108600`; local Vortex source checkout at `/tmp/dmm-vortex` has no `game-projectzomboid` extension.
- Follow-up: live-test Workshop sync/mutation with a subscribed Project Zomboid item, and add Nexus/archive support only after source verification or an explicit DMM-native design decision.

### Split Decky Quick Access From The Full DMM App

- Date: 2026-07-30
- Area: Decky UI architecture and controller focus
- Decision: Keep the Decky Quick Access Menu panel as a compact status/launcher surface and move DMM's dense tabbed workflows into a registered Steam route at `/decky-mod-manager` using Decky's native `Tabs` component.
- Options considered: keep custom tabs inside QAM and patch scroll/focus behavior; move the tab strip into `titleView`; register a full route and use native routed tabs while QAM only launches/statuses the app.
- Rationale: QAM owns global back/focus/scroll behavior, so fake tab bars and nested scroll containers were repeatedly clipped or treated as visible when only a sliver was on screen. Source review showed mature plugins use QAM for compact controls and `routerHook.addRoute` for full-page workflows. Decky's `Tabs` component is documented for active-tab state plus `autoFocusContents`, which fits the full route and avoids intercepting L1/R1 in the QAM shell.
- Tradeoffs: users now press `Open DMM` for full in-Deck management instead of doing every workflow directly inside the sidebar. The payoff is native focus, more screen space, and fewer fragile QAM layout hacks. Critical quick actions like server start/stop and phone URL stay in QAM.
- Verification/source references: AutoFlatpaks registers `/flatpak-manager`, keeps QAM as a small panel, and uses `Tabs` in the full manager page; DeckWebBrowser registers a route for its tabbed browser and keeps QAM for settings/launch; SteamGridDB registers a route for artwork workflows; `@decky/ui` `Tabs` exposes `activeTab`, `onShowTab`, and `autoFocusContents`.
- Follow-up: remove remaining sidebar-only assumptions from dense workflows, manually validate Gaming Mode navigation, and keep future Nexus/FOMOD/conflict/rollback workflows in routes or modals rather than QAM tabs.

### Restore User-Owned Config Patches Through Deployment Manifests

- Date: 2026-07-30
- Area: Deployment safety and extension lifecycle hooks
- Decision: Extension-generated changes to existing user-owned config files use restore-aware deployment mappings. The extension writes generated target content and, when the target already exists, a restore copy of the original content under a durable generated area inside DMM staging. The core validates those paths, records both in the deployment manifest, deploys the generated file as a copy, restores the original on purge/removal, and refuses purge restore if the target changed after DMM deployed it.
- Options considered: let extensions edit INI files directly; deploy full generated INI files as ordinary copy mappings; add a core-owned restore-aware mapping primitive.
- Rationale: Vortex archive invalidation mutates existing INI files, but direct extension writes would bypass preview, manifest, rollback, repair, and purge. Ordinary copy mappings would make purge delete the user's original INI. A restore-aware mapping keeps game-specific merge logic in the extension while preserving DMM's core ownership of filesystem writes and recovery semantics.
- Tradeoffs: this is still whole-file restore, not a semantic multi-owner INI merge engine. If the user edits the file after DMM deploys the generated version, purge refuses to overwrite and requires review. That is safer than clobbering user edits, but a future advanced conflict UI should offer a guided merge/restore path. Durable generated paths are required for restore-aware mappings because event work directories are rebuilt during preview/apply cycles.
- Verification/source references: Vortex `extensions/gamebryo-archive-invalidation/src/index.ts` sets `[Archive] bInvalidateOlderFiles=1` and `sResourceDataDirsFinal=""`; Vortex `gameSupport.ts` declares Fallout 4 `Fallout4/Fallout4.ini` and Skyrim SE `Skyrim Special Edition/Skyrim.ini`.
- Follow-up: expose generated config patches in advanced deployment/rollback UI, and revisit finer-grained INI patch ownership before supporting more games with complex shared config merges.

### Queue Steam Workshop Mutations As Backend-Owned Decky-Executed Jobs

- Date: 2026-07-30
- Area: Steam integration and privilege boundaries
- Decision: DMM mirrors Workshop state into the backend for UI/debug visibility, but all mutating Steam Workshop operations are queued as backend-owned `steam-workshop-action` jobs and executed by the Decky frontend through typed `SteamClient.Apps` calls. The backend atomically claims a queued action before Decky mutates Steam state, and Decky reports completion or failure back to the backend.
- Options considered: call Steam APIs directly from each UI surface; patch Workshop files on disk; make Decky own Workshop state locally; use the same backend-published/Decky-executed action pattern already used for Steam launch options.
- Rationale: SteamClient is only available in the Decky/Steam context, while the Go backend owns app state, extension capability validation, jobs, events, logging, and cross-surface UI. Queueing intent in the backend keeps one source of truth and avoids phone/tablet UI trying to call privileged Steam APIs.
- Tradeoffs: Decky must be loaded and connected for mutating Workshop actions to complete, and live testing still needs a game with real Workshop subscriptions. The payoff is clean boundaries, replayable logs, atomic duplicate suppression, and a reusable model for future Steam operations such as Workshop load order.
- Verification/source references: local `@decky/ui` SteamClient type definitions expose `SubscribeWorkshopItem`, `SetWorkshopItemsDisabledLocally`, `SetWorkshopItemsLoadOrder`, `GetSubscribedWorkshopItems`, `GetDownloadedWorkshopItems`, and `DownloadWorkshopItem` under `SteamClient.Apps`; live Decky introspection also showed these methods in the runtime `AppsWorkshop` list.
- Follow-up: add product UI for Workshop item enable/disable/unsubscribe, live-test against a subscribed Workshop game, and add load-order execution after extension load-order semantics are defined for the target game.

### Use Nexus GraphQL V2 Only Inside The Nexus Browse Adapter

- Date: 2026-07-30
- Area: Nexus catalog integration
- Decision: DMM keeps its own API REST-first, but the Nexus upstream adapter may call Nexus Mods' V2 GraphQL `mods` query for browse/search because the stable V1 REST API does not provide an equivalent mod search endpoint.
- Options considered: skip in-app Nexus browse/search; scrape Nexus web pages; force search through the phone/browser only; use Nexus V2 GraphQL inside `internal/catalog/nexus` and return normalized REST JSON from DMM.
- Rationale: the user wants Nexus browsing/search inside DMM, and Nexus' official documentation exposes searchable `mods` filters/sorts through the V2 GraphQL endpoint. Keeping GraphQL behind the Nexus adapter prevents GraphQL concepts from leaking into Decky/Svelte clients or core DMM business logic.
- Tradeoffs: Nexus V2 is documented as still in development, so search may need adjustment if Nexus changes the schema. Download links and file listing remain on the stable V1 REST API path where those operations already work.
- Verification/source references: official Nexus GraphQL docs list endpoint `https://api.nexusmods.com/v2/graphql`, the `mods` query, `ModsFilter` fields such as `gameDomainName`, `nameStemmed`, `supportsVortex`, and `status`, and `ModsSort` fields such as `downloads`, `endorsements`, `updatedAt`, and `name`.
- Follow-up: keep robust user-facing errors around search, live-test the Decky modal on the Steam Deck, and revisit the adapter if Nexus publishes stable REST/OAuth search endpoints.

### Separate Semantic Game Version From Steam Build ID

- Date: 2026-07-30
- Area: Extension framework and FOMOD dependency evaluation
- Decision: Store Steam manifest `buildid` as `steam_build_id`, store semantic installer-facing game version separately as `version`, and let first-party Go extensions register game-version provider functions. FOMOD `gameDependency` checks consume only the semantic `version` value.
- Options considered: treat Steam `buildid` as the FOMOD game version; keep game dependencies unsupported; add an extension-owned game-version provider contract and keep build ID as diagnostic/provider input.
- Rationale: Vortex delegates current game version to each game extension's `getInstalledVersion` provider, and FOMOD docs describe `gameDependency version` as a minimum game version. Steam build IDs are useful state but are not the same semantic version string, so using them directly would incorrectly satisfy some installer branches.
- Tradeoffs: games without a version provider still skip `gameDependency` branches until their extension implements version discovery. This is safer than guessing and keeps the parser/core generic.
- Verification/source references: Vortex `installer_fomod_shared/delegates/SharedDelegates.ts` calls `gameInfo.getInstalledVersion(discovery)` for `getCurrentGameVersion`; FOMOD documentation describes `gameDependency` as specifying a minimum game version.
- Follow-up: add concrete version providers while building each game extension, starting with verified Vortex source for Fallout 4, Skyrim SE, Witcher 3, and Stardew where relevant.

### Route FOMOD Installer Choices Through Extension Capabilities

- Date: 2026-07-30
- Area: Installer planning and extension architecture
- Decision: FOMOD parsing remains a generic DMM core package, but a game extension must register an installer-choice capability for kind `fomod` with the mod type and target root used to deploy selected files.
- Options considered: keep the previous generic `mod_type=fomod` planner; hardcode Bethesda/Gamebryo FOMOD roots in the server; add a generic SDK `InstallerChoiceSpec` and require extensions to opt in.
- Rationale: Vortex’s FOMOD installer delegates game-specific path handling through game support metadata such as `pluginPath`, where Fallout 4 and Skyrim SE use `Data`. DMM should mirror that boundary by letting extensions declare the destination semantics while the core validates and stages the selected files.
- Tradeoffs: games without an explicit FOMOD capability now block FOMOD archives instead of showing a choice UI. That is intentional before release because it avoids producing undeployable or wrongly rooted installs.
- Verification/source references: Vortex `installer_fomod_native/installer.ts`, `installer_fomod_ipc/installer.ts`, and `installer_fomod_shared/utils/gameSupport.ts` show `getPluginPath(gameId)` and `pluginPath: "Data"` for Fallout 4 and Skyrim SE.
- Follow-up: add stop-pattern and condition/dependency support to the installer-choice spec before claiming full FOMOD parity, then live-test with a real Fallout/Skyrim FOMOD archive.

### Compile First-Party Extension SDK Specs Into Internal Registry Records

- Date: 2026-07-30
- Area: Extension architecture
- Decision: First-party game extensions now return `internal/extensions/sdk.Extension` specs and register through the SDK registrar interface. The core compiles those specs into internal `gameext.Extension` records before building the active registry.
- Options considered: keep extensions returning internal compiled `gameext.Extension` structs directly; introduce an SDK package with specs and a compiler boundary; jump directly to runtime-loaded Go `.so` plugins.
- Rationale: direct internal structs let game packages couple to core registry implementation details. Runtime-loaded plugins add build, ABI, and packaging complexity before the SDK surface is stable. A compiled SDK spec keeps the extension authoring model in Go while creating the boundary needed for later first-party plugin packaging.
- Tradeoffs: the SDK still exposes install planning and runtime requirement domain structs, so this is not a fully external public API yet. It does remove direct `gameext` construction from Stardew and gives the core one validation point for extension outputs.
- Verification/source references: Vortex extensions register behavior through a host context rather than returning internal host state directly; DMM full Go test suite passed after converting Stardew and the first-party registry to the SDK path.
- Follow-up: harden SDK package boundaries, add lifecycle hook execution, persist extension capability snapshots for audit/debug, and defer runtime-loaded Go plugin bundles until Stardew plus at least one non-Stardew extension validate the API.

### Use Intent-Level UI Preference Patches Instead Of Whole Snapshot Writes

- Date: 2026-07-30
- Area: Shared UI state
- Decision: Game UI preferences such as favorites, recents, and sort order are owned by the Go backend. Decky and the phone/tablet web app must update them through granular intent-level patch requests and consume `ui.changed` events instead of sending whole preference snapshots from local component state.
- Options considered: keep the existing full `PUT /api/settings/ui` snapshot route; add separate routes for favorite/recent/sort; add a single `PATCH /api/settings/ui` route with optional intent fields.
- Rationale: full snapshot writes let one stale client overwrite another surface's preferences. A single patch endpoint preserves a small API surface while letting clients express "favorite this game", "record this recent game", or "set this sort" without resending unrelated state.
- Tradeoffs: patch semantics are a little more explicit than replacing a struct, and tests need to cover merge behavior. The payoff is one real source of truth with no local-only or stale overwrite paths.
- Verification/source references: live Deck state showed backend-stored preferences, while user testing showed favorites selected in the web app did not reliably appear in Decky. Code review found both clients writing complete `favorite_game_ids`/`recent_games` snapshots from local state.
- Follow-up: live-test web-to-Decky and Decky-to-web favorite/sort/recents sync, then mark the MVP tracker complete if both surfaces repaint from backend events.

### Keep Decky Sidebar Compact And Use Large Modals For Dense Workflows

- Date: 2026-07-30
- Area: Decky UI architecture
- Decision: Keep the normal Decky plugin panel inside the standard Quick Access Menu width, fix all row layouts so they never clip in that width, and use wider Decky/Steam modals for dense workflows such as Nexus search, FOMOD installers, conflict review, rollback inspection, and detailed mod metadata.
- Options considered: force the Decky/Steam Quick Access Menu wider with CSS or private class patches; keep the sidebar standard and redesign compact rows; open dense workflows in Decky modals with explicit `popupWidth`/`popupHeight`/full-size props where supported.
- Rationale: Decky owns the Quick Access Menu container and the public plugin contract exposes plugin content, not a supported per-plugin sidebar width. Widening the QAM would be fragile across SteamOS/Decky updates and could affect other panels. A compact sidebar plus large modal split preserves native Decky behavior while giving DMM enough space for search/wizard/detail workflows.
- Tradeoffs: dense workflows require one more interaction to open a modal, and modal sizing still needs live Deck validation. The payoff is a stable, professional sidebar that never relies on layout overflow or unsupported global UI mutation.
- Verification/source references: `@decky/api` plugin type exposes `content` and `titleView` but no panel sizing contract. `@decky/ui` modal props expose `popupWidth`, `popupHeight`, and full-size-related flags for modal workflows.
- Follow-up: implement Nexus search/browse as a game-scoped Decky modal backed by the Go Nexus API, and keep game/mod rows stacked and bounded inside the standard sidebar.

### Defer Runtime-Loaded Go Plugin Bundles Until After The First-Party SDK Stabilizes

- Date: 2026-07-30
- Area: Extension architecture
- Decision: MVP extensions are first-party Go packages compiled into DMM, with each extension exposing a single registration function invoked by the host. Runtime-loaded `.so` bundles are deferred until after the first-party SDK boundary has proven stable across multiple game extensions.
- Options considered: compile first-party extension packages into DMM; load first-party `.so` plugins with Go's `plugin` package immediately.
- Rationale: the user chose Go structs/registration code and accepted Go plugin ABI coupling for first-party extensions, but current MVP development still benefits from normal `go test`, cross-package refactors, Deck packaging simplicity, and direct compiler checks. The registrar/SDK boundary preserves the same authoring shape that a later `.so` loader would call, without adding plugin build/deploy failure modes before more than one extension exists.
- Tradeoffs: this does not yet prove dynamic plugin loading or binary compatibility on Steam Deck. It does prevent new game work from depending on core server/storage packages and gives us a stable migration path to runtime-loaded bundles when first-party packaging actually needs it.
- Verification/source references: Vortex extension docs model extensions as an entrypoint that registers features through a context object; the Stardew Vortex extension composes game, installer, mod type, UI, diagnostics, and runtime registrations in its `index.ts`.
- Follow-up: revisit runtime-loaded first-party bundles after Stardew plus at least one non-Stardew extension validate the Go SDK surface.

### Manage External Plugin Lists Through Deployment Plans

- Date: 2026-07-30
- Area: Deployment architecture
- Decision: Gamebryo-style generated files such as `plugins.txt` and `loadorder.txt` are produced under DMM staging and added to the normal deployment plan as copy mappings with an extension-declared external target root, rather than written as ad hoc post-deploy side effects.
- Options considered: write `plugins.txt` directly after deployment; keep plugin activation as metadata only; extend deployment mappings to support DMM-managed external target roots.
- Rationale: direct writes would bypass preview, manifest recording, purge, repair, and rollback. Metadata-only support would deploy `.esp/.esm/.esl` files without activating them, which is not useful for Bethesda/Gamebryo users. Multi-root deployment keeps generated external files inside the same managed-file model while preserving the game folder as the default root.
- Tradeoffs: deployment plans now contain an optional `target_roots` map and action-level target roots, so UI/debug surfaces must handle more than one managed root. Proton app-data path resolution is currently Steam Deck/Steam-library based and will need refinement for non-Steam or non-standard Proton users.
- Verification/source references: Vortex `gamebryo-plugin-management/src/util/PluginPersistor.ts` writes `plugins.txt` and `loadorder.txt`; `gamebryo-plugin-management/src/util/gameSupport.ts` declares per-game app-data paths, plugin formats, native plugins, and ESL support.
- Follow-up: add UI visibility for generated plugin-list actions, implement LOOT/load-order sorting, and extend the same capability to more Gamebryo games after live Fallout/Skyrim validation.

### Model Vortex Query-Mod-Path Wrapper Stripping As Extension Metadata

- Date: 2026-07-30
- Area: Install planning and extension parity
- Decision: Add `StripCommonRoot` to extension-declared installer specs, and let the core strip exactly one shared top-level archive wrapper before applying the extension's target root.
- Options considered: keep archive-root copies literal; add server-side heuristics for Fallout/Skyrim/Witcher archives; expose wrapper stripping as declarative installer metadata.
- Rationale: Vortex's installer helper exposes `stripCommonRoot`, and Vortex game registrations for Fallout 4, Skyrim SE, and Witcher 3 all use `mergeMods: true` with `queryModPath` roots. The extension must declare this behavior, while the core validates and normalizes paths.
- Tradeoffs: this handles one common wrapper level only. More complex stop-folder behavior, installer conditions, and game-specific menu/config merge logic still require additional extension SDK capabilities.
- Verification/source references: Vortex `mod_management/util/installerHelpers.ts` implements `stripCommonRoot`; Vortex `IGame.validator.json` documents `queryModPath`, `mergeMods`, and stop-folder interaction; target game registrations set `mergeMods: true`.
- Follow-up: add explicit stop-pattern metadata and lifecycle hook execution before claiming full Vortex install parity.

### Persist Extension Capability Snapshots As Audit State Only

- Date: 2026-07-30
- Area: Extension architecture and storage
- Decision: On backend startup, write the live compiled extension summaries into an `extension_snapshots` table and expose them through a debug endpoint, while keeping live Go extension registration as the only behavior source of truth.
- Options considered: do not persist extension information; store extension behavior/specs in the database; store an identity/capability snapshot for diagnostics.
- Rationale: the user wants the extension system to be inspectable and eventually packageable, but storing executable behavior in SQLite would create a second source of truth. A snapshot supports audit/debug/version review without driving installation behavior.
- Tradeoffs: snapshots help diagnose what the running backend registered, but they are not a plugin loader and do not solve runtime-loaded bundle packaging.
- Verification/source references: DMM `gameext.ExtensionSummary` is produced by the compiled first-party registry; tests verify startup sync and the `/api/extensions/snapshots` endpoint.
- Follow-up: include extension version/build metadata once first-party extension packages have explicit versions, and revisit runtime-loaded Go bundles after SDK boundaries stabilize.

### Use Go Custom Installer Hooks For Procedural Vortex Installer Logic

- Date: 2026-07-30
- Area: Extension SDK and Vortex parity
- Decision: Add extension-owned custom match/build hooks to installer specs for games whose Vortex extensions use procedural archive transforms that cannot be represented cleanly as simple declarative copy modes.
- Options considered: keep adding core instruction modes for each special case; force every installer into declarative metadata; expose Go custom installer hooks on first-party extension specs.
- Rationale: Witcher 3's Vortex extension uses procedural installers for menu mods, mixed mod/DLC archives, content-only archives, and DLC transforms. Putting those transforms in DMM core would be game-specific hardcoding. Go hooks preserve the chosen first-party Go extension model while keeping server/storage/deployment responsibilities in core.
- Tradeoffs: custom hooks are executable code and therefore require stronger SDK boundaries and validation. DMM mitigates this by still validating/staging returned copy instructions in the core path before deployment.
- Verification/source references: Vortex `extensions/games/game-witcher3/src/installers.ts` registers `witcher3menumodroot`, `witcher3mixed`, `witcher3tl`, `witcher3content`, and `witcher3dlcmod` as procedural installers; DMM tests cover these archive shapes.
- Follow-up: define a stable public hook context before runtime-loaded Go plugin bundles, and add lifecycle hook execution for deploy/profile events.

### Declare FOMOD Stop Folders In Game Extensions

- Date: 2026-07-30
- Area: FOMOD planning and extension parity
- Decision: FOMOD stop-folder/plugin-path metadata belongs in each game extension's installer-choice capability. The generic FOMOD planner consumes those declared folders to strip wrapper paths before applying the extension target root.
- Options considered: leave FOMOD paths literal; hardcode Bethesda stop folders in the core planner; add extension-declared stop folders to the SDK `InstallerChoiceSpec`.
- Rationale: Vortex passes `getStopPatterns(gameId, game)` and `getPluginPath(gameId)` into its FOMOD installer. Fallout 4 and Skyrim SE share Gamebryo stop folders but differ in script-extender/tool names, so DMM needs reusable helpers plus per-extension declarations instead of core game-specific branching.
- Tradeoffs: this handles Vortex stop-folder path normalization but still does not implement file/game dependency predicates, installer-side dependency checks, or LOOT/plugin sorting. Games without a FOMOD installer-choice spec still block rather than guessing.
- Verification/source references: Vortex `installer_fomod_native/installer.ts` and `installer_fomod_shared/utils/gameSupport.ts` show stop pattern and plugin path handling for Gamebryo/Bethesda games.
- Follow-up: add deployed-file dependency evaluation once DMM has a verified profile/deployment-state contract, then live-test a real Fallout 4 or Skyrim SE FOMOD archive.

### Execute Extension Deployment Hooks Through Validated Mappings

- Date: 2026-07-30
- Area: Extension lifecycle and deployment safety
- Decision: DMM executes extension lifecycle hooks through typed Go SDK handlers. `will-deploy` may return additional deployment mappings from DMM-managed generated paths; ephemeral generated files may use the hook work directory, while restore/profile state must use durable staging paths. `did-deploy` and `did-purge` are notification-style hooks whose returned mappings are ignored and logged.
- Options considered: keep lifecycle handlers as capability names only; let extensions mutate game files directly during lifecycle events; let `will-deploy` return extra mappings that the core validates and applies through the normal deployment plan.
- Rationale: Vortex game extensions subscribe to events such as `will-deploy`, `did-deploy`, and `did-purge`. Letting DMM extensions write game files directly would bypass preview, conflict detection, rollback, purge, and manifest recording. Mapping-return hooks preserve extension ownership of game-specific generation while keeping filesystem writes under DMM core control.
- Tradeoffs: complex Vortex hooks that mutate application state or depend on Redux-style profile changes need further API surface. Post-deploy hooks cannot alter the just-applied deployment in this model; anything file-affecting must happen during `will-deploy`.
- Verification/source references: Vortex `mod_management/index.ts` emits `will-deploy` before the final deployment, and game extensions such as Witcher 3 and Stardew Valley register `will-deploy`/`did-deploy`/`did-purge` handlers.
- Follow-up: migrate concrete Witcher 3 menu/config merge generation and Stardew runtime/profile actions onto this hook contract, then add user-visible hook failure reporting for critical pre-deploy failures.

### Allow Extension Hooks To Replace Deployment Mappings

- Date: 2026-07-30
- Area: Extension lifecycle and load-order deployment semantics
- Decision: Extend `will-deploy` hook results with `ReplaceMappings` so an extension may return a full transformed mapping set when Vortex-compatible behavior changes deployed paths, while the core still validates and applies those mappings through the normal deployment plan.
- Options considered: let extensions mutate target paths directly; add game-specific path rewriting to the server; make hooks append only generated files; allow typed extension hooks to replace the mapping set.
- Rationale: Vortex `mergeMods` handlers can alter the deployed mod folder name for the whole mod, as seen in Unreal pak game extensions that prefix folders with `AAA`, `AAB`, etc. Append-only hooks are enough for generated files like `mods.settings`, but not for path transforms where the original unprefixed mappings must not deploy. A replacement result keeps that behavior in the game extension and keeps final filesystem writes under core preview/conflict/rollback control.
- Tradeoffs: replacement hooks need stronger validation because a bad extension can rewrite many targets. DMM mitigates this by passing the replaced mappings back through the same deploy planner and by tagging mappings with the local installed-mod ID so transforms can group files by DMM-owned mod identity instead of source archive guesses.
- Verification/source references: Vortex `game-codevein`, `game-spyroreignitedtrilogy`, and `game-bloodstainedritualofthenight` extensions use `mergeMods` plus load-order serialization to prefix pak mod folders; their `makePrefix` helper produces the `AAA`, `AAB`, `AAC` pattern.
- Follow-up: expose transformed target folders in advanced deployment preview and reuse the helper for other Unreal pak games when their Vortex source/extensions are verified.

### Generate Witcher 3 Managed Load Order As A Deployment Mapping

- Date: 2026-07-30
- Area: Extension lifecycle and external target deployment
- Decision: The Witcher 3 extension generates a basic DMM-managed `mods.settings` file during `will-deploy` and returns it as a copy mapping to the game Proton Documents folder. The core still validates and applies that mapping through the normal deployment manifest, rollback, repair, and purge path.
- Options considered: advertise Witcher load-order support without implementation; let the hook write `mods.settings` directly; generate a managed file under the hook work directory and return a deployment mapping.
- Rationale: Vortex's Witcher extension writes `mods.settings` from enabled managed mod folder names during deploy/load-order serialization. DMM can safely support the managed subset now by deriving folder names from extension-produced target mappings, while leaving merged/manual/profile-specific Vortex semantics incomplete.
- Tradeoffs: this does not yet implement Vortex's menu XML merge, Script Merger prompts, manual mods.settings preservation, collection load order import, or profile merge backups. It does establish the safe external-target pattern needed for Witcher and similar Proton games.
- Verification/source references: Vortex `game-witcher3/src/eventHandlers.ts`, `iniParser.ts`, `loadOrder.tsx`, `util.ts`, and `common.ts`.
- Follow-up: add profile/load-order state to the SDK before implementing Vortex's advanced Witcher merge/manual semantics, then live-test on a clean Witcher 3 install.

### Inject FOMOD File Dependency State Into The Planner

- Date: 2026-07-30
- Area: FOMOD installer planning
- Decision: FOMOD parsing remains pure archive/XML parsing. Conditional `fileDependency` evaluation is driven by explicit file-state maps or a caller-provided resolver, and the server provides a resolver from the current game target path plus extension plugin-activation state when applying installer choices.
- Options considered: keep all file/game dependencies unsupported; let the FOMOD parser read the game folder directly; inject file state through `PlanOptions`; derive plugin `Active`/`Inactive`/`Missing` from the game extension's plugin activation capability.
- Rationale: FOMOD file dependencies are relative to the installer destination folder, but only the server knows the current game path, extension target root, active profile, and plugin activation contract. Injecting a resolver keeps the parser reusable, while extension-declared plugin activation gives Bethesda/Gamebryo FOMODs a generic way to distinguish active and inactive `.esp/.esm/.esl` files without game-specific core branches.
- Tradeoffs: inactive support currently depends on DMM-managed profile/plugin activation state and the deployed plugin-list file when available. It is not a full LOOT/manual-load-order model yet, and game dependency predicates are still blocked until DMM has a verified game-version contract.
- Verification/source references: FOMOD documentation describes `fileDependency` states as `Active`, `Inactive`, and `Missing`; Vortex FOMOD IPC docs expose `checkIfFileExists`, `isPluginPresent`, and `isPluginActive`; DMM tests cover active, missing, and disabled-profile inactive plugin dependency behavior.
- Follow-up: add game-version dependency support after extension game-version discovery is defined, then verify with a real Fallout 4 or Skyrim SE FOMOD that uses plugin dependency predicates.

### Declare Ignored Deployment Conflict Patterns In Extensions

- Date: 2026-07-30
- Area: Extension framework and deployment planning
- Decision: Add an extension-owned conflict-ignore capability and let the generic deploy planner skip matching targets as non-blocking, non-overwriting actions. The core consumes validated patterns from the active game extension and never branches on a specific game.
- Options considered: ignore Fallout 4's Vortex `ignoreConflicts` detail for now; hardcode the Fallout path in deployment; add generic extension metadata and keep unmanaged-file overwrites blocked.
- Rationale: Vortex's Fallout 4 extension declares `**/PersistantSubgraphInfoAndOffsetData.txt` as an ignored conflict. DMM needs a generic way to model that Vortex behavior without moving Fallout-specific knowledge into the backend. Skipping a matching unmanaged target is safer than silently overwriting files DMM did not create.
- Tradeoffs: this implements the safe planning semantics, not full Vortex conflict UI parity. Future conflict UI still needs per-file winners, rules, and power-user inspection.
- Verification/source references: Vortex `extensions/games/game-fallout4/src/index.js` declares `details.ignoreConflicts` with the persistent subgraph offset pattern.
- Follow-up: expose ignored conflict rules in advanced deployment details and add richer per-file winner/load-rule UI before claiming full Vortex conflict parity.

### Extend Gamebryo Activation With Native Plugin Manifests

- Date: 2026-07-30
- Area: Extension framework and Gamebryo plugin activation
- Decision: Add `NativePluginManifests` to the extension-owned plugin activation spec and have the backend read those game-root manifest files when generating `plugins.txt`/`loadorder.txt` and evaluating FOMOD plugin dependency state.
- Options considered: keep static native plugin lists only; hardcode `Fallout4.ccc` and `Skyrim.ccc` in the backend; declare manifest filenames in each Gamebryo extension.
- Rationale: Vortex augments native plugin lists from `.ccc` files during Gamebryo initialization. The file names are game-specific, so they belong in the game extension, while parsing and validation are generic backend responsibilities.
- Tradeoffs: this supports installed native plugins listed in `.ccc` files, but it does not implement LOOT sorting or original-format timestamp ordering for older Gamebryo games.
- Verification/source references: Vortex `gamebryo-plugin-management/src/util/gameSupport.ts` calls `applyNativePlugins(api, "skyrimse", "Skyrim.ccc")` and `applyNativePlugins(api, "fallout4", "Fallout4.ccc")`.
- Follow-up: add the same manifest metadata to future Gamebryo extensions and implement richer plugin sorting/validation before claiming full Bethesda parity.

### Seed FF7 Rebirth From Verified Extension Page Until Source Is Available

- Date: 2026-07-30
- Area: First-party game extension coverage
- Decision: Add a partial Final Fantasy VII Rebirth first-party Go extension from the verified Nexus Vortex extension page and independently verified Steam/Proton executable layout, but keep advanced behavior marked incomplete until the extension package/source can be inspected.
- Options considered: skip FF7 Rebirth entirely until source is available; implement only Steam/Nexus IDs; implement conservative installer hooks for the archive shapes explicitly described by the Vortex extension page.
- Rationale: the user asked for FF7 Rebirth as a near-term target, and the official extension page documents the supported mod categories, no-symlink constraint, Nexus domain, version history, and target folders. That is enough to support common pak, FF7RML, UE4SS, binary, and root-folder archive shapes without encoding arbitrary mod-specific hacks.
- Tradeoffs: this is not full one-for-one Vortex parity because the actual extension code was not in the main Vortex source checkout. Config/save installers, load-order details, partition checks, and edge-case installer priority may differ until the package/source is obtained.
- Verification/source references: Nexus mod page `site/mods/1150` for the FF7 Rebirth Vortex extension; Valve Proton issue `2909400` executable listing for `End/Binaries/Win64/ff7rebirth_.exe`.
- Follow-up: inspect the actual extension package/source when authenticated download access is available, then adjust installer priority, load-order, config/save, and no-symlink deployment behavior to exact parity.

## Steam Workshop Coexistence Before Workshop Management

Decision: MVP will support coexistence with Steam Workshop content before implementing full Workshop provider management.

Options considered:
- Treat any Workshop content as dirty external mod state and block DMM deployment. This is safe but too restrictive for games where Workshop mods live outside DMM deployment targets.
- Ignore Workshop entirely. This avoids work but creates confusing review warnings and may incorrectly classify Steam-managed content as unmanaged local modifications.
- Detect Workshop content as Steam-owned external content, surface it in diagnostics, and allow DMM deployment when the game extension says coexistence is safe. Full Workshop subscribe/enable/disable/load-order control remains post-MVP.

Choice: detect and label Workshop content, leave it untouched, and allow DMM-managed deployments alongside it where extension policy permits. Deployment conflict checks still block actual target collisions in game folders/appdata.

Rationale: this matches the current product goal: deploy Nexus/Vortex-compatible mods without forcing users to purge Steam Workshop mods. It keeps ownership boundaries clear: Steam owns workshop content, DMM owns its staging/deployment manifest, and extensions declare when coexistence is safe for a game.

Tradeoffs: diagnostics and game detection become slightly more nuanced. Some games merge Workshop content into app-specific mod folders, so coexistence must be extension-reviewed rather than globally assumed.

Follow-up: add Steam Workshop discovery for `steamapps/workshop/content/{appID}` and relevant app manifests; add extension policy for workshop coexistence; update Review UI copy; keep Workshop enable/disable/provider integration post-MVP.

Implementation update:
- Steam discovery now detects Workshop content and app workshop manifests separately from unmanaged game-folder markers.
- Game extensions declare Steam Workshop coexistence through the Go SDK. First-party target extensions currently declare coexistence only, not active management.
- The extension SDK includes a Workshop action declaration shape for future `subscribe`, `unsubscribe`, `enable`, and `disable` actions. Those actions are metadata/contracts only until the Decky frontend Steam API executor is verified.
- Product boundary: DMM core should persist desired Workshop management state, jobs, and audit logs; Decky should execute verified Steam frontend APIs; extensions should define whether enable/disable is a Steam subscription operation, a load-order/plugin state operation, or unsupported for that game.
- Live Decky API observation: `SteamClient.Workshop` and `SteamClient.UGC` were empty in the Decky runtime, but `SteamClient.Apps` exposed Workshop-related methods such as `DownloadWorkshopItem`, `GetDownloadedWorkshopItems`, and `GetSubscribedWorkshopItemDetails`. The logged method list was truncated, so the next investigation step is to log the full `SteamClient.Apps` method list and verify signatures before implementing subscribe/unsubscribe/enable-disable actions.

## Captured Install Route Replaces Pending Import Route

Decision: replace the old `/api/imports/pending` route family and `pending-import` job type with `/api/captured-installs` and `captured-install`.

Options considered:
- Keep old pending/import naming and hide it in the UI. This reduces code churn but preserves a stale model in the product core.
- Add aliases from old routes/job types to new routes/job types. This is safer for released clients, but DMM has no release boundary yet and compatibility aliases would violate the pre-MVP cleanup guideline.
- Break the API now and update every first-party caller/test in the same change.

Choice: break the API now and remove the old route/job type without an alias.

Rationale: the product model changed: Nexus `nxm://` links are captured and downloaded immediately while valid; the user's decision is about installing the cached archive. Keeping "pending import" around would keep old download-approval thinking embedded in source, tests, and API payloads.

Tradeoffs: existing local pending captured jobs from older test builds will not restore through the new route/table. That is acceptable pre-MVP; test devices can recapture links or reset captured actions.

### Stop Catalog Dispatch On Provider-Owned Failures

- Date: 2026-07-31
- Area: Remote provider architecture
- Decision: Catalog resolvers return `catalog.ErrUnsupportedURL` only when a URL clearly does not belong to that provider. The server continues to the next resolver only for that sentinel error; all provider-owned parse, credential, API, and metadata failures stop dispatch and return the real error.
- Options considered: keep trying every resolver until one succeeds; require clients to send an explicit catalog name; use a sentinel unsupported error with provider-ordered dispatch.
- Rationale: direct archive import is intentionally a catch-all for generic HTTP archive URLs, but it must not catch a Thunderstore, Nexus, GitHub, or ModDB URL whose provider resolver failed. Provider-owned failures need to be diagnosable and fixed in that provider adapter instead of being hidden by a less-specific download path.
- Tradeoffs: some malformed URLs may now fail earlier with a provider-specific error rather than falling through to direct. That is preferred for MVP because it prevents silent wrong-source installs and keeps source tags trustworthy.
- Verification/source references: Thunderstore official OpenAPI schema exposes `/api/experimental/package/{namespace}/{name}/` and `/api/experimental/package/{namespace}/{name}/{version}/` with `download_url`; live API requests confirmed those endpoints and package download redirects.
- Follow-up: apply the same ownership/error discipline to GitHub, ModDB/mod.io, and CurseForge adapters as each provider is verified.

### Provider Capability API Is A Read Model

- Date: 2026-07-31
- Area: Remote provider architecture
- Decision: Expose provider/source readiness through a backend `GET /api/catalogs` read model instead of storing provider capability metadata as a separate behavior source of truth.
- Options considered: hardcode source labels only in each UI; store provider capabilities in config/database and let UI infer support from that; expose a backend read model generated from registered resolvers plus current credentials and planned-provider policy.
- Rationale: UI-only labels drift quickly as providers are added, while config/database-owned capabilities would become a second truth separate from compiled resolvers. A backend read model keeps the resolver registry authoritative for actual behavior, while still letting web/Decky surfaces show ready, credential-gated, planned, deferred, platform, and source-tag state consistently.
- Tradeoffs: planned providers are still described in backend code until they become real resolvers/config entries. That is acceptable because the list is product status, not executable install behavior.
- Verification/source references: the current resolver registry contains Nexus, Thunderstore, GitHub Releases, and direct archive resolvers. mod.io and CurseForge official APIs require credentials before DMM can claim working import/download support. ModDB remains deferred until an official supported automated API/client path is verified.
- Follow-up: as mod.io, CurseForge, ModDB, local archive, and future providers are implemented, migrate each from planned/deferred status to real resolver/config-backed readiness and add source filtering/sorting over installed mod lists.

### Provider Credentials Rebuild Resolver Registry

- Date: 2026-07-31
- Area: Remote provider architecture
- Decision: Store mod.io and CurseForge API keys in the normal DMM config, expose only configured booleans/readiness through APIs, and rebuild the in-memory resolver registry when provider settings change.
- Options considered: require a backend restart after editing provider keys; read provider keys directly from config inside each resolver call; rebuild the resolver list after settings writes.
- Rationale: a restart requirement would make provider setup feel broken from the phone/tablet UI. Reading mutable config inside every resolver would leak server config concerns into provider adapters. Rebuilding the resolver list after a settings write keeps providers as plain Go objects with explicit dependencies while preserving immediate UI-driven setup.
- Tradeoffs: resolver instances are still process-local and first-party compiled. Future account/session providers may need a credential store abstraction once OAuth, token refresh, or per-provider validation flows are added.
- Verification/source references: mod.io official OpenAPI requires `api_key` security for game/modfile endpoints and returns `download.binary_url` on modfile objects; CurseForge official REST docs require `x-api-key` and expose `GET /v1/mods/{modId}/files/{fileId}/download-url`.
- Follow-up: add validation endpoints for provider keys after live keys are available, and move from config-file secrets to an OS/keyring-backed secret store if auth/pairing expands beyond local MVP.

### Event-Hook Deployment Mode For Packed Archives

- Date: 2026-08-08
- Area: Extension/deployment framework
- Decision: Add an extension-declared `event-hook` mod-type deployment mode. Normal mod types still require direct staged-file target mappings, but `event-hook` mod types may stage targetless DMM-owned payload files so a game extension can generate deployment mappings during `will-deploy`.
- Options considered: special-case MGSV/SnakeBite in core; allow every staged manifest to contain targetless files; add a generic extension-declared deployment mode.
- Rationale: SnakeBite source shows MGSV mods mutate packed `00.dat`/`01.dat` QAR/FPK archives instead of deploying loose files directly. That behavior belongs in the game extension, but the core deployment planner must allow the extension to own staged payloads before it generates safe, restore-aware deployment outputs.
- Tradeoffs: this does not implement the QAR/FPK writer by itself. It creates the framework shape needed for packed-archive extensions while preserving strict errors for normal targetless manifests.
- Verification/source references: SnakeBite `ModManager.cs`, `BackupManager.cs`, and GzsTool QAR/FPK sources; DMM focused tests `TestDeployRunsExtensionWillDeployHookMappings` and `TestDeployRejectsTargetlessManifestForDirectModType`.
- Follow-up: implement a source-verified SnakeBite payload planner and QAR/FPK generation hook in the MGSV extension, then apply the same mode only to other packed-archive extensions that require it.
