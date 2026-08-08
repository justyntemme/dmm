# Decky Mod Manager Guidelines

These guidelines translate `notes.md` into build decisions. Treat `notes.md` as the raw Q&A record and this file as the working development contract.

## Steam Deck SSH Access

- Use the passwordless project test key directly; do not rely on the local SSH agent or 1Password when connecting as Codex.
- Steam Deck target: `deck@192.168.8.102`
- SSH key: `/Users/justyntemme/.ssh/decky_mod_manager_test`
- Public key: `/Users/justyntemme/.ssh/decky_mod_manager_test.pub`
- The normal `/Users/justyntemme/.ssh/id_rsa` key is passphrase-protected and may require 1Password. Do not use it for unattended DMM testing.
- Verify access with: `ssh -i ~/.ssh/decky_mod_manager_test -o IdentityAgent=none -o BatchMode=yes -o IdentitiesOnly=yes -o ConnectTimeout=5 deck@192.168.8.102 'printf ok'`
- Copy packages with: `scp -i ~/.ssh/decky_mod_manager_test -o IdentityAgent=none -o BatchMode=yes -o IdentitiesOnly=yes -o ConnectTimeout=5 dist/decky-mod-manager.tar.gz deck@192.168.8.102:/home/deck/.testing/decky-mod-manager.tar.gz`
- If access fails, first verify the Deck is online, then verify `/home/deck/.ssh/authorized_keys` still contains the public key from `/Users/justyntemme/.ssh/decky_mod_manager_test.pub`.
- If the public key must be reinstalled, use `ssh-copy-id -i ~/.ssh/decky_mod_manager_test.pub deck@192.168.8.102` from this machine, authenticating through the user's normal password or 1Password-backed key only for that repair step.

## Guideline 1: Verify Upstream Behavior First

- Do not assume behavior when it can be verified. Before copying, adapting, or claiming compatibility with a mod manager, plugin store, upstream catalog, game extension, installer format, or Steam/Deck API, inspect the source, official documentation, observed runtime state, or a real fixture.
- Before implementing or changing Vortex-compatible behavior, verify how Vortex or the relevant official game extension models the same behavior from source, documentation, or observed runtime state.
- Do not guess archive install rules, mod types, deployment roots, launcher behavior, dependency semantics, metadata extraction, or protocol handling when a Vortex implementation or official game-handler source exists.
- For Vortex plugin behavior, clone or inspect the relevant Vortex extension source before implementation. Do the same for each future plugin store/upstream: inspect its official client/API/schema/source instead of inferring from one downloaded archive.
- Use source verification as the default path for Nexus/Vortex compatibility work. Local observations from one downloaded mod are useful test fixtures, not architectural authority.
- When Vortex behavior is Windows-specific or does not map cleanly to Steam Deck/Linux, document the verified Vortex behavior first, then document the Linux-specific adaptation and why it is necessary.
- If a decision was previously inferred before source verification, mark it for review and either confirm it against upstream source or explicitly redesign it.
- Keep citations or source file references in implementation notes, tests, or documentation whenever we encode Vortex-derived behavior.

## Extension Boundary

- Every supported game must have an explicit DMM game extension, even when the install behavior is basic. The core app must not infer that a game, Nexus domain, archive layout, launch tool, load order file, Workshop behavior, runtime requirement, or deployment root is supported without an extension declaring it.
- Game-specific logic belongs in the extension API implementation for that game. Generic packages may provide reusable primitives such as archive-root planning, launch-tool setup, plugin activation generation, event hooks, target-root resolution, FOMOD evaluation, rollback, and deployment, but they must not contain game-specific branches.
- If a game needs behavior that is not covered by the current extension API, add a reusable extension capability first and then have the game extension declare that capability. Do not solve it by adding a one-off game branch to `server`, `storage`, `deploy`, `steam`, `installplan`, or UI code.
- First-party compiled Go extensions are the MVP extension packaging model. They are bundled in this repository for now, but their code is still the only allowed place for game-specific behavior. Runtime-loaded/community extensions are a later packaging and security boundary, not a license to bypass the extension API during MVP.
- Metadata-only and research-blocked extensions are acceptable for games with verified source/catalog signals but incomplete install support. They should explain the known source state without pretending that archive installation, launch tools, Workshop actions, or load order are supported.

## Decisions Requiring Source Review

- Stardew SMAPI launch integration: verify the Vortex extension's primary-tool and deployment behavior, then define the Steam Deck equivalent for launching through SMAPI.
- Stardew SMAPI installer extraction: verify the current Vortex installer source for Linux payload extraction, generated files, keep-existing target policies, and version/update handling.
- Stardew root-folder installer behavior: verify how Vortex distinguishes root `Content/` archives from normal SMAPI manifest mods, including nested wrapper-folder cases.
- Stardew manifest installer behavior: verify Vortex's manifest parser, locale filtering, nested manifest handling, metadata attributes, and how it picks installed display names.
- Runtime/dependency diagnostics: verify Vortex's SMAPI compatibility/dependency checks before expanding DMM's Review tab logic.
- FOMOD support: verify Vortex's installer-choice data model and persistence before implementing interactive installer UI.
- Existing Vortex/manual-mod detection: verify Vortex deployment manifest shape and cleanup semantics before any adoption or cleanup feature.
- Deployment method selection: verify Vortex deployment/hardlink/symlink behavior, then document Steam Deck filesystem differences before changing DMM's default deployment strategy.

## Architecture Decision Log

- Keep `decisions.md` as a concise architecture decision log for choices the user has not already made directly.
- Add a `decisions.md` entry when choosing between competing architecture patterns, storage models, extension boundaries, event delivery mechanisms, deployment semantics, privilege boundaries, or UI/system integration mechanisms.
- Do not record routine implementation details, small bug fixes, obvious refactors, or decisions already made explicitly with the user.
- Each entry should state the decision, options considered, rationale, tradeoffs, verification/source references when relevant, and follow-up work.
- If source verification or user review changes a decision, update or replace the entry instead of accumulating stale decision history.

## Pre-MVP Compatibility Policy

- Do not spend development time on backward compatibility with older DMM builds before the first real release boundary.
- Breaking internal API, config, database, package, or UI contracts is acceptable and encouraged when it produces simpler, faster, more reliable, or more maintainable code.
- Do not add compatibility shims for old backend/frontend status shapes, deprecated local config keys, stale package layouts, or abandoned in-flight implementation paths unless they are needed to recover the current developer test device.
- If a pre-MVP breaking change invalidates local test data, prefer a clear migration reset, diagnostic note, or one-time developer cleanup command over permanent compatibility code.
- Backward compatibility becomes a product requirement only after an explicit release/versioning policy exists.
- Use one active implementation paradigm for each product path. Do not keep old code paths, fallback behavior, or parallel update strategies alive after a newer architecture is selected; remove the old path and fix the selected path when it breaks.
- Temporary developer rescue tools are allowed only when explicitly scoped outside normal runtime behavior and documented as disposable.

## Commit Strategy

- Commit at coherent feature boundaries instead of batching unrelated work into large catch-all commits.
- Use Conventional Commits-style messages: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `build:`, or `chore:` followed by a concise imperative summary.
- Keep each commit focused on one logical change: backend behavior, Decky integration, web UI, docs, tests, or packaging. Split work when one change grows into multiple reviewable concerns.
- Run the relevant lightweight verification before committing. For cross-layer changes, prefer `go test ./...`, Decky build, web build, and package smoke checks when the touched files warrant it.
- Do not commit transient AI planning artifacts, scratch files, local logs, downloaded archives, device-specific secrets, or generated dependency folders.
- Commit durable project documentation when it is part of the requested product record, such as guidelines, roadmap, testing docs, architecture docs, and extension target inventories.
- If the worktree contains unrelated user changes, leave them out of the commit unless the user explicitly asks to include the full current tree.
- Mention known unverified areas or follow-up requirements in the final response after a commit, not by hiding them in the commit message.

## Product Direction

- Build a Steam Deck-first mod manager named Decky Mod Manager.
- MVP install path is a Decky Loader plugin that bundles and controls a Go backend.
- The phone/tablet browser UI is the primary MVP management surface.
- MVP supports Steam games on Steam Deck only.
- MVP focuses first on Nexus Mods "Mod Manager Download" / Vortex-compatible flows, and includes popular non-Steam mod upstreams as first-class catalog sources when their official API/client behavior can be verified.
- MVP provider architecture must support all popular non-Steam mod upstreams we can verify through official APIs, documented clients, schemas, or source review. Nexus remains the first critical source; direct archive URLs are supported as a generic selected-game import path; Steam Workshop is treated separately as Steam-owned platform content rather than as a browsed/searchable remote catalog.
- Persist source/catalog identity for every captured, cached, installed, enabled, disabled, updated, or removed mod. Every visible mod row should show a small source pill, and UI surfaces should be able to tag, filter, and sort mods by source.
- Steam Workshop content should coexist with DMM-managed mods where the game extension declares it safe. Steam Workshop unsubscribe, enable/disable, and load-order movement should use documented or directly verified Steam/Decky APIs; do not add filesystem or client-state hacks for Workshop management. MVP does not need a Steam Workshop search/browse UI, only installed Workshop content management.
- MVP workflow: capture a Nexus `nxm://` Mod Manager Download link from the controlled Decky browser flow, or paste a Nexus `https://www.nexusmods.com/...` URL / `nxm://` URL into Decky or the phone UI, resolve it, download/cache it immediately while the link is fresh, then install it into the selected profile through Action Center when manual install is required.
- Staging, install planning, deployment manifests, and file operations are implementation details for the default experience; they remain inspectable through advanced/power-user surfaces.
- User-level `nxm://` OS protocol registration is MVP for the Deck browser flow; pasted `nxm://` links remain a supported manual capture path.
- Future direction is to replace Vortex more broadly, but first milestones stay narrow and Steam Deck-focused.
- Existing manually installed mods and existing Vortex/NMM libraries are future import features, not MVP.
- Games with existing Vortex/manual mod state must be detected and treated as externally managed/dirty for MVP.
- Do not automatically clean, purge, adopt, or overwrite files from broken Vortex environments in MVP.
- Choose a clean or low-risk installed Steam game with Nexus/Vortex-compatible downloads for initial testing.
- The app is device-local. Multi-device control is out of scope, but export/import of mod lists should be considered after MVP.
- Per-game profiles/loadouts are MVP.

## MVP Game Scope

- Inspect the Steam Deck over SSH and build the first supported-games list from installed Steam libraries, including microSD paths.
- Prefer a fresh or low-risk game for the first vertical slice.
- Candidate test games from user notes:
  - The Witcher 3
  - Fallout 4
  - Final Fantasy VII Remake/Rebirth, depending on installed title
  - Project Zomboid, but likely Steam Workshop-focused and not ideal for Nexus MVP
- Nexus/Vortex-supported games are the MVP target. Games without Vortex/Nexus mod manager support can be detected but ignored initially.
- First custom handler should be chosen after Steam library inspection.
- Current MVP vertical-slice target: `Stardew Valley` (`413150`, Nexus domain `stardewvalley`).
- Stardew Valley MVP support targets the Steam Deck-native Linux install.
- Windows/Proton Stardew support is post-MVP, but must remain extension-driven when implemented.
- Stardew support must detect the installed runtime shape before installing runtime payloads:
  - Native Linux install: `StardewValley` launcher/script and Linux SMAPI payload with `StardewModdingAPI`.
  - Post-MVP Windows/Proton install: `Stardew Valley.exe`/Proton compatibility state and Windows SMAPI payload with `StardewModdingAPI.exe`.
- Windows/Proton support must be extension-driven. The generic backend must not assume that every Steam Deck install wants Linux payloads just because DMM runs on Linux.
- Avoid initial testing on Witcher 3, FF7 Rebirth, Fallout 4, Skyrim, Cyberpunk 2077, and Oblivion Remastered because Vortex/manual mod state was detected.

## UX Guidelines

- Phone and tablet UI are first-class MVP targets.
- UI is touch-first and dark-mode-only for MVP.
- Desktop browser layout is non-MVP, but the app should not be unusable on desktop-sized browsers.
- The default user mental model is profile-first:
  - Select game.
  - Use the selected/default profile.
  - Download or install a captured mod.
  - See the mod in that profile.
  - Enable/disable mods with simple toggles.
  - Apply profile changes when needed.
- The default user experience should not require understanding staging, installer target mappings, deployment manifests, symlink strategy, purge, or repair.
- First screen or primary navigation should expose:
  - Games list
  - Active profile/loadout
  - Mod list
  - Active downloads/jobs
  - Installed mods
  - Enabled mods
  - Disabled mods
  - Paste URL/import field
- Design for simple install-and-play flows while still keeping power-user controls available.
- Staging, deployment preview, purge, repair, conflict details, blocked install internals, and file-level controls should live behind an advanced view unless they need immediate user attention.
- The primary mod-management surface should be a profile mod list with clear enabled/disabled state, profile-scoped priority when relevant, pending/apply-needed state, and a simple apply action.
- Avoid exposing backend pipeline terms such as "staged", "manifest", "install plan", or "target mapping" in primary user flows. Use them only in advanced/debug views and logs.
- Nexus links can expire quickly, so DMM should download captured Nexus archives immediately after capture. Approval gates the local install step, not the network download.
- Decky Settings should expose "Auto-install captured downloads" and "Auto-enable installed mods"; auto-install defaults on, auto-enable defaults off, and both must be explicit and easy to disable.
- Auto-enable may install, enable, and deploy files when there are no conflicts. When auto-enable is off, newly installed mods must remain disabled until the user enables them.
- The phone/tablet web app should not expose these Deck behavior switches. They belong in the Steam Deck plugin because they change Deck-side capture/install behavior.
- Gaming Mode must show Decky notifications for Nexus request capture and install/download transitions, especially when the Nexus browser page only says that a download is starting.
- FOMOD installer choices should be presented as clear touch-friendly forms in the browser UI and, for Deck-only flows, in a Decky modal rather than a crowded sidebar view.
- If "Auto-enable installed mods" is enabled and a first-time FOMOD/installer-choice request is reached from a Deck-side flow, DMM should attempt to open a Decky-native choice modal automatically, even if the Decky sidebar tab is not already open.
- If Decky/Steam overlay APIs cannot safely open that modal from the background, DMM should show a Decky notification explaining that installer choices are required and tell the user to open Decky Mod Manager and click the installer entry to continue. The same installer-choice request must also remain visible in the phone/tablet UI.
- The "Auto-enable installed mods" setting should include helper text that FOMOD/installer-choice menus may appear as Decky modals before deployment can continue.
- Destructive actions must require confirmation.
- Downloads, installs, deployment, purge, and repair should appear in a visible activity/job queue.
- Background jobs must continue if the phone disconnects.
- UI should avoid unnecessary typing but must not hide important controls from advanced users.
- Prioritize fast first load and clear status on Steam Deck hardware, while keeping the interface polished enough for daily use.

## Decky Plugin Guidelines

- Bundle the Go backend with the Decky plugin for MVP.
- The Decky plugin should show:
  - Server running/stopped status
  - LAN IP/URL in plain text
  - QR code for the phone UI
  - Basic start/stop toggle
  - Debug/settings tabs
- Debug/settings should include:
  - Current IP/port
  - Backend health/version
  - Logs or a log tail
  - Storage paths
  - LAN-only setting
  - Optional "keep server running" behavior with a warning
- Decky plugin manages the backend process directly for MVP.
- User-level systemd service support is post-MVP and should wait until auth/pairing exists.
- The plugin release layout must be compatible with Decky plugin installation.
- Eventually support installation from a GitHub release ZIP.

## Decky Capability Boundary

- The Go backend owns domain state and decisions: game detection, extension metadata, install planning, staging, profile state, deployment manifests, runtime requirements, and the desired launch-tool state for a game/profile.
- The Decky frontend owns Decky/Steam-client capabilities that are only available inside Steam's frontend context.
- The Decky frontend may call Steam client APIs such as `SteamClient.Apps.RegisterForAppDetails` and `SteamClient.Apps.SetAppLaunchOptions` when a backend-published runtime action requires it.
- The backend must not depend on callbacks from the frontend to continue core install/deploy work. Frontend-executed capability actions are explicit user/admin actions with a request, result, and audit trail.
- Service contract shape:
  - Backend exposes required runtime actions, including `app_id`, current status, desired launch option, source extension, reason, and risk level.
  - Decky frontend reads the action, shows the user what will change, invokes the Steam client capability, then reports the observed result back to the backend.
  - Backend stores the result and re-runs diagnostics from Steam/game state instead of blindly trusting the UI response.
- Prefer a verified Steam client API for Steam launch options over editing Steam config files directly.
- Direct `localconfig.vdf` mutation is not a product/runtime path before release. If we ever need a developer rescue tool, it must live outside normal app flow and must not be wired as an automatic runtime path.
- If a community Decky plugin already exposes a stable integration point for a Steam capability, verify its source and decide whether to integrate rather than duplicating behavior.

## Network And Security Guidelines

- MVP binds to all interfaces by default, but must enforce "LAN only" filtering by default.
- LAN-only filtering must be controlled by a settings toggle and enabled by default.
- Users may disable LAN-only filtering for VPN/tunnel use cases such as Tailscale.
- Implement LAN-only filtering in one HTTP middleware using Go `net/netip` checks against `RemoteAddr`.
- Do not trust proxy forwarding headers for MVP.
- No auth token, pairing, or HTTPS for MVP.
- Remote access outside the home network is explicitly out of scope.
- Because MVP has no auth, the server must be easy to stop from Decky.
- Add clear warning text in Decky/settings whenever the server is running without auth, especially if LAN-only filtering is disabled.
- Logs must redact sensitive material. This includes Nexus API keys, future auth tokens, and URLs containing tokens or credentials.
- Add a token/session reset control only when auth or pairing exists.

## Logging And Diagnostics Guidelines

- Every feature change must include enough logging to diagnose failures over SSH without requiring the user to manually transcribe UI errors.
- Log each meaningful external action before and after it runs, including command name, return code, and bounded stdout/stderr where applicable.
- Log the decision path for platform integration features such as Decky process control, `nxm://` registration, browser launch, deployment, and filesystem writes.
- Never log Nexus API keys, `nxm://` `key`/`expires` values, future auth tokens, cookies, or credentials.
- Prefer stable log locations:
  - Decky plugin: `/home/deck/.local/state/decky-mod-manager/plugin.log`
  - Backend: `/home/deck/.local/state/decky-mod-manager/backend.log`
  - Steam frontend JS: `/home/deck/.local/share/Steam/logs/webhelper_js.txt`
- UI error messages should summarize the problem, while logs should contain the detailed diagnostic trail.

## Nexus Catalog Guidelines

- Name the upstream abstraction `RemoteModCatalog` unless we decide to shorten it during implementation.
- Nexus is the first remote catalog and remains the critical MVP source, but MVP must include the provider architecture and initial support for multiple popular non-Steam mod stores/upstreams.
- Pasted Nexus HTTPS URLs and pasted `nxm://` URLs are MVP.
- Direct local archive import is useful for testing and future flexibility, but Nexus URL import has priority.
- Use Nexus API-key authentication if an API key is available.
- Provide a settings field for manually entering the Nexus API key.
- Also attempt best-effort discovery of an existing Nexus/Vortex API key on the Steam Deck as a convenience.
- Nexus sign-in/API-key setup is MVP because downloads generally require an authenticated Nexus account.
- Firefox on the Steam Deck may already be signed into Nexus, but backend downloads should prefer API credentials over browser-session scraping.
- User is not Nexus Premium, so implementation must respect non-premium download behavior and rate limits.
- Cache Nexus metadata locally for installed/downloaded mods and provide a purge path.
- Preserve original source URL and provider metadata for every mod.
- Original downloaded archive retention should default to "keep until cleanup", with cleanup settings.
- Use Nexus/Vortex-style archive filenames when practical, but do not block core function on exact naming.
- Provider failures should be represented as retryable background jobs.

## Provider Model Guidelines

- Define a catalog interface around resolving URLs and downloads first, not broad browsing/search.
- A `RemoteModCatalog` should normalize external data into internal app entities:
  - Game
  - Mod
  - Mod version
  - File
  - Download
  - Source metadata
- Search/browse is non-MVP unless required by the Nexus download flow or explicitly prioritized for a verified provider.
- Each future catalog should own its credential requirements behind a common credentials interface.
- Avoid hard-coding Nexus game domain mappings where an API can resolve them.
- Implement verified popular non-Steam sources through the shared catalog pipeline; do not create one-off download or install paths per provider.
- Credential-gated providers can ship behind explicit key setup. Ask the user for API keys only when live verification requires them.
- Do not assume that a provider has a supported automated API because a website can be scraped. If official API/client/source verification fails, mark that provider deferred and allow selected-game direct archive import instead of adding scraper-dependent runtime behavior.

## Storage Guidelines

- App data default: `/home/deck/.local/share/decky-mod-manager`.
- Config default: `/home/deck/.config/decky-mod-manager/config.json`.
- Separate folders for:
  - `downloads`
  - `staging`
  - `db`
  - `logs`
  - `backups`
  - `tmp`
- Use per-game staging folders.
- Use per-mod unpacked staging folders under each game.
- Profiles share staged mod files. Profile state controls enabled mods, priority, conflicts, and deployment.
- Prefer staging on the same filesystem/partition as the target game when using hardlinks or move-like deployment.
- Support internal SSD, microSD, and external Steam library paths, including `/run/media/...`.
- Detect low disk space before download, extraction, staging, and deployment.
- Support cleanup settings for archives and cached metadata.
- Moving staging/download directories is post-MVP.

## Steam Discovery Guidelines

- Automatically detect Steam libraries from Steam config files.
- Scan default Steam Deck internal and microSD library locations.
- Show Steam app ID and install path in the UI.
- Detect Proton compatdata paths where mod files may need to be installed into prefixes.
- Detect whether an installed Steam game is running native Linux or Windows/Proton when that affects installer payloads, deployment roots, runtime dependencies, or launch tools.
- Manual game path registration is post-MVP.
- Non-Steam game support is post-MVP.
- Test primarily in Gaming Mode because Decky is the install/control surface.

## Deployment Guidelines

- Use a Vortex-inspired deployment model, adjusted for Linux/Steam Deck.
- Default to Vortex-style deployment behavior unless there is a clear Linux/Steam Deck reason not to.
- Treat DMM as owning a virtualized deployment layer, not the game's source files:
  - Original archives remain in DMM-managed downloads.
  - Extracted mod contents remain in DMM-managed staging.
  - The game folder receives only deployment artifacts that DMM can identify through its manifest, preferably links.
  - Profile switching reconciles DMM-owned deployment artifacts against the target profile; it must not copy profile state by mutating staged mod contents.
  - The game folder must never be treated as the canonical mod store.
- Primary deployment strategy should be selected per game and filesystem:
  - Hardlink when staging and game target are on the same filesystem and the game supports it.
  - Symlink when hardlink is not viable and the game supports symlinks.
  - Copy only as an explicit extension-declared strategy.
- Maintain a deployment manifest per game in SQLite.
- Deployment manifests must be profile-aware so each game/profile combination can be purged, repaired, and rolled back safely.
- Manifests are the authority for what DMM may remove or repair. Anything not present in the manifest is unmanaged and must be left alone unless the user enters an explicit adoption/cleanup flow.
- Never remove or overwrite unmanaged files without explicit user confirmation.
- Never delete parent game directories during purge. Purge removes only manifest-owned files/links.
- Existing game files, folders, or links should block deployment as conflicts unless they are recognized as current DMM-managed artifacts for the same game/profile transition.
- Profile switching should compute a transition plan:
  - Remove links/files owned by the old profile manifest and absent from the new profile.
  - Keep links/files already matching the new profile.
  - Add links/files required by the new profile.
  - Report conflicts before touching unmanaged files.
- For MVP, block deployment to games with detected Vortex/manual mod state unless explicitly enabled for a controlled test.
- Support purge in MVP.
- Support apply-time rollback in MVP: if deployment verification or manifest persistence fails, DMM must restore the files it changed during that attempt.
- User-visible rollback jobs are post-MVP and require a product decision about whether rollback means last deployment, named restore points, or profile-transition history.
- Verify deployed links/files against the deployment manifest.
- If broken deployment state is detected, ask before repairing.
- Use simple mod priority order for MVP.
- Mod enablement and priority are profile-specific.
- Expose file conflicts in the UI.
- Warn when two enabled mods write the same destination file.
- Per-file conflict winner selection is only MVP if simple priority is insufficient for the first supported game.
- Keep load order distinct from deployment priority when a game needs separate plugin/load-order handling.

## Install Pipeline Guidelines

- Use a staged, inspectable pipeline:
  - Resolve source URL.
  - Create persistent job.
  - Download archive.
  - Verify/checksum archive where possible.
  - Inspect archive entries before extraction.
  - Reject or warn on absolute paths and `..` traversal.
  - Extract into temporary workspace, not directly into deployable staging.
  - Detect top-level wrapper folder where obvious.
  - Detect supported installer metadata such as FOMOD.
  - For MVP, fail safely with an explicit unsupported-installer message when installer choices are required.
  - After MVP, pause for user installer choices when required.
  - Run an installer-planning stage that produces explicit install instructions, a mod type, deployable files, and target mappings.
  - Install only normalized deployable outputs into per-mod staging.
  - Plan deployment.
  - Apply deployment.
  - Persist logs and diagnostics.
- Do not treat a downloaded/extracted archive as a deployable mod solely because extraction succeeded.
- Do not add one-off rules for specific Nexus mod IDs, filenames, or individual apps. If an archive layout cannot be handled by the current provider/game installer planner, keep the archive cached and surface a blocked/unsupported install result.
- Do not invent deployment target paths for installed records that lack install-plan target mappings. Undeployable installed records should be recovered/restaged through the current planner or removed by the user.
- Follow Vortex's separation of download, install, mod type, and deployment: Nexus download metadata identifies source; game/provider installer planning identifies what gets staged; deployment manifests identify what DMM owns in the game folder.
- Model installer planning as metadata evaluation first: installer matchers classify archive shape, installer specs emit install instructions, spec-declared metadata extractors validate/ingest manifest attributes, mod types define deploy roots, and runtime requirements are derived from the resulting staged metadata.
- Installer planning must consider the detected game runtime platform when upstream metadata has platform-specific payloads. For Stardew MVP, native Linux installs use the Linux `install.dat` payload; post-MVP Windows/Proton support must select the Windows payload through extension metadata instead of generic app logic.
- Prefer declarative metadata extractors for common archive manifests before adding procedural parser code. A custom parser is acceptable when the source format has nested dependency/runtime semantics that cannot be represented by the generic extractor.
- Game-specific behavior belongs in Vortex-modeled specs or reviewed game-handler capabilities, not scattered through generic server, storage, deployment, or UI code.
- When installer metadata says a payload file should not overwrite a pre-existing game file, express that as a target policy on the install mapping and persist it in the staged manifest.
- Recognized but unsupported installer modes, such as future non-FOMOD custom installers, should produce blocked install candidates with precise installer IDs and reasons.
- Downloads and installs must be cancelable through context cancellation, and cancellation must clean up persisted captured-install/action-center state.
- Failed installs should leave enough diagnostics for debugging.
- Support common Nexus archive formats needed by Vortex-compatible downloads, starting with `.zip` and adding `.7z`/`.rar` as needed.
- FOMOD detection and a clear unsupported-installer failure are part of the current MVP slice.
- Interactive FOMOD installer support is required for complete no-phone modding and must not silently stage archives that need user choices before that UI exists.
- Auto-enable may proceed only for installers that can produce a complete plan without user choices, or for installer-choice mods with a saved preset/headless choice set that was previously chosen by the user.
- First-time FOMOD installs must pause as an installer-choice request until the user completes the options either in a phone/tablet web UI or a Decky-native choice surface.
- Persist pending installer-choice jobs after the post-MVP installer architecture is selected, so the user can disconnect and resume choosing options later.
- Prefer safe Go libraries for archive handling where practical.
- Shelling out to `7z` or `bsdtar` is acceptable only through a constrained wrapper with path validation, argument hygiene, and clear error handling.
- Installing extraction tools on the Deck is acceptable for debugging, but production should not rely on mutable system packages if avoidable.
- Archive helper dependencies should be visible in settings/debug UI.
- Dependency UI should show missing tools in red and installed tools in green.
- If the app can install a dependency safely, show an install button.
- If the app cannot install a dependency safely, show concise install guidance and a relevant help link.
- Recheck dependency status after install attempts or user action.

## Game Handler Guidelines

- MVP should start with generic reusable handler primitives plus Vortex-modeled metadata specs declared by explicit game extensions.
- A game handler should define:
  - Game identity and Steam app IDs
  - Supported Nexus game domains
  - Deployment eligibility policy, including whether review-state deployment is allowed for a controlled supported target
  - Install roots and mod target directories
  - Proton prefix usage
  - Archive inspection/install rules
  - Install-plan generation from archive layout and provider metadata
  - Supported mod types and their deploy roots
  - Runtime/dependency requirements derived from handler/provider metadata, not hard-coded one-off assumptions
  - Validation checks
  - Deployment strategy preferences
  - Load order rules if applicable
- Implement game extensions as internal Go packages for MVP. External/community extension packaging is future work, but the core/extension behavior boundary still applies now.
- Prefer declarative/spec-driven game metadata inside those Go packages. Procedural code is acceptable only when the installer behavior, metadata extraction, or runtime validation genuinely cannot be represented by the current metadata evaluator; when that happens, extend the evaluator before adding one-off logic.
- Do not add game-specific runtime requirements directly to the generic server/UI. If a game needs SMAPI, script extenders, mod loaders, launch options, or equivalent runtime integration, expose that through handler/provider metadata attached to the staged mod, then have diagnostics evaluate the enabled mod types.
- Launch tools are game-extension capabilities. A handler may declare tools such as SMAPI, their required files, platform-specific executable names, desired Steam launch behavior, and whether the tool should become the default/primary launch target when required by enabled mod metadata.
- Primary launch-tool selection must be expressed in extension metadata/rules, such as "enabled mods of these mod types or metadata requirements require tool `smapi` as the primary launch tool." Generic code evaluates the rule; it must not know Stardew-specific tool semantics.
- Stardew's extension should mirror Vortex's primary-tool model: when enabled mods require SMAPI and SMAPI is deployed, DMM should configure the Steam launch path to run SMAPI rather than making generic server code know Stardew-specific launch rules.
- If the requirement cannot be derived from the current handler/provider metadata, surface it as an unsupported/missing handler capability rather than pretending DMM can validate it generically.
- Prototype/TexMod-style support must be expressed through extension-declared launch-tool and installer capabilities. Prefer a typed optional dynamic launch-input spec on the generic launch-tool API, such as generated profile config or enabled-mod file list inputs, rather than a game-specific branch or a stringly special case. Static launch tools such as SMAPI must not be forced to opt into dynamic inputs.
- External commands are disabled by default. They may be justified later for tools such as script extenders, patchers, load-order tools, archive conversion, or game-specific installers.
- Any future external-command handler must require explicit user approval and show the command/action clearly.
- Interactive FOMOD installers should be driven through the web UI once implemented.
- Non-FOMOD interactive/custom installer systems are post-MVP unless the first selected game requires them.
- Unsupported mods should fail with a clear reason instead of silently staging unusable files.
- Handler logic must be layout/mod-type driven, not hard-coded to one Nexus file or one manually observed archive.

## Backend Architecture Guidelines

- Use idiomatic Go with simple package boundaries.
- Prefer the standard library HTTP stack for MVP. Add Chi only if route management becomes noisy.
- Avoid GraphQL.
- Use REST endpoints for UI and future helper clients.
- Use Server-Sent Events for job/status updates unless WebSockets become necessary.
- Do not use the HTTP `QUERY` method for MVP. Use `GET` for simple reads, `POST` for commands and complex query bodies, `PUT/PATCH` for updates, and `DELETE` for destructive actions.
- Use SQLite for persistent state.
- Persist job state across restarts.
- Persist structured identifiers on long-lived jobs, such as app ID, catalog, provider game domain, mod ID, and file ID. UI filtering and diagnostics should use these fields first and treat titles/messages as display text, not business state.
- Use JSON config for MVP.
- Use structured JSON logs on disk and human-readable summaries in UI.
- Keep domain/application logic independent from Decky and HTTP handlers. Decky and phone UI should be clients of the same backend capabilities.
- Suggested package shape:
  - `cmd/dmm-server`
  - `internal/config`
  - `internal/storage`
  - `internal/steam`
  - `internal/catalog`
  - `internal/catalog/nexus`
  - `internal/library`
  - `internal/archive`
  - `internal/install`
  - `internal/deploy`
  - `internal/games`
  - `internal/jobs`
  - `internal/server`
  - `internal/web`
  - `internal/decky`
- Keep interfaces small and driven by concrete MVP needs.

## Frontend Guidelines

- Use a phone/tablet-first Svelte + Vite web UI served by the Go backend.
- Build the Svelte UI as static assets and ship them with the Decky plugin/backend package.
- The Go backend remains the source of truth for business logic. Svelte owns presentation, interaction state, and submitting user intent.
- Do not use SvelteKit for MVP. Go handles HTTP serving and API routing.
- Keep the Svelte app intentionally thin: showing data, navigating local UI state, collecting user choices, and interacting with backend APIs.
- Avoid moving Nexus, archive, deployment, profile, conflict, or game-handler business rules into the frontend.
- Frontend assets may be served from the plugin bundle; embedding into the Go binary is optional and should follow Decky packaging needs.
- PWA/offline support is post-MVP, but cached installed/downloaded metadata can be shown when Nexus is unavailable.
- Use one realtime update mechanism consistently. Default: Server-Sent Events.
- Avoid marketing-page structure. The first screen is the management app.
- Use touch-appropriate controls, compact status, and stable layouts that do not shift during job updates.
- Support common iPhone, Android phone, iPad, and Android tablet viewport sizes as first-class MVP layouts.

## Data Model Guidelines

- MVP entities:
  - Game
  - Profile/loadout
  - Steam library
  - Remote catalog account
  - Source metadata
  - Mod
  - Mod version
  - Archive/download
  - Installed mod
  - Profile-specific enabled/disabled state
  - Profile-specific mod priority
  - Deployment
  - Deployed file
  - File conflict
  - Job
  - Log/event
- A mod should belong to one game in the MVP data model.
- Mod versions must be first-class to support upgrades and downgrades.
- Disabled mods should remain in the database and staging unless explicitly uninstalled.
- Each game should have at least one default profile.
- Track checksums for archives and staged files where practical.
- Keep enough metadata to recreate, audit, purge, and rollback deployments.
- Profiles mean switchable mod sets/loadouts for a game and are MVP.
- Profile switching should trigger a deployment plan that purges/replaces only files needed to move from the old profile to the selected profile.

## Testing Guidelines

- Focus tests where bugs can damage game folders:
  - Deployment planning
  - Manifest writing
  - Purge
  - Rollback
  - Conflict detection
  - Archive path traversal checks
  - Steam library discovery parsing
- Use temporary directories for deployment tests on local macOS/Linux.
- Keep tests lightweight and behavior-focused. Do not spend MVP time recreating full Steam, Decky, Nexus, Vortex, or game environments virtually when that turns into more work than the feature itself.
- Prefer small tests around stable contracts, path safety, planner output, persisted state, and extension metadata evaluation over brittle mocks that are over-fitted to one implementation detail.
- Use enough coverage to protect the mod deployment pipeline, but avoid test designs that slow development by fighting mocked environments that still cannot prove real Steam Deck behavior.
- Manual testing on the actual Steam Deck is acceptable and expected for environment-specific behavior.
- Mocked Nexus tests are useful but not required before the first vertical slice.
- Real Nexus downloads should be manual/integration tests, not default unit tests.
- Decky plugin testing can be manual for MVP.
- Playwright UI tests are post-MVP unless the UI becomes complex enough to justify them earlier.
- Test on macOS locally for quick cycles and on the Deck for final behavior.

## Build And Release Guidelines

- Build the backend for Linux AMD64 for Steam Deck.
- Prefer a self-contained Go binary inside the Decky plugin package.
- Provide a Makefile for common tasks.
- Decky plugin packaging is the only MVP install path.
- Manual install scripts are non-MVP.
- GitHub Actions should eventually build release artifacts.
- Include checksums for release artifacts.
- Self-update is non-MVP; updates should happen through Decky/GitHub release flow later.
- License is GPL-3.0-or-later.

## Licensing Guideline

- Use GPL-3.0-or-later.
- The project owner can re-license their own code later if desired.
- Before accepting outside contributions, add a `CONTRIBUTING.md` note or CLA-style policy so contribution rights are clear.

## Immediate Milestones

1. Inspect the Steam Deck filesystem and Steam libraries over SSH.
2. Choose first supported game/mod vertical slice from installed games, preferring a clean game over the originally mentioned candidates if needed.
3. Scaffold Go module and package structure.
4. Implement config, data directories, logging, SQLite, and backend health endpoint.
5. Implement Steam library and installed-game discovery.
6. Implement job system with persisted state and WebSocket domain events.
7. Implement Nexus URL parsing for HTTPS and `nxm://`.
8. Implement Nexus API-key discovery/config and metadata/download flow.
9. Implement archive inspection and safe extraction.
10. Implement generic staging install.
11. Implement deployment planner, conflict detection, symlink/hardlink/copy strategy, manifest, purge, and apply-time rollback.
12. Implement mobile-first web UI for games, imports, mods, jobs, deploy, purge, conflicts, and logs.
13. Implement Decky plugin wrapper with start/stop, URL, QR, settings, and debug tab.
14. Package and test on the Steam Deck in Gaming Mode.

## Open Questions

These need explicit answers before or during implementation:

1. None currently. Revisit after the latest Deck package is installed and tested in Gaming Mode.

## Broken Vortex / Existing Mods Guideline

- Vortex is currently broken on the Steam Deck, and some games may already have mods installed.
- MVP should not attempt global Vortex cleanup.
- Scans should classify games as clean, possibly modded, Vortex-managed, unsupported, or safe-to-test.
- Broken Vortex cleanup/import should be a later explicit wizard with dry-run, backups, confidence levels, and user confirmation.

## Reference Notes

- Vortex's public deployment documentation describes hardlink deployment as the default and states that deployment checks/rebuilds links and purge removes deployed links.
- Vortex requires the staging folder and game mod folder to be on the same partition for hardlinks.
- Vortex documents symlink deployment as cross-partition-capable, but with game compatibility caveats.
- Vortex documents VFS as intentionally not its default due to portability, maintenance, and performance tradeoffs.
- Nexus API documentation is the source for Nexus integration details during implementation.
