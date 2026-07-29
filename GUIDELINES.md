# Decky Mod Manager Guidelines

These guidelines translate `notes.md` into build decisions. Treat `notes.md` as the raw Q&A record and this file as the working development contract.

## Product Direction

- Build a Steam Deck-first mod manager named Decky Mod Manager.
- MVP install path is a Decky Loader plugin that bundles and controls a Go backend.
- The phone/tablet browser UI is the primary MVP management surface.
- MVP supports Steam games on Steam Deck only.
- MVP focuses on Nexus Mods "Mod Manager Download" / Vortex-compatible flows.
- MVP workflow: capture a Nexus `nxm://` Mod Manager Download link from the Deck browser, or paste a Nexus `https://www.nexusmods.com/...` URL / `nxm://` URL into Decky or the phone UI, resolve it, download it, approve it if approval is required, and then manage it as a mod in the selected profile.
- Staging, install planning, deployment manifests, and file operations are implementation details for the default experience; they remain inspectable through advanced/power-user surfaces.
- User-level `nxm://` OS protocol registration is MVP for the Deck browser flow; pasted `nxm://` links remain the fallback.
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
- Avoid initial testing on Witcher 3, FF7 Rebirth, Fallout 4, Skyrim, Cyberpunk 2077, and Oblivion Remastered because Vortex/manual mod state was detected.

## UX Guidelines

- Phone and tablet UI are first-class MVP targets.
- UI is touch-first and dark-mode-only for MVP.
- Desktop browser layout is non-MVP, but the app should not be unusable on desktop-sized browsers.
- The default user mental model is profile-first:
  - Select game.
  - Use the selected/default profile.
  - Download or approve a mod.
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
- Download approval is required by default. A user setting may allow automatic download approval for faster Deck-only flows, but the setting must be explicit and easy to disable.
- Gaming Mode must show Decky notifications for Nexus request capture and install/download transitions, especially when the Nexus browser page only says that a download is starting.
- FOMOD installer choices should be presented as clear touch-friendly forms in the browser UI.
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
- Nexus is the only remote catalog required for MVP.
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
- Search/browse is non-MVP unless required by the Nexus download flow.
- Each future catalog should own its credential requirements behind a common credentials interface.
- Avoid hard-coding Nexus game domain mappings where an API can resolve them.
- Design for future catalogs such as ModDB, Thunderstore, GitHub Releases, direct URL, and local archive import, but do not implement them before the Nexus vertical slice works.

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
  - Copy only as an explicit fallback.
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
- Do not invent deployment target paths for staged records that lack install-plan target mappings. Legacy staged records should be recovered/restaged through the current planner or removed by the user.
- Follow Vortex's separation of download, install, mod type, and deployment: Nexus download metadata identifies source; game/provider installer planning identifies what gets staged; deployment manifests identify what DMM owns in the game folder.
- Model installer planning as metadata evaluation first: installer matchers classify archive shape, installer specs emit install instructions, spec-declared metadata extractors validate/ingest manifest attributes, mod types define deploy roots, and runtime requirements are derived from the resulting staged metadata.
- Prefer declarative metadata extractors for common archive manifests before adding procedural parser code. A custom parser is acceptable when the source format has nested dependency/runtime semantics that cannot be represented by the generic extractor.
- Game-specific behavior belongs in Vortex-modeled specs or reviewed game-handler capabilities, not scattered through generic server, storage, deployment, or UI code.
- When installer metadata says a payload file should not overwrite a pre-existing game file, express that as a target policy on the install mapping and persist it in the staged manifest.
- Recognized but unsupported installer modes, such as future non-FOMOD custom installers, should produce blocked install candidates with precise installer IDs and reasons.
- Downloads and installs must be cancelable through context cancellation, and cancellation must clean up persisted pending request state.
- Failed installs should leave enough diagnostics for debugging.
- Support common Nexus archive formats needed by Vortex-compatible downloads, starting with `.zip` and adding `.7z`/`.rar` as needed.
- FOMOD detection and a clear unsupported-installer failure are part of the current MVP slice.
- Interactive FOMOD installer support is post-MVP and must not silently stage archives that need user choices before that UI exists.
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

- MVP should start with a generic file/folder handler plus Vortex-modeled metadata specs for the first selected game.
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
- Implement handlers as internal Go packages for MVP.
- External plugin handlers are future work.
- Prefer declarative/spec-driven game metadata inside those Go packages. Procedural code is acceptable only when the installer behavior, metadata extraction, or runtime validation genuinely cannot be represented by the current metadata evaluator; when that happens, extend the evaluator before adding one-off logic.
- Do not add game-specific runtime requirements directly to the generic server/UI. If a game needs SMAPI, script extenders, mod loaders, launch options, or equivalent runtime integration, expose that through handler/provider metadata attached to the staged mod, then have diagnostics evaluate the enabled mod types.
- If the requirement cannot be derived from the current handler/provider metadata, surface it as an unsupported/missing handler capability rather than pretending DMM can validate it generically.
- Prototype/TexMod-style support is non-MVP.
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
- Do not spend MVP time building a full fake Steam Deck environment.
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
6. Implement job system with persisted state and SSE status stream.
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
