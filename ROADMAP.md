# Decky Mod Manager Roadmap

This roadmap tracks the remaining work for the first usable MVP and separates it from the work that should follow immediately after MVP.

## MVP Goal

Install a Nexus Mods "Mod Manager Download" / `nxm://` mod from Gaming Mode on Steam Deck, capture and cache the archive immediately, optionally approve the local install from a phone or tablet, add it to the selected game/profile disabled by default, let the user enable/disable it without understanding staging internals, and provide safe deployment, logging, and rollback behavior underneath that simple workflow.

## MVP Status

### Current Decisions

- First vertical-slice game: Stardew Valley (`413150`, Nexus domain `stardewvalley`).
- Highest implementation priority is Stardew extension-framework parity with the required behavior from Vortex's Stardew Valley extension, including SMAPI installer handling, SMAPI mod install rules, runtime requirements, primary launch tool behavior, and Steam Deck launch configuration.
- The next required MVP feature after Stardew extension parity is FOMOD/installer-choice support with Decky modal support for Deck-only flows.
- Stardew Valley MVP support targets the native Linux Steam install on Steam Deck. Windows/Proton support is post-MVP and must remain extension-driven when implemented.
- The default user mental model is profile-first: select a game, select a profile, capture/download a mod immediately, approve install when needed, then enable or disable that mod in the profile.
- The UI must not make ordinary users reason about staging, install planning, target mappings, manifests, or deployment transactions before they can use a mod.
- Staging, preview, deploy, purge, repair, blocked install plans, and file-level operations are still core safety mechanisms, but they should be exposed as advanced/power-user views unless immediate user action is required.
- Nexus automatic download links may be short-lived, so DMM should download/cache captured `nxm://` links immediately and move user approval to the install/stage/profile decision.
- Decky Settings owns `Auto-install captured downloads`. It defaults on for MVP Deck-only flow, skips the phone/tablet local-install confirmation after the archive is cached, and must be easy to disable.
- Decky Settings owns `Auto-enable installed mods`. It defaults off for safety; when enabled, DMM may install, enable, and deploy profile changes if there are no conflicts or installer-choice prompts.
- ZIP, 7z, and RAR archives are all MVP-relevant because Nexus mods commonly use all three.
- The internal safe path remains: capture request, immediately download/cache the archive, inspect it, produce an explicit install plan, ask for install choices/approval when required, stage only deployable outputs, update the selected profile's mod list disabled by default, preview/apply deployment through DMM-owned manifests, and report failures clearly.
- The primary happy path should feel like: capture/download mod immediately, approve install if prompted or let auto-install continue, see the mod in the selected profile disabled by default, then toggle it on/off from the phone/tablet UI or Decky plugin.
- Gaming Mode must show immediate Decky feedback after Nexus hands DMM a download link. A user should not see only the Nexus "download starting" page and then silence; the Decky plugin should notify when a request is captured, waiting for approval, processing, completed, or failed.
- Deck-only MVP flow must let a user browse Nexus in Gaming Mode, click Mod Manager Download, let DMM auto-install according to Decky settings, then use the Decky plugin `Mods` surface for the running game to enable or disable mods without needing a phone.
- DMM must follow a Vortex-style virtualized deployment model: archives and extracted mods live in DMM storage, profiles define enabled staged mods, and game folders receive only DMM-owned deployment artifacts tracked by manifests.
- Profile switching must reconcile manifests and links, not mutate staged mods or overwrite unmanaged game files.
- Like Vortex, DMM must separate download, install planning, staging, and deployment. A downloaded Nexus archive is not automatically a deployable mod until a game/provider installer plan has identified its mod type, deployable files, and target mapping.
- The Go backend publishes required launch-tool actions, while the Decky frontend executes Steam frontend-only capabilities such as setting Steam launch options. This is an explicit service contract, not an ad hoc callback path.
- The next architecture step is a durable backend queue plus typed domain events delivered over WebSocket. The queue owns work execution, retries, cancellation, and crash recovery; WebSocket owns realtime phone/tablet updates and Decky modal flows such as FOMOD choices.
- Game list favorites and sorting are implemented for the phone/tablet web drawer: mobile web defaults to `Recent`, supports favorites pinned above normal games, and offers `A-Z`/`Z-A` sorting.

### Decky Mods UX Requirements

- Decky should use tabs where they create more usable sidebar space than dropdown-heavy navigation.
- The existing Decky Mods UI should be promoted/polished as a first-class `Mods` tab or equivalent tab-level surface, not hidden behind a click-through button.
- The `Mods` surface should prioritize the active game, active/default profile, installed mod rows, enable/disable toggles, search/filter, and compact status labels.
- Mod rows must be focusable with D-pad/controller navigation, visibly highlight when focused, and scroll correctly without requiring pointer clicks.
- Staging, deployment preview, manifests, target mappings, purge, repair, and recovery belong in advanced/debug surfaces unless an error requires user action.
- Enabling or disabling a mod from Decky should update the selected profile and apply profile changes automatically, while clearly noting that an already-running game may need a restart.
- If no game is running or the running game is unsupported, Decky should show a compact default state with a game selector or a clear unsupported-game message.

### Done

- Decky plugin package that bundles the Go backend, Svelte web UI, and NXM handler.
- Decky plugin starts and stops the backend process directly.
- Decky plugin auto-starts the Go backend when the plugin loads, so the phone/tablet UI can come back after reboot without manually opening the Decky panel first.
- Decky plugin shows backend status, phone/tablet URL, Nexus link, dependency status, pasted URL import, and NXM handler registration tools.
- Backend serves REST APIs and the mobile/tablet web UI.
- LAN-only access enforcement is enabled by default.
- LAN-only middleware coverage proves public remotes are rejected while LAN/loopback requests pass, and the trusted-tunnel toggle disables that rejection.
- Sparse or older config files preserve secure defaults such as LAN-only access, the default listen address, and default data directory. MVP fast-install settings default on, but must remain explicit Decky settings.
- Steam library and installed-game discovery.
- SQLite storage for discovered games and profiles.
- Storage migration handles older MVP database shapes by adding missing columns introduced during development.
- Default profile creation and profile switching APIs.
- Nexus API key configuration and validation path.
- Nexus HTTPS URL parsing.
- Nexus `nxm://` parsing for real Nexus path-style links.
- Captured URL parsing now goes through registered catalog resolvers instead of directly calling Nexus parsing from the server; Nexus remains the only download-link provider in the MVP.
- Nexus game domains and mod/file IDs are normalized and validated before they are used in DMM download or staging paths.
- Firefox Flatpak / portal stale-handler issue documented through logs and fixed manually during testing.
- User-level `nxm://` handler registration.
- NXM handler logs redacted URLs and forwards captured installs to the backend.
- Captured installs appear in the phone/tablet Action Center when local install choices or explicit install approval are required.
- Captured installs can be approved from the phone/tablet UI when automatic install is disabled or a user decision is required.
- Install approval is guarded server-side so stale captured metadata cannot move a failed, completed, canceled, or already-running install job back to running.
- Captured installs and active download/extraction work can be canceled from the phone/tablet UI.
- Failed downloads and retryable install failures keep captured install metadata and can be retried from cached state where possible; unsupported installer/layout failures still move to blocked install candidates instead of retrying blindly.
- Clearing visible captured installs preserves completed install history in Jobs while removing waiting, running, queued, failed, and persisted action-center state.
- Captured install records, resolved links, and jobs are persisted across backend restarts.
- Jobs now persist structured source/game metadata, and captured-install storage backfills that payload when actions are restored or attached internally, so the phone/tablet UI can scope install actions and game operations by app ID or Nexus game domain instead of parsing user-facing job text.
- Existing MVP-era SQLite databases are upgraded in place for persisted jobs, captured installs, install manifests, profile priority, source metadata, and checksums.
- Interrupted downloads restore as waiting install actions after backend restart and persist that corrected state, so the user can continue from cached state instead of getting stuck with a stale running job.
- Endpoint-level coverage proves a captured Nexus request can download a Nexus archive over HTTP, install it from cached storage, record it as a disabled Stardew profile mod, and clear the captured-install action.
- Coverage uses an extensionless Nexus-CDN-style download URL, proving archive signature detection in the normal install path and not only the recovery path.
- Captured installs download archives into DMM-managed storage.
- Captured downloads are recorded with archive path and source metadata.
- Downloaded archives, staged files, and deployment manifests record SHA-256 checksums where files are locally available.
- Captured Stardew Valley installs can be inspected, extracted into managed storage, persisted as profile mods, and shown in the Plugins tab.
- Install planning now separates extraction from deployable staging: archives are extracted to a temporary workspace, the game install planner emits explicit deployable file instructions, and only planned outputs are copied into DMM staging.
- Install planning is now driven by Vortex-modeled game metadata specs rather than one-off Nexus file IDs, filenames, or procedural per-mod rules.
- Installer metadata extractors are now declared by the game spec, so manifest validation and attribute ingestion are not hardcoded into the planner core. The generic JSON manifest extractor can ingest common fields such as display name, unique ID, version, entry DLL, and minimum API/runtime version for future Vortex-style game specs without new parser code.
- Vortex game ID to Steam app ID lookup now comes from the same install-plan registry instead of server-side game/domain switches, keeping import routing aligned with supported game specs.
- Deployment eligibility now comes from supported game specs as well. The server asks the registry whether a Steam app ID is deployable and whether review-state deployment is allowed, instead of carrying game-specific deployment branches.
- Download recovery is routed through the same game/domain registry, so orphaned Nexus archives are only restaged for games with a registered Vortex metadata spec and supported Nexus domain.
- Mod type deploy roots are now modeled separately from installer selection. Installers choose a mod type and emit install instructions; the selected mod type supplies the deploy root, matching Vortex's installer/mod-type split and avoiding per-installer path duplication.
- The current Stardew spec preserves Vortex's installer/mod-type separation for the MVP slice: `stardew-valley-installer` maps valid manifest-based mods into `Mods/`, `sdvrootfolder` maps root-folder archives to game-root targets, and `smapi-installer` extracts the Linux embedded payload into game-root targets.
- Verified SMAPI archive payloads contain separate Linux and Windows inner payloads. MVP targets the Linux payload for native Stardew; post-MVP Windows/Proton support must select the Windows payload from detected game runtime through extension metadata.
- Install-plan metadata can mark individual target mappings with policy such as `keep-existing`, which lets SMAPI preserve an existing game-owned `steam_appid.txt` without treating the whole deployment as conflicted.
- Install-plan metadata controls staged mod display naming, so installer/runtime packages such as SMAPI can use the Nexus archive name while normal manifest-based mods use their manifest display name without adding server-side mod-type special cases.
- SMAPI manifest parsing now follows Vortex's tolerant shape for common real-world archives by accepting UTF-8 BOMs and trailing commas before validating manifest identity. SMAPI manifest attributes are now extracted through a spec-declared metadata extractor, including logical file names, name, unique ID, version, entry DLL, minimum API version, content-pack target, and dependencies.
- New staged manifests preserve the install planner ID and detection evidence, so DMM can trace a staged mod back to the Vortex-style archive/layout metadata that classified it.
- Game diagnostics use a declarative runtime requirement registry keyed by game and mod type. For the Stardew slice, this reports missing SMAPI/runtime state and required manifest dependency mods from staged metadata without adding one-off Nexus mod/file rules to server/UI code.
- Deployment planning now consumes stored install-plan target mappings from staged manifests and fails clearly when an installed record lacks a deployable manifest.
- Unsupported installer/layout results are persisted as blocked install candidates and exposed through Action Center instead of being treated as deployable mods.
- Blocked install candidates can be cleared from Action Center after the user has reviewed the reason.
- Installed mods can be removed from the Plugins UI; this deletes the installed-mod row and DMM-managed staging directory while preserving downloaded archives for recovery/reinstall.
- Download recovery restages cached Nexus archives that are not already installed. Invalid installed records are surfaced as recovery-needed and must be removed or reinstalled through the current profile/captured-install path; DMM does not keep a separate legacy recovery state.
- Failed archive extraction removes partial staging output before the failed job is surfaced, so broken installs do not leave deployable-looking debris.
- Repeated downloads/reinstalls of the same mod keep download history but show one profile mod row.
- Endpoint-level coverage proves duplicate captured installs of the same Nexus mod/file still show one installed plugin row in the game UI.
- Deployment planning fails clearly when an installed mod record points at a stale or missing staging path. Pre-MVP DMM does not preserve hidden path-recovery fallbacks; fixing the current data model is preferred over keeping obsolete compatibility behavior.
- Installed mod records are attached to the default/selected profile and can be enabled, disabled, or reprioritized from the Plugins tab.
- Profile mod enable/disable and priority updates validate that the selected profile and staged mod belong to the same game before writing profile state, preventing cross-game profile contamination.
- Installed mod records can be removed from the Plugins tab without deleting the cached downloaded archive.
- Stardew Valley staged plugins can generate a deployment preview targeting `Mods/`.
- Stardew Valley deployment preview is profile-aware and shows kept, added, replaced, removed, and conflicting deployment artifacts.
- Switching to a profile with no enabled mods can still generate remove actions for the current deployed profile instead of erroring.
- Endpoint-level coverage proves deploying an empty profile removes the current profile's DMM-owned links, clears active deployment files, and leaves staged mod records untouched.
- Reset managed mods reconciles DMM-owned deployment manifests, removes DMM-owned deployed artifacts, clears installed mod rows and staging folders, clears installer candidates, and keeps cached downloads available for recovery.
- Stardew Valley deployment can be explicitly applied from the preview when there are no conflicts.
- Manual and automatic deployment refuse no-op plans that contain only kept/skipped files, so users do not get misleading zero-change deploy jobs.
- Deployment manifests are persisted in SQLite.
- Active deployment status is exposed through the API and shown in the Plugins deployment card, so users can distinguish pending preview changes from files that are currently deployed.
- Existing DMM-owned links are manageable during profile transitions; unmanaged files remain conflicts.
- Duplicate target files are resolved deterministically by simple profile mod priority; priority is profile-scoped and editable in the Plugins tab, and losing mappings are shown as skipped preview actions.
- Purge removes only manifest-owned files and leaves unmanaged files and parent directories intact.
- Job list and WebSocket domain events for captured-install/download state.
- Archive inspection for ZIP safety checks.
- ZIP extraction support.
- Extensionless Nexus CDN archive paths are detected by file signature, so valid ZIP/7z/RAR downloads are not rejected only because the URL path is a GUID.
- 7z/RAR inspection and extraction support through external helper tools, with traversal/FOMOD detection when `7z` listing is available.
- Archive helper failures now produce actionable messages that name the missing/failing helper and point users to the Decky plugin Dependencies view.
- Game Plugins includes a recovery action that scans DMM-managed downloads and stages orphaned archives that failed before install records were created.
- Stardew/SMAPI `manifest.json` names are used for recovered/staged mods when archive names are missing or GUID-like.
- FOMOD archives are detected and fail with an explicit unsupported-installer message instead of being staged incorrectly.
- Deployment planner/apply skeleton with hardlink, symlink, and copy strategies.
- Deployment apply verifies links/files before recording the manifest.
- Replacement/removal deploy actions keep temporary backups and restore DMM-owned files when apply verification fails.
- Deployment applies through a prepared transaction: file changes are committed only after manifest persistence succeeds, and uncommitted file changes are rolled back on verification or manifest-record failures.
- Repair endpoint and UI action recreate missing/broken DMM-managed links and refuse to overwrite unmanaged files.
- Dirty/non-clean game state is blocked for non-MVP games before deployment.
- Decky plugin primary UI is split into Manage, Settings, and Debug.
- Decky Settings now focuses on routine server access and dependency status, while NXM registration/testing lives in Debug to keep the main Deck workflow less crowded.
- Decky Debug view can load diagnostic tails for plugin, backend, NXM handler, and Steam JS logs.
- Game diagnostics endpoint summarizes profiles, staged mods, active jobs, blocked candidates, deployment state, preview readiness, and validation warnings for live MVP checks.
- Game diagnostics now includes handler-derived runtime requirements from enabled staged mod metadata, so DMM can report that profile files are deployed but the game runtime/loader needed to use them is missing.
- Backend logs include archive extraction decisions and accepted install-plan summaries with format, entry count, installer flags, mod type, and instruction count.
- Game Plugins view shows active install/deploy/purge activity in context.
- Game Plugins view now leads with a mod-management dashboard: add-from-Nexus, deployment readiness, enabled/blocked/conflict counts, installed profile mods, and advanced file preview separated from primary deploy actions.
- Game Plugins view has been shifted toward the MVP profile-first model: selected profile summary, simple profile mod enable/disable toggles, profile apply state, and advanced deployment/file controls behind disclosure sections.
- Game Plugins now puts selected profile status before import controls, treats Nexus import as a secondary panel inside the selected game, and uses user-facing mod status labels in the profile mod list.
- Game Plugins disables repair/purge until an active deployment manifest exists.
- Completed install jobs now refresh deployment preview state automatically, and the Plugins view only reports a profile as applied when the preview proves there are no pending changes or conflicts.
- Endpoint-level coverage proves a user-facing profile mod toggle removes DMM-managed files on apply and recreates them when re-enabled and applied again.
- Action Center install cards explain what local install action will do after the archive is already captured/cached.
- Install settings now separate automatic install from automatic enable/deploy. Auto-install defaults on for MVP Deck-only flow, and newly installed mods remain disabled unless auto-enable is explicitly enabled.
- Auto-install and auto-enable settings belong in the Decky plugin Settings tab, not in the phone/tablet web app.
- Endpoint-level coverage proves a captured Nexus request immediately downloads/caches the archive, then either waits for local install approval or auto-installs from cached state.
- Endpoint-level coverage proves auto-enable can install, enable, deploy DMM-owned symlinks, record the deployment manifest, and clear the captured-install action.
- Decky plugin receives backend domain events over WebSocket and emits Gaming Mode toast notifications for capture/download/install transitions while the plugin is loaded.
- Deploy, purge, and staged-mod removal now require an explicit in-app confirmation that summarizes the game-folder or staging impact before the API call runs.
- Manual and automatic deployment publish throttled per-file job progress while applying profile changes, so large deploys do not appear frozen in the phone/tablet UI or Decky job notifications.
- The default web landing surface now prioritizes Action Center items and gives a direct path to choose a game when no user action is waiting.
- Endpoint-level coverage proves the Stardew recovery path can stage an orphaned Nexus archive, preview deployment, deploy symlinks, record the manifest, purge manifest-owned links, and clear the active deployment manifest.
- Test packaging scripts for fast SSH deployment to the Steam Deck.
- Host-side Deck installer accepts explicit SSH options for auth-agent recovery without editing the script.
- Manual Deck transfer bundle exists for non-SSH package/script staging.
- Deck package smoke script exists for repeatable non-installing package validation before replacing the live Decky plugin.
- Deck package smoke script supports local shape-only validation for non-Linux development hosts.
- `make mvp-audit` runs Go tests, web/Decky builds, Python syntax, testing script syntax, local backend smoke, packaging, package shape validation, and whitespace checks.
- `make deck-transfer` rebuilds the package and creates the non-SSH transfer bundle.
- `make mvp-release` runs the full repo audit and then creates the transfer bundle from the audited package.
- Deck-side live acceptance script checks the post-deploy MVP state through backend diagnostics: health, staged/enabled mods, active work, deployment manifest, deployed file count, preview availability, and preview conflicts.
- Deck-side live profile toggle script verifies the user-facing enable/disable workflow removes DMM-managed files on profile apply, restores them after re-enable/apply, and leaves symlinks pointing into DMM storage.
- Deck-side live auto-install script can temporarily enable automatic install for captured downloads, wait for a fresh Nexus capture, and fail if the request stops at manual install approval.
- Deck-side Stardew file visibility script verifies enabled profile mods are represented by DMM-managed symlinked SMAPI manifests under the live game `Mods/` folder before manual game launch.
- The packaged Linux build has been smoke-tested on the Deck from `/home/deck/.testing/decky-mod-manager.tar.gz`.
- The latest package copied to `/home/deck/.testing/` includes deployment status UI/API updates, install-request approval copy, and game diagnostics output, and passed a non-installing package smoke test on the Deck.
- Latest local package passed local shape validation.
- Manual transfer bundle was created through `make mvp-release` at `dist/deck-transfer/` and `dist/decky-mod-manager-deck-transfer.tar.gz`; local checksums and package-shape validation passed.
- A scripted copied-data Deck rehearsal against copied DMM data and copied Stardew files can assert expected recovered source mod IDs and planner IDs, preview zero conflicts, deploy DMM-owned symlinks, and purge those symlinks without touching the real game folder.
- The latest copied-data Deck rehearsal recovered and verified SMAPI (`2400`, planner `vortex:stardewvalley:smapi-installer`, 46 files) and Visible Fish (`8897`, planner `vortex:stardewvalley:stardew-valley-installer`, 7 files), previewed zero conflicts, deployed 96 files in the copied game, and purged cleanly.
- The strict copied-data Deck rehearsal also verified DMM-managed SMAPI root links for `StardewModdingAPI`, `StardewModdingAPI.dll`, `StardewModdingAPI.deps.json`, and `smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll`, proving the metadata-driven SMAPI planner produces runtime-facing files through the deployment manifest.
- Live Deck validation captured a fresh Nexus `nxm://` request from Gaming Mode, showed the Decky download toast, installed from cached state through the phone/tablet UI, added the mod to the selected profile, deployed profile changes from the UI, and passed the Deck-side MVP live acceptance script with 4 staged/enabled mods, 44 deployed symlinked files, 0 blocked candidates, 0 active jobs, and 0 preview conflicts.
- Live Stardew file visibility validation passed with 4 enabled profile mods represented by 4 DMM-managed SMAPI `manifest.json` symlinks in the live game `Mods/` folder.
- The newest transfer bundle was copied to `/home/deck/.testing/`, checksum-verified on the Deck, and passed the non-installing Deck package smoke test.
- A Deck-side installed-package check exists to compare `/home/deck/homebrew/plugins/decky-mod-manager` against the staged package before diagnosing live UI/API behavior.
- Both package install paths now run the installed-package verifier, so an install cannot report success while leaving stale backend or web UI files in the live Decky plugin directory.

### Remaining

1. Archive extraction
   - Status: complete for the MVP vertical slice.
   - Completed: ZIP safety checks; 7z/RAR helper execution; helper presence surfaced in dependencies; explicit FOMOD rejection.
   - Completed: helper-tool errors name the missing/failing helper and direct users to the Decky plugin Dependencies view.

2. Generic staging install
   - Status: mostly complete for MVP, with broad Vortex metadata ingestion as the P1 deployment-pipeline workstream.
   - Completed: archive extraction into a temporary install workspace; manifest checksums; install planner returns explicit deployable files from Vortex-modeled installer specs.
   - Completed: the first Vortex-modeled Stardew metadata spec separates installer matching from mod type/deploy root behavior, covers manifest-based SMAPI mods, recognizes root-folder `Content/` archives, and handles SMAPI installer archives by extracting the Linux embedded payload and staging game-root files through explicit target mappings.
   - Completed: import URL resolution is catalog-dispatched, and non-Nexus catalogs are rejected clearly before download because no non-Nexus download provider exists yet.
   - Completed: automatic SMAPI payload installer mode from Vortex metadata extracts the Linux `install.dat` payload, stages SMAPI game-root files, and materializes the generated deps file from the game install folder when available.
   - Completed: target mapping policy is persisted through staged manifests and deployment planning, allowing metadata-driven skips for files that should not overwrite existing game-owned targets.
   - Completed: relaxed SMAPI manifest parsing handles BOM/trailing-comma manifests that Vortex accepts, preventing valid Nexus automatic downloads from being blocked by strict JSON parsing.
   - Completed: SMAPI manifest attribute extraction persists mod identity, version, entry DLL, content-pack targets, and dependency metadata into staged manifests for diagnostics.
   - Completed: diagnostics use staged manifest dependency metadata to warn when a required Stardew framework/dependency mod is not enabled in the selected profile.
   - Remaining: expand the metadata evaluator so installer specs can express Vortex rules/dependencies beyond SMAPI manifests, attributes from other game extensions, installer-choice pauses, and broader extension-derived game specs beyond the reviewed Stardew slice.
   - Remaining: add retry/continue actions for blocked install candidates once a supported installer/dependency handler exists.
   - MVP requirement: archives that cannot produce an install plan must remain downloaded/cached but must not be treated as enabled deployable mods.
   - Decision needed after MVP: whether Vortex metadata is imported through a declarative schema, translated from extension source/package data, compiled as reviewed Go specs, or supported through a hybrid.

3. First supported game handler
   - Status: incomplete for the expanded MVP vertical slice.
   - Completed: Stardew Valley maps simple staged SMAPI mod folders into `Mods/`; unsupported game domains fail clearly; unsupported install plans appear as blocked install candidates in UI/jobs.
   - Completed: live Gaming Mode validation against a fresh simple Stardew mod after installing the then-current package.
   - Completed: native Linux Stardew support selects the Linux SMAPI payload through the Stardew extension metadata.
   - Completed: SMAPI primary launch-tool requirements are expressed through extension metadata/rules and evaluated generically from enabled profile mod metadata.
   - Completed: backend launch-action endpoints publish and verify the desired Steam launch action without generic server code hardcoding Stardew-specific behavior or editing Steam config files.
   - Remaining: live-verify the newest installed package deploys SMAPI launch-root files as copies and launches Stardew through SMAPI in Gaming Mode.
   - Post-MVP: implement and verify Windows/Proton Stardew end to end on Steam Deck through the same extension framework.

4. FOMOD support
   - Status: broad MVP support implemented; live success validation still required with a valid browser-generated Nexus FOMOD capture.
   - Verified upstream behavior: Vortex delegates FOMOD parsing/planning to `@nexusmods/fomod-installer-native`, queues the shared installer dialog, stores final `installerChoices`, and can reuse saved choices for unattended installs.
   - Completed: FOMOD detection and safe blocking before staging.
   - Completed: FOMOD/installer-choice mods pause after download/archive inspection as persisted install candidates instead of failing as dead ends.
   - Completed: `fomod/ModuleConfig.xml` parsing for required files, install steps, groups, options, selected file/folder entries, conditional visibility, option dependency state, file dependency state, game dependency checks, priority/order semantics, nested `.fomod` archives, and exact-file saved choices.
   - Completed: choices are presented in the phone/tablet UI and the Decky modal, applied through the normal staging/install-plan path, and staged as disabled profile mods unless the user explicitly enables them.
   - Completed: extension-registered installer-choice capabilities own target roots, mod types, stop-folder normalization, and destination-prefix policy instead of using generic core FOMOD assumptions.
   - Remaining: real Nexus `nxm://` success validation with a valid FOMOD archive, richer image/condition-message presentation, and more live fixtures across Fallout 4/Skyrim/Witcher-style installers.
   - Direction: continue the native Go FOMOD evaluator for MVP, with extension-owned inputs and core-owned validation/deployment. Do not add a second FOMOD engine unless source verification proves a concrete parity gap that the Go evaluator cannot reasonably close.

5. Deployment preview
   - Status: complete for the MVP vertical slice.
   - Completed: Stardew preview supports add, keep, replace, remove, conflict, and skipped duplicate-target actions; deployment preview consumes stored staged install-plan target mappings and rejects invalid installed records without target mappings.
   - Completed: live Gaming Mode validation that a newly staged mod deploys from the stored target manifest after installing the then-current package.
   - Decision needed after MVP: per-game handler contract for target mapping, mod types, safe deploy roots, and unsupported layout reporting.

6. Deployment apply
   - Status: complete for the MVP vertical slice.
- Completed: deploy, purge, repair, and captured-install actions are represented as jobs.
   - Completed: manual and automatic deploys publish throttled per-file progress messages while profile changes are applied.

7. Rollback and repair
   - Status: explicit user-facing rollback is MVP.
   - Completed: apply-time backup restoration, manifest-based purge, repair of missing/broken DMM-owned links, and game-level reset of DMM-managed mods.
   - Completed locally: deployment planning can reconcile an active manifest with zero installed/enabled profile mods by producing remove actions for DMM-owned deployed files instead of returning `no installed profile mods are available to apply`.
   - Completed: deleting an installed profile mod now applies the selected profile through the backend so DMM-owned deployed files are reconciled before the user sees the mod list update.
   - Remaining: rollback jobs, rollback manifests/history, restore-point retention, rollback preview, and UI surfaces.
   - MVP UX model: simple users see a single safe `Restore last applied state` action only when a rollback is available; power users can open advanced rollback history, named restore points, preview, retention settings, and job/file details.
   - Scope boundary: rollback restores DMM-owned deployed artifacts from DMM manifests. It must not claim to restore unmanaged/manual files or external Vortex artifacts.

8. Mobile/tablet UI completion
   - Status: mostly complete for MVP.
- Completed: home captured-install actions, action cancellation, game drawer, settings drawer, game modules, in-game plugin actions, profile-scoped mod priority controls, contextual game activity, global Jobs as history.
- Completed: failed retryable captured-install actions show a Retry action in the Action Center.
   - Completed: live-verified profile-first Plugins deployment path from phone/tablet UI after approving a fresh Nexus request.
   - Completed: profile-first Plugins polish pass places selected profile state before import controls and keeps advanced deployment tools secondary.
   - Completed: endpoint-level test coverage verifies immediate download followed by cached install approval or auto-install.
   - Completed: Deck-side live auto-install verifier exists for the next real Nexus capture.
   - Remaining: live-verify the auto-install setting with a fresh Nexus capture after installing the latest package.
   - Completed: live-verified Decky notification for a captured Nexus capture/download transition in Gaming Mode.
   - Remaining: user-required morning validation for final Gaming Mode toast coverage after event architecture lands.
   - Remaining: final real-device review after the next package install.

9. Logging and diagnostics
   - Status: complete for the MVP vertical slice.
   - Completed: structured backend logs for capture, resolve, download, extract, stage, deploy, purge, repair; Decky Debug diagnostic tails; documented log paths.
   - Completed: routine health/job polling is no longer logged at info level, so recent log tails preserve install/deploy failures and mutating actions.
   - Completed: archive extraction decisions and accepted install-plan summaries are logged before staging.

10. Stardew SMAPI launch integration
   - Status: complete for native Linux MVP; launch-option mutation uses Decky's Steam API, and live launch verification proved Stardew launches through SMAPI with deployed mods.
   - Verified upstream behavior: Vortex registers SMAPI as a supported/default primary tool for Stardew and selects the SMAPI executable name by platform.
   - Completed: diagnostics can report that enabled Stardew SMAPI mods require SMAPI files and a SMAPI launch path.
   - Completed: extension metadata marks SMAPI as the default/primary launch tool when enabled mod metadata requires SMAPI.
   - Completed: backend runtime-action endpoints declare the desired launch-tool change for Decky without editing Steam config files.
   - Completed: Decky frontend syncs required launch actions and uses Steam's frontend API when available.
   - Completed: SMAPI launch-root files can be extension-marked as copy deployments, preserving VFS symlinks for normal mod files while avoiding .NET host resolution through staging.
   - Completed: live-verified Stardew launches through SMAPI in Gaming Mode after DMM deployment.

11. End-to-end Steam Deck validation
   - Status: current package is installed and package-shape verified; remaining work is live MVP behavior validation.
   - The current package was installed on the Deck through the temporary privileged test wrapper, passed installed-package verification, rebooted the Steam Deck, and SSH returned after restart.
   - Scripted copied-data rehearsal passed on the Deck without touching the real Stardew folder, including explicit expected-planner assertions for the cached SMAPI and Visible Fish Nexus downloads.
   - Deck-side live acceptance script passed against the running backend after a fresh Nexus request, approval, staging, and UI-driven deployment.
   - Deck-side Stardew file visibility script exists to prove the active deployment produces DMM-managed SMAPI manifest symlinks in the live game `Mods/` folder.
   - Deck-side Stardew file visibility script passed against the current live deployment with 4 DMM-managed SMAPI manifest symlinks visible under the live game `Mods/` folder.
   - Completed validation: install the then-current package, start backend from Gaming Mode, capture a fresh Nexus automatic download, approve from phone/tablet, download, inspect, stage, deploy, and verify the DMM-owned deployment manifest and symlink count.
   - Remaining validation: rerun MVP live, Stardew file visibility/runtime, auto-install, and real-device UI checks against the current installed package.
   - Remaining validation: launch Stardew Valley and confirm the deployed mod behavior is visible in game, then purge and redeploy from the live game folder once the in-game check has passed.

12. Game list favorites and sorting
   - Status: complete for MVP.
   - Completed: phone/tablet web game drawer persists local favorites and pins them to the top of the list.
   - Completed: phone/tablet web game drawer supports `Recent`, `A-Z`, and `Z-A` sorting.
   - Completed: `Recent` uses DMM-local game selection history for MVP.

## MVP Acceptance Criteria

- A Steam Deck user can install the Decky plugin from a package.
- The Decky plugin can start and stop the Go backend.
- The Decky plugin shows a reachable phone/tablet URL.
- The phone/tablet UI loads cleanly on mobile and iPad-sized screens.
- Nexus API key configuration works.
- Clicking a Nexus "Mod Manager Download" link in the Deck browser can create a captured install.
- Clicking a Nexus "Mod Manager Download" link in Gaming Mode shows a Decky notification when DMM captures the request or starts processing it.
- Pasting an `nxm://` link into Decky or the web UI can create a captured install.
- Captured Nexus requests download/cache immediately while the link is fresh.
- If auto-install is disabled, the user can approve a cached local install from the phone/tablet UI after download.
- Auto-install captured downloads and auto-enable installed mods can be toggled from Decky Settings and are not configurable from the phone/tablet web UI.
- The game list supports favorite games pinned at the top and user-selectable sorting by `A-Z`, `Z-A`, and `Recent`.
- Auto-install/auto-enable does not bypass FOMOD or installer-choice prompts unless a valid saved preset/headless choice set exists.
- Failed retryable captured installs can be retried without recapturing the Nexus link while DMM still has valid cached metadata.
- The archive downloads to DMM-managed storage.
- The archive is inspected before extraction.
- The mod is installed under DMM-managed staging as an implementation detail.
- The installed mod appears in the selected profile's mod list.
- The mod can be enabled or disabled for the selected profile with a simple toggle.
- Ordinary profile/mod management does not require the user to understand staging, manifests, or file operations.
- Stardew native Linux installs can install/deploy SMAPI and configure Steam to launch through `StardewModdingAPI`.
- Windows/Proton Stardew support is documented as post-MVP and not required for MVP sign-off.
- Runtime platform detection prevents unsupported platform/payload combinations from being silently deployed.
- A power-user deployment preview can show what will be written to the game folder.
- A power-user profile switch preview can show what DMM-owned artifacts will be removed, kept, or added.
- A normal user can restore the last applied DMM deployment state without understanding deployment manifests.
- A power user can inspect rollback history, restore points, rollback preview, retention, and rollback job details.
- Deployment applies using a safe strategy and records a manifest.
- Purge removes deployed files from the manifest.
- Removing or resetting mods cannot leave stale DMM deployment manifests, broken DMM-owned symlinks, or active deployment files pointing at deleted staging folders.
- Purge leaves unmanaged files and parent game directories intact.
- Dirty/external game state blocks unsafe deployment by default.
- Logs are sufficient to diagnose a failed install without asking the user to transcribe UI errors.

## Right After MVP

These items are targeted for the first release after the MVP vertical slice. They are intentionally not being implemented until the listed decisions are made, because each one affects storage shape, extension boundaries, security posture, or long-term UI expectations.

Completion rule for this roadmap section: an item is considered ready for post-MVP implementation only when its owner-facing decision is answered. Until then, it must stay marked incomplete and must not be hidden behind a workaround.

1. Additional game handlers
   - Status: in progress under the first-party compiled Go extension model.
   - Decision: game-specific behavior belongs in first-party Go extension modules using the DMM extension SDK. JSON/YAML/Lua/Starlark are not the behavior source of truth for MVP.
   - Why the decision matters: target mapping, mod-type detection, dependency checks, lifecycle hooks, launch tools, and game-specific tools must be expressed through extension APIs and validated by the generic core.
   - Implementation status: many installed-game extensions are registered, but each game still needs source-backed archive validation before it can be marked release-ready.

2. Handler-derived runtime requirements
   - Status: partially complete.
   - Completed: new staged manifests persist install-plan metadata such as `mod_type`, and the current compiled handler path emits runtime requirements into game diagnostics and the mobile Review tab from enabled mod metadata. Stardew SMAPI deployment now reports whether SMAPI is present or referenced by Steam launch options before the user expects deployed profile files to have in-game effect.
   - Decision: runtime requirements such as mod loaders, script extenders, launch options, and game tools are expressed by compiled game extensions through reusable SDK capabilities.
   - Why the decision matters: DMM must not add one-off game checks. Requirements must be generated from the same extension/provider knowledge that determines install and deploy behavior.
   - Live finding: DMM-managed Stardew mod files can be deployed correctly while the game still shows no effect if the relevant runtime/launcher path is missing. This is now surfaced from enabled staged mod metadata through the handler-derived requirement path, not hardcoded as individual mod business logic in generic UI/server code.
   - Implementation status: broader requirement contract remains blocked until the extension/metadata model is selected.

3. Explicit rollback jobs
   - Status: incomplete.
   - Decision needed: last-deployment rollback, named restore points, or profile-transition history.
   - Why the decision matters: this changes deployment manifest retention and the UI model for what a user expects "undo" to mean after several profile changes.
   - Implementation status: blocked until the user-facing rollback model is selected.

4. Dependency install helpers
   - Status: incomplete.
   - Decision needed: whether Decky may run privileged installer actions, or whether DMM only detects and links to manual instructions.
   - Why the decision matters: installing helpers such as `7z`, `unar`, SMAPI, or script extenders may require privilege escalation and has a different trust model than read-only detection.
   - Implementation status: blocked until the privilege/security model is selected.

5. Windows/Proton Stardew support
   - Status: incomplete.
   - Decision needed: how DMM should detect forced Proton state, select Windows SMAPI payloads, and configure launch options for Proton without editing Steam-owned config files directly when a Decky/Steam API path exists.
   - Why the decision matters: platform selection affects installer payloads, launch tool paths, deployment validation, and rollback semantics.
   - Implementation status: post-MVP, extension-driven, and blocked until native Linux Stardew is validated end to end.

6. Auth and pairing
   - Status: incomplete.
   - Decision needed: simple local pairing code, API token, reverse-proxy expectation, or HTTPS-first model.
   - Why the decision matters: this controls how much the LAN-only default can safely evolve for Tailscale, browser bookmarks, shared networks, and multiple phones/tablets.
   - Implementation status: blocked until the access-control model is selected.

7. Vortex/manual adoption wizard
   - Status: incomplete.
   - Decision needed: read-only import/report first, DMM-managed adoption, or cleanup flow that can remove existing Vortex artifacts.
   - Why the decision matters: adoption can destroy or orphan user files if DMM mistakes unmanaged files for deployable artifacts.
   - Implementation status: blocked until the adoption/cleanup risk model is selected.

8. Mod update checks
   - Status: incomplete.
   - Decision needed: Nexus-only update checks first, or generic provider update contract before adding update UI.
   - Why the decision matters: update records can either be tied to Nexus source metadata now or shaped around future upstream providers from the start.
   - Implementation status: blocked until the provider update-contract decision is selected.

## Down The Road

These are important for the release after MVP, but should not block the first vertical slice.

- Systemd user service mode after auth/pairing exists.
- Auth, pairing, HTTPS, or token-based access for non-LAN use.
- Tailscale/non-LAN setup guidance beyond the current LAN-only toggle.
- Full Vortex environment cleanup/import wizard.
- Existing manual mod adoption.
- Multiple mod upstreams such as ModDB, Thunderstore, GitHub Releases, direct URL, and local archive import.
- Nexus browsing/search inside DMM.
- Mod update checks and update workflows.
- Full dependency installation helpers.
- Game-specific modules for non-generic installers.
- Prototype/TexMod-style installers.
- Script extender detection and install helpers.
- Load-order tools and game-specific plugin sorting.
- Advanced conflict UI with per-file winner selection.
- Multiple deployment strategies per game/profile with migration between strategies.
- Moveable staging/download locations.
- Disk-space budgeting and cleanup policy UI.
- Export/import mod lists.
- Non-Steam game support.
- Desktop app or richer Deck-native client.
- In-Deck lightweight install approval flow that does not require a phone.
- PWA/offline metadata view.
- Playwright UI regression tests.
- Self-update through GitHub/Decky release flow.
- Production release signing/checksums.
