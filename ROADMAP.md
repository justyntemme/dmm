# Decky Mod Manager Roadmap

This roadmap tracks the remaining work for the first usable MVP and separates it from the work that should follow immediately after MVP.

## MVP Goal

Install a Nexus Mods "Mod Manager Download" / `nxm://` mod from Gaming Mode on Steam Deck, approve it from a phone or tablet when approval is required, add it to the selected game/profile, let the user enable/disable it without understanding staging internals, and provide safe deployment, logging, and rollback behavior underneath that simple workflow.

## MVP Status

### Current Decisions

- First vertical-slice game: Stardew Valley (`413150`, Nexus domain `stardewvalley`).
- Highest implementation priority is Stardew extension-framework parity with the required behavior from Vortex's Stardew Valley extension, including SMAPI installer handling, SMAPI mod install rules, runtime requirements, primary launch tool behavior, and Steam Deck launch configuration.
- The next required MVP feature after Stardew extension parity is FOMOD/installer-choice support with Decky modal support for Deck-only flows.
- Stardew Valley MVP support includes both native Linux and Windows/Proton Steam installs on Steam Deck. Native Linux is the immediate implementation target; Windows/Proton is required before MVP sign-off but should not block validating the native Linux launch path first.
- The default user mental model is profile-first: select a game, select a profile, download/approve a mod, then enable or disable that mod in the profile.
- The UI must not make ordinary users reason about staging, install planning, target mappings, manifests, or deployment transactions before they can use a mod.
- Staging, preview, deploy, purge, repair, blocked install plans, and file-level operations are still core safety mechanisms, but they should be exposed as advanced/power-user views unless immediate user action is required.
- Download approval should be controlled by a setting named around user intent, such as "approve downloads by default" or "require download approval"; default behavior is approval required.
- Auto-deploy should exist as a user setting for fast Deck-only use, but it must remain off by default until staging and deployment are reliable.
- ZIP, 7z, and RAR archives are all MVP-relevant because Nexus mods commonly use all three.
- The internal safe path remains: capture request, approve download when required, inspect the archive, produce an explicit install plan, stage only deployable outputs, update the selected profile's mod list, preview/apply deployment through DMM-owned manifests, and report failures clearly.
- The primary happy path should feel like: capture/download mod, approve if prompted, see it in the selected profile, toggle it on/off, and let DMM apply the safe deployment work in the background or through a simple "Apply profile changes" action.
- Gaming Mode must show immediate Decky feedback after Nexus hands DMM a download link. A user should not see only the Nexus "download starting" page and then silence; the Decky plugin should notify when a request is captured, waiting for approval, processing, completed, or failed.
- DMM must follow a Vortex-style virtualized deployment model: archives and extracted mods live in DMM storage, profiles define enabled staged mods, and game folders receive only DMM-owned deployment artifacts tracked by manifests.
- Profile switching must reconcile manifests and links, not mutate staged mods or overwrite unmanaged game files.
- Like Vortex, DMM must separate download, install planning, staging, and deployment. A downloaded Nexus archive is not automatically a deployable mod until a game/provider installer plan has identified its mod type, deployable files, and target mapping.
- The Go backend publishes required launch-tool actions, while the Decky frontend executes Steam frontend-only capabilities such as setting Steam launch options. This is an explicit service contract, not an ad hoc callback path.

### Done

- Decky plugin package that bundles the Go backend, Svelte web UI, and NXM handler.
- Decky plugin starts and stops the backend process directly.
- Decky plugin shows backend status, phone/tablet URL, Nexus link, dependency status, pasted URL import, and NXM handler registration tools.
- Backend serves REST APIs and the mobile/tablet web UI.
- LAN-only access enforcement is enabled by default.
- LAN-only middleware coverage proves public remotes are rejected while LAN/loopback requests pass, and the trusted-tunnel toggle disables that rejection.
- Sparse or older config files preserve secure defaults such as LAN-only access, the default listen address, default data directory, and auto-deploy disabled.
- Steam library and installed-game discovery.
- SQLite storage for discovered games and profiles.
- Storage migration handles older MVP database shapes by adding missing columns introduced during development.
- Default profile creation and profile switching APIs.
- Nexus API key configuration and validation path.
- Nexus HTTPS URL parsing.
- Nexus `nxm://` parsing for real Nexus path-style links.
- Import URL parsing now goes through registered catalog resolvers instead of directly calling Nexus parsing from the server; Nexus remains the only download-link provider in the MVP.
- Nexus game domains and mod/file IDs are normalized and validated before they are used in DMM download or staging paths.
- Firefox Flatpak / portal stale-handler issue documented through logs and fixed manually during testing.
- User-level `nxm://` handler registration.
- NXM handler logs redacted URLs and forwards install requests to the backend.
- Pending install requests appear in the phone/tablet UI.
- Pending install requests can be approved from the phone/tablet UI.
- Approval is guarded server-side so stale pending metadata cannot move a failed, completed, canceled, or already-running install job back to running.
- Pending install requests and active pending-import downloads/extractions can be canceled from the phone/tablet UI.
- Failed pending-import downloads and retryable staging failures keep their pending request metadata and can be retried from the phone/tablet UI; unsupported installer/layout failures still move to blocked install candidates instead of retrying blindly.
- Clearing visible install requests preserves completed install history in Jobs while removing waiting, running, queued, failed, and persisted pending request state.
- Pending install requests, resolved links, and jobs are persisted across backend restarts.
- Jobs now persist structured source/game metadata, and pending-import storage backfills that payload when requests are restored or attached internally, so the phone/tablet UI can scope pending imports and game operations by app ID or Nexus game domain instead of parsing user-facing job text.
- Existing MVP-era SQLite databases are upgraded in place for persisted jobs, pending imports, install manifests, profile priority, source metadata, and checksums.
- Interrupted pending-import downloads restore as waiting install requests after backend restart and persist that corrected state, so the user can approve the download again instead of getting stuck with a stale running job.
- Endpoint-level coverage proves approving an install request can download a Nexus archive over HTTP, stage it, record it as an enabled Stardew mod, and clear the pending request.
- Approval coverage uses an extensionless Nexus-CDN-style download URL, proving archive signature detection in the normal install path and not only the recovery path.
- Approved requests download archives into DMM-managed storage.
- Approved downloads are recorded with archive path and source metadata.
- Downloaded archives, staged files, and deployment manifests record SHA-256 checksums where files are locally available.
- Approved Stardew Valley requests can be inspected, extracted into staging, persisted as staged plugins, and shown in the Plugins tab.
- Install planning now separates extraction from deployable staging: archives are extracted to a temporary workspace, the game install planner emits explicit deployable file instructions, and only planned outputs are copied into DMM staging.
- Install planning is now driven by Vortex-modeled game metadata specs rather than one-off Nexus file IDs, filenames, or procedural per-mod rules.
- Installer metadata extractors are now declared by the game spec, so manifest validation and attribute ingestion are not hardcoded into the planner core. The generic JSON manifest extractor can ingest common fields such as display name, unique ID, version, entry DLL, and minimum API/runtime version for future Vortex-style game specs without new parser code.
- Vortex game ID to Steam app ID lookup now comes from the same install-plan registry instead of server-side game/domain switches, keeping import routing aligned with supported game specs.
- Deployment eligibility now comes from supported game specs as well. The server asks the registry whether a Steam app ID is deployable and whether review-state deployment is allowed, instead of carrying game-specific deployment branches.
- Download recovery is routed through the same game/domain registry, so orphaned Nexus archives are only restaged for games with a registered Vortex metadata spec and supported Nexus domain.
- Mod type deploy roots are now modeled separately from installer selection. Installers choose a mod type and emit install instructions; the selected mod type supplies the deploy root, matching Vortex's installer/mod-type split and avoiding per-installer path duplication.
- The current Stardew spec preserves Vortex's installer/mod-type separation for the MVP slice: `stardew-valley-installer` maps valid manifest-based mods into `Mods/`, `sdvrootfolder` maps root-folder archives to game-root targets, and `smapi-installer` extracts the Linux embedded payload into game-root targets.
- Verified SMAPI archive payloads contain separate Linux and Windows inner payloads. MVP must select the correct payload from detected game runtime: Linux payload for native Stardew, Windows payload for Proton Stardew.
- Install-plan metadata can mark individual target mappings with policy such as `keep-existing`, which lets SMAPI preserve an existing game-owned `steam_appid.txt` without treating the whole deployment as conflicted.
- Install-plan metadata controls staged mod display naming, so installer/runtime packages such as SMAPI can use the Nexus archive name while normal manifest-based mods use their manifest display name without adding server-side mod-type special cases.
- SMAPI manifest parsing now follows Vortex's tolerant shape for common real-world archives by accepting UTF-8 BOMs and trailing commas before validating manifest identity. SMAPI manifest attributes are now extracted through a spec-declared metadata extractor, including logical file names, name, unique ID, version, entry DLL, minimum API version, content-pack target, and dependencies.
- New staged manifests preserve the install planner ID and detection evidence, so DMM can trace a staged mod back to the Vortex-style archive/layout metadata that classified it.
- Game diagnostics use a declarative runtime requirement registry keyed by game and mod type. For the Stardew slice, this reports missing SMAPI/runtime state and required manifest dependency mods from staged metadata without adding one-off Nexus mod/file rules to server/UI code.
- Deployment planning now consumes stored install-plan target mappings from staged manifests and skips legacy raw-staged records that lack explicit target mappings.
- Unsupported installer/layout results are persisted as blocked install candidates and exposed through the game Requests UI instead of being treated as deployable staged mods.
- Blocked install candidates can be cleared from the game Requests UI after the user has reviewed the reason.
- Staged mods can be removed from the Plugins UI; this deletes the installed-mod row and DMM-managed staging directory while preserving downloaded archives for recovery/reinstall.
- Download recovery restages legacy installed records that lack install-plan target mappings instead of skipping them as already installed; unrecoverable legacy rows remain visible as `needs_recovery` and can be removed.
- Failed archive extraction removes partial staging output before the failed job is surfaced, so broken installs do not leave deployable-looking debris.
- Repeated downloads/restaging of the same mod keep download history but show one staged plugin row.
- Endpoint-level coverage proves duplicate approvals of the same Nexus mod/file still show one installed plugin row in the game UI.
- Deployment planning can recover staged file paths from the current data directory when an older absolute staging path is stale.
- Installed mod records are attached to the default/selected profile and can be enabled, disabled, or reprioritized from the Plugins tab.
- Profile mod enable/disable and priority updates validate that the selected profile and staged mod belong to the same game before writing profile state, preventing cross-game profile contamination.
- Installed mod records can be removed from the Plugins tab without deleting the cached downloaded archive.
- Stardew Valley staged plugins can generate a deployment preview targeting `Mods/`.
- Stardew Valley deployment preview is profile-aware and shows kept, added, replaced, removed, and conflicting deployment artifacts.
- Switching to a profile with no enabled mods can still generate remove actions for the current deployed profile instead of erroring.
- Endpoint-level coverage proves deploying an empty profile removes the current profile's DMM-owned links, clears active deployment files, and leaves staged mod records untouched.
- Known gap: deleting/removing all staged mods can still leave a stale active deployment manifest because deployment planning currently errors before reconciling manifest-owned files when no staged mod rows remain.
- Stardew Valley deployment can be explicitly applied from the preview when there are no conflicts.
- Manual and automatic deployment refuse no-op plans that contain only kept/skipped files, so users do not get misleading zero-change deploy jobs.
- Deployment manifests are persisted in SQLite.
- Active deployment status is exposed through the API and shown in the Plugins deployment card, so users can distinguish pending preview changes from files that are currently deployed.
- Existing DMM-owned links are manageable during profile transitions; unmanaged files remain conflicts.
- Duplicate target files are resolved deterministically by simple profile mod priority; priority is profile-scoped and editable in the Plugins tab, and losing mappings are shown as skipped preview actions.
- Purge removes only manifest-owned files and leaves unmanaged files and parent directories intact.
- Job list and SSE updates for request/download state.
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
- Game Plugins view now leads with a mod-management dashboard: add-from-Nexus, deployment readiness, enabled/blocked/conflict counts, staged mods, and advanced file preview separated from primary deploy actions.
- Game Plugins view has been shifted toward the MVP profile-first model: selected profile summary, simple profile mod enable/disable toggles, profile apply state, and advanced deployment/file controls behind disclosure sections.
- Game Plugins now puts selected profile status before import controls, treats Nexus import as a secondary panel inside the selected game, and uses user-facing mod status labels in the profile mod list.
- Game Plugins disables repair/purge until an active deployment manifest exists.
- Completed install jobs now refresh deployment preview state automatically, and the Plugins view only reports a profile as applied when the preview proves there are no pending changes or conflicts.
- Endpoint-level coverage proves a user-facing profile mod toggle removes DMM-managed files on apply and recreates them when re-enabled and applied again.
- Install request cards now explain what approval will do before the user starts the download/install-planning path.
- Install settings now separate automatic download approval from automatic deployment; download approval remains required by default.
- The auto-accept download setting belongs in the Decky plugin Settings tab, not in the phone/tablet web app.
- Endpoint-level coverage proves automatic download approval can resolve links, start the pending import without manual approval, download the archive, inspect/stage it, and add it to the selected profile.
- Endpoint-level coverage proves manual approval with automatic deployment enabled can download, stage, deploy DMM-owned symlinks, record the deployment manifest, and clear the pending import.
- Decky plugin polls backend install jobs and emits Gaming Mode toast notifications for request/download/install transitions while the plugin is loaded.
- Deploy, purge, and staged-mod removal now require an explicit in-app confirmation that summarizes the game-folder or staging impact before the API call runs.
- Manual and automatic deployment publish throttled per-file job progress while applying profile changes, so large deploys do not appear frozen in the phone/tablet UI or Decky job notifications.
- The default web landing surface now prioritizes pending install requests and gives a direct path to choose a game when no requests are waiting.
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
- Deck-side live auto-approval script can temporarily enable automatic download approval, wait for a fresh Nexus capture, and fail if the request stops at manual approval.
- Deck-side Stardew file visibility script verifies enabled profile mods are represented by DMM-managed symlinked SMAPI manifests under the live game `Mods/` folder before manual game launch.
- The packaged Linux build has been smoke-tested on the Deck from `/home/deck/.testing/decky-mod-manager.tar.gz`.
- The latest package copied to `/home/deck/.testing/` includes deployment status UI/API updates, install-request approval copy, and game diagnostics output, and passed a non-installing package smoke test on the Deck.
- Latest local package passed local shape validation.
- Manual transfer bundle was created through `make mvp-release` at `dist/deck-transfer/` and `dist/decky-mod-manager-deck-transfer.tar.gz`; local checksums and package-shape validation passed.
- A scripted copied-data Deck rehearsal against copied DMM data and copied Stardew files can assert expected recovered source mod IDs and planner IDs, preview zero conflicts, deploy DMM-owned symlinks, and purge those symlinks without touching the real game folder.
- The latest copied-data Deck rehearsal recovered and verified SMAPI (`2400`, planner `vortex:stardewvalley:smapi-installer`, 46 files) and Visible Fish (`8897`, planner `vortex:stardewvalley:stardew-valley-installer`, 7 files), previewed zero conflicts, deployed 96 files in the copied game, and purged cleanly.
- The strict copied-data Deck rehearsal also verified DMM-managed SMAPI root links for `StardewModdingAPI`, `StardewModdingAPI.dll`, `StardewModdingAPI.deps.json`, and `smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll`, proving the metadata-driven SMAPI planner produces runtime-facing files through the deployment manifest.
- Live Deck validation captured a fresh Nexus `nxm://` request from Gaming Mode, showed the Decky download toast, approved the request from the phone/tablet UI, staged the mod into the selected profile, deployed profile changes from the UI, and passed the Deck-side MVP live acceptance script with 4 staged/enabled mods, 44 deployed symlinked files, 0 blocked candidates, 0 active jobs, and 0 preview conflicts.
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
   - Remaining: detect Stardew runtime platform as native Linux or Windows/Proton from Steam/game state.
   - Remaining: select Linux or Windows SMAPI payload by detected runtime platform.
   - Remaining: express primary launch-tool requirements in extension metadata/rules, then configure and validate the platform-correct SMAPI launch tool without generic server code hardcoding Stardew-specific behavior.
   - Remaining: implement and verify Windows/Proton Stardew end to end on Steam Deck before MVP sign-off.

4. FOMOD support
   - Status: incomplete, MVP-required, and next after Stardew extension-framework parity.
   - Completed: FOMOD detection and explicit unsupported-installer failure.
   - MVP requirement: FOMOD/installer-choice mods must pause after download and archive inspection with a persisted installer-choice request instead of failing as a dead end.
   - Parse installer options needed for common Nexus/Vortex-compatible mods.
   - Present choices in the phone/tablet UI.
   - Present a Decky-native or Steam-overlay-friendly installer-choice surface for first-time no-phone installs, or explicitly block no-phone auto-deploy for first-time FOMOD mods.
   - Persist paused installer-choice jobs.
   - Store approved installer-choice presets so repeat installs/updates can run headlessly when the FOMOD structure still matches.
   - Apply selected choices into staging.
   - Decision needed before implementation: whether to implement a minimal FOMOD parser in Go first, bind/reuse Nexus' native FOMOD installer components, or treat FOMOD as a game-handler capability.

5. Deployment preview
   - Status: complete for the MVP vertical slice.
   - Completed: Stardew preview supports add, keep, replace, remove, conflict, and skipped duplicate-target actions; deployment preview consumes stored staged install-plan target mappings; legacy staged records without target mappings are skipped until recovered or removed.
   - Completed: live Gaming Mode validation that a newly staged mod deploys from the stored target manifest after installing the then-current package.
   - Decision needed after MVP: per-game handler contract for target mapping, mod types, safe deploy roots, and unsupported layout reporting.

6. Deployment apply
   - Status: complete for the MVP vertical slice.
   - Completed: deploy, purge, repair, and install requests are represented as jobs.
   - Completed: manual and automatic deploys publish throttled per-file progress messages while profile changes are applied.

7. Rollback and repair
   - Status: partially complete.
   - Completed: apply-time backup restoration, manifest-based purge, and repair of missing/broken DMM-owned links.
   - Remaining: deleting a staged mod that has active deployed files must either block with a clear "apply/purge first" message or run a safe remove-and-reconcile flow that removes only manifest-owned artifacts before deleting staging.
   - Remaining: add a game-level "Reset DMM-managed mods" action that purges the active deployment manifest, disables/removes profile mod associations, clears staged mod rows and staging folders as requested, and leaves cached downloads available for recovery unless the user explicitly chooses to delete them too.
   - Remaining: fix deployment planning so an active manifest with zero staged mods can still produce remove actions instead of returning `no staged mods are available to deploy`.
   - Remaining: explicit user-visible rollback jobs.
   - Decision needed: rollback UX model: last-deployment rollback only, named restore points, or profile-transition history.

8. Mobile/tablet UI completion
   - Status: mostly complete for MVP.
   - Completed: home install requests, request cancellation, game drawer, settings drawer, game modules, in-game plugin actions, profile-scoped mod priority controls, contextual game activity, global Jobs as history.
   - Completed: failed retryable install requests show a Retry action in both the global request list and game Requests tab.
   - Completed: live-verified profile-first Plugins deployment path from phone/tablet UI after approving a fresh Nexus request.
   - Completed: profile-first Plugins polish pass places selected profile state before import controls and keeps advanced deployment tools secondary.
   - Completed: endpoint-level test coverage verifies the automatic download approval pipeline through download and staging.
   - Completed: Deck-side live auto-approval verifier exists for the next real Nexus capture.
   - Remaining: live-verify the automatic download approval setting with a fresh Nexus capture after installing the latest package.
   - Completed: live-verified Decky notification for a captured Nexus request/download transition in Gaming Mode.
   - Remaining: final real-device review after the next package install.

9. Logging and diagnostics
   - Status: complete for the MVP vertical slice.
   - Completed: structured backend logs for capture, resolve, download, extract, stage, deploy, purge, repair; Decky Debug diagnostic tails; documented log paths.
   - Completed: routine health/job polling is no longer logged at info level, so recent log tails preserve install/deploy failures and mutating actions.
   - Completed: archive extraction decisions and accepted install-plan summaries are logged before staging.

10. Stardew SMAPI launch integration
   - Status: incomplete and MVP-blocking.
   - Verified upstream behavior: Vortex registers SMAPI as a supported/default primary tool for Stardew and selects the SMAPI executable name by platform.
   - Completed: diagnostics can report that enabled Stardew SMAPI mods require SMAPI files and a SMAPI launch path.
   - Remaining: extension rule that marks SMAPI as the default/primary launch tool when enabled mod metadata requires SMAPI.
   - Remaining: backend runtime-action endpoint that declares the desired launch-tool change instead of directly mutating Steam frontend state.
   - Remaining: Decky frontend action that reads current launch options and applies the desired launch option through Steam's frontend API.
   - Remaining: backend verification after Decky applies the action.
   - Remaining: backup/restore/drift-detection strategy if a direct `localconfig.vdf` fallback is ever needed.

11. End-to-end Steam Deck validation
   - Status: live acceptance passed for the previous installed package; the newest staged package still needs install/live verification.
   - The previous package was installed on the Deck and used from Gaming Mode.
   - The newest package is staged in `/home/deck/.testing/` and passes checksum/package-smoke validation, but `live_installed_package_check.sh` currently proves the live Decky plugin directory is stale.
   - Scripted copied-data rehearsal passed on the Deck without touching the real Stardew folder, including explicit expected-planner assertions for the cached SMAPI and Visible Fish Nexus downloads.
   - Deck-side live acceptance script passed against the running backend after a fresh Nexus request, approval, staging, and UI-driven deployment.
   - Deck-side Stardew file visibility script exists to prove the active deployment produces DMM-managed SMAPI manifest symlinks in the live game `Mods/` folder.
   - Deck-side Stardew file visibility script passed against the current live deployment with 4 DMM-managed SMAPI manifest symlinks visible under the live game `Mods/` folder.
   - Completed validation: install the then-current package, start backend from Gaming Mode, capture a fresh Nexus automatic download, approve from phone/tablet, download, inspect, stage, deploy, and verify the DMM-owned deployment manifest and symlink count.
   - Remaining validation: install the newest staged package and rerun installed-package, MVP live, Stardew file visibility/runtime, auto-approval, and real-device UI checks.
   - Remaining validation: launch Stardew Valley and confirm the deployed mod behavior is visible in game, then purge and redeploy from the live game folder once the in-game check has passed.

## MVP Acceptance Criteria

- A Steam Deck user can install the Decky plugin from a package.
- The Decky plugin can start and stop the Go backend.
- The Decky plugin shows a reachable phone/tablet URL.
- The phone/tablet UI loads cleanly on mobile and iPad-sized screens.
- Nexus API key configuration works.
- Clicking a Nexus "Mod Manager Download" link in the Deck browser can create an install request.
- Clicking a Nexus "Mod Manager Download" link in Gaming Mode shows a Decky notification when DMM captures the request or starts processing it.
- Pasting an `nxm://` link into Decky or the web UI can create an install request.
- By default, the user can approve an install request from the phone/tablet UI before the archive downloads/installs.
- If the user enables automatic download approval, captured requests can proceed without manual approval while still surfacing job state and errors.
- Auto-accept download requests can be toggled from Decky Settings and is not configurable from the phone/tablet web UI.
- Auto-deploy does not bypass FOMOD or installer-choice prompts unless a valid saved preset/headless choice set exists.
- Failed retryable install requests can be retried without recapturing the Nexus link while DMM still has valid request metadata.
- The archive downloads to DMM-managed storage.
- The archive is inspected before extraction.
- The mod is staged under DMM-managed staging as an implementation detail.
- The approved mod appears in the selected profile's mod list.
- The mod can be enabled or disabled for the selected profile with a simple toggle.
- Ordinary profile/mod management does not require the user to understand staging, manifests, or file operations.
- Stardew native Linux installs can install/deploy SMAPI and configure Steam to launch through `StardewModdingAPI`.
- Stardew Windows/Proton installs can install/deploy the Windows SMAPI payload and configure Steam to launch through `StardewModdingAPI.exe` before MVP sign-off.
- Runtime platform detection prevents Linux SMAPI payloads from being installed into Windows/Proton Stardew and prevents Windows SMAPI payloads from being installed into native Linux Stardew.
- A power-user deployment preview can show what will be written to the game folder.
- A power-user profile switch preview can show what DMM-owned artifacts will be removed, kept, or added.
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
   - Status: incomplete.
   - Decision needed: compiled Go handlers for the next release, scriptable handlers, or a hybrid model.
   - Why the decision matters: this sets the extension and security model for target mapping, mod-type detection, dependency checks, and game-specific tools.
   - Implementation status: blocked until the handler extension model is selected.

2. Handler-derived runtime requirements
   - Status: partially complete.
   - Completed: new staged manifests persist install-plan metadata such as `mod_type`, and the current compiled handler path emits runtime requirements into game diagnostics and the mobile Review tab from enabled mod metadata. Stardew SMAPI deployment now reports whether SMAPI is present or referenced by Steam launch options before the user expects deployed profile files to have in-game effect.
   - Decision needed: whether future runtime requirements such as mod loaders, script extenders, launch options, and game tools are expressed by compiled handlers, provider metadata, a manifest format, or a hybrid model.
   - Why the decision matters: DMM must not add one-off game checks. A mod manager needs requirements generated from the same handler/provider knowledge that determines install and deploy behavior.
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

5. Auth and pairing
   - Status: incomplete.
   - Decision needed: simple local pairing code, API token, reverse-proxy expectation, or HTTPS-first model.
   - Why the decision matters: this controls how much the LAN-only default can safely evolve for Tailscale, browser bookmarks, shared networks, and multiple phones/tablets.
   - Implementation status: blocked until the access-control model is selected.

6. Vortex/manual adoption wizard
   - Status: incomplete.
   - Decision needed: read-only import/report first, DMM-managed adoption, or cleanup flow that can remove existing Vortex artifacts.
   - Why the decision matters: adoption can destroy or orphan user files if DMM mistakes unmanaged files for deployable artifacts.
   - Implementation status: blocked until the adoption/cleanup risk model is selected.

7. Mod update checks
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
