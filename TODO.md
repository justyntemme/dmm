# TODO

## Active Execution Order

1. Fix and validate generic installer-choice/multi-choice application.
   - Latest deployed state: `/api/games/{appID}/install-candidates/{candidateID}/apply` now returns a running job immediately and completes installer application in the background, so Decky/Python request timeouts no longer cancel long archive staging.
   - Live Deck validation: Ridgeside Village multi-component selection returned `running`, completed as `Installed Ridgeside Village [SMAPI component] disabled; enable it to deploy`, removed the install candidate, and preserved the selected profile.
2. Finish valid FOMOD live validation and any remaining FOMOD bugs.
   - Use browser-generated Nexus `nxm://` links for real validation; direct Nexus HTTPS file downloads are browser-required on the current non-premium path.
   - Latest deployed state: the Fallout 4 Pip-Boy Flashlight FOMOD installed from the cached browser-captured Nexus archive after DMM skipped missing selected `README` choice sources as Vortex-compatible warnings while keeping missing required install files fatal.
   - Latest deployed state: Decky no longer immediately reopens the same FOMOD modal after `Apply Choices`; candidate auto-display is suppressed while the backend apply job is in progress and released when the installer-choice job reaches a terminal state.
   - Latest deployed state: installed mods now have an explicit reconfigure path. Normal reinstall reuses saved choices; Decky `Menu Reconfigure` and the phone/tablet `Reconfigure` button force installer choices to appear again from the cached archive.
3. Implement MGSV/SnakeBite-style packed archive mutation as a generic extension-framework capability.
   - Required before enabling MGSV `.MGSV` packages: safe QAR/FPK mutation, backup, verify, rollback, and extension-owned game-specific directives.
4. Continue source-verified target extension validation and fill only the generic API gaps exposed by real Vortex/source behavior.

## Current Manual-Test Loop

1. Validate the latest Decky full-route UI in Gaming Mode.
   - QAM is now a compact launcher/status panel, and `/decky-mod-manager` owns the dense tabs through Decky's native route/tab surface.
   - Latest local fixes keep the Manage page bottom action above Steam's button-hint bar and make the Nexus browser modal use fixed filters plus a bounded result scroller so result rows cannot collapse into slivers.
   - Latest local cleanup renames the Decky `Mods` route tab to `Games`, removes the global external `Open Nexus Mods` button, and keeps Nexus browsing inside the game-scoped DMM modal.
   - Latest code/deployed state: after a successful browser-generated Nexus `nxm://` capture, the controlled DMM browser closes back to the selected game's `Games` page instead of the generic home route.
   - Latest code/deployed state: the selected game page now has an explicit `Explore Mods` source section. Nexus appears as a selectable source action instead of being hidden as an implicit game-header action.
   - Needs manual validation: D-pad focus, game row highlighting, mod row highlighting, Nexus modal sizing, and Manage paste-link spacing after the latest package.

2. Validate the latest captured-install and FOMOD paths with real browser-generated `nxm://` links.
   - Direct DMM Nexus HTTPS file installation now reports a clean browser-required error for non-premium Nexus accounts instead of pretending the install was accepted.
   - Latest code/deployed state: pasted plain Nexus HTTP/HTTPS pages in both phone/tablet and Decky `Mod Link` now open the controlled Deck browser automatically so the user can click Nexus Mod Manager Download and capture the short-lived `nxm://` credential.
   - Remaining live path must use the Deck browser's Mod Manager Download button so DMM receives the short-lived `nxm://` key.

3. Validate rollback/recovery surfaces in the phone/tablet UI.
   - Backend restore-last-applied and deployment-history endpoints exist and were API-checked on the Deck.
   - The web Advanced Recovery UI still needs visual review and failure/toast validation before the MVP rollback item can be marked complete.

## Current MVP Priorities

1. Expand provider/source architecture beyond Nexus as MVP scope.
   - Nexus remains the first implemented provider, but the architecture must now treat popular non-Steam stores as MVP-compatible sources once verified from official APIs/clients.
   - Do not implement a provider by guessing from one archive. Clone/inspect official clients, API docs, schemas, or source first.
   - Every mod row should show a small source pill/tag such as `Nexus`, `Steam Workshop`, `Thunderstore`, `Modrinth`, `GameBanana`, `ModDB`, `GitHub`, `Direct`, or `Local`.
   - Persist source/catalog identity through capture, cache, install, profile state, updates, removal, rollback/history, and UI filtering.
   - GameBanana MVP scope is verified URL import for supported submission pages through the official Core/Item/Data API; direct `/dl/` links are selected-game direct archive imports unless a reliable source-aware parent lookup is verified.
   - Steam Workshop is not a normal browsed catalog for MVP. DMM should manage already-installed/subscribed Workshop content where verified Steam/Decky APIs allow enable, disable, unsubscribe, and load-order movement.
   - Latest code state: phone/tablet Mods renders Steam Workshop items alongside DMM-managed mods with source pills and queued Steam actions. Review remains diagnostics-only for Workshop coexistence.
   - Latest code state: job/history and selected-game activity rows show source pills whenever a catalog/source is known.
   - Latest code state: installed-mod update checks are source-specific. Nexus update checks/downloads are implemented; GitHub Releases, Modrinth, Thunderstore, GameBanana, mod.io, and CurseForge implement optional catalog update capabilities; unsupported providers persist an explicit unsupported update status with provider-specific messaging instead of being skipped.
   - Latest live state: `testing/live_provider_resolve_check.sh` passes against the Deck for Direct, GitHub Releases, Thunderstore, Modrinth, and GameBanana. mod.io and CurseForge remain key-gated and skipped unless test URLs/API keys are supplied. ModDB remains deferred.

2. Finish MVP API auth/pairing.
   - Decky must generate a process-scoped token on backend launch, pass it to the backend, and include it in every Decky bridge/backend request.
   - The `nxm://` handler must read the same local token file so browser-generated Nexus captures still reach `/api/captured-installs`.
   - The phone/tablet URL should pair through a URL fragment token, then send `X-DMM-Token` on API calls and a WebSocket query token for realtime events.
   - `/api/health` may stay unauthenticated for local liveness checks; every other API/read-model/mutation/event endpoint must reject remote clients without the token when auth is enabled.
   - Add visible Decky security copy and a token/session reset action before MVP approval.

3. Finish local archive import browser UX.
   - Hide Deck-side archive browsing behind an accordion button such as `Import Mod Archive`.
   - Opening the accordion must browse the Deck user's Downloads/DMM Intake folders first, with Up Directory, Enter Path, Refresh, folder rows, and archive import rows.
   - The phone/tablet app must only see DMM-approved archive roots, not arbitrary Deck filesystem paths. Reject symlink escapes and parent traversal in the backend.
   - Keep local archive import game-scoped and install-to-profile aware. It should reuse the same captured-install/profile/FOMOD pipeline as other sources.

4. Finish phone/tablet UI refresh validation under the event model.
   - Captured installs should update immediately after capture, local install, cancel, retry, failure, and completion.
   - Action buttons must become disabled/in-flight immediately so the user cannot submit the same action twice.
   - Profile mod lists, deployment status, jobs, runtime requirements, launch actions, and mod update badges should update without manual browser refresh.
   - Latest code state: global Action Center now refreshes jobs and installer candidates from action-related WebSocket events even when no game is selected, while selected-game panes refresh only when the event belongs to the selected game.
   - Latest live state: failed captured-install actions can be dismissed through the normal cancel path; stale failed Fallout actions were canceled on the Deck and no open Action Center jobs remained.

5. Finish MVP FOMOD / installer-choice support.
   - Download/cache happens immediately as normal.
   - Installer-choice actions must persist from local cached archive state.
   - Phone/tablet UI must show touch-friendly choices and apply selected files.
   - Decky modal flow is required for no-phone installs where Decky can safely present the modal.
   - Auto-install/auto-enable must pause for FOMOD unless a compatible saved preset exists.
   - Live validation must use a browser-generated `nxm://` FOMOD link because direct Nexus HTTPS file installs are blocked for non-premium accounts.
   - Latest code state: phone/tablet and Decky installer-choice UI now presents evaluated FOMOD steps as a wizard instead of one long form. `SelectAtMostOne` groups expose a Vortex-style `None` option, and Required/NotUsable choices are locked in the UI while backend validation remains authoritative. Backend-evaluated option state is serialized as `effective_type`, preserving the parsed/default FOMOD `type` so changing earlier wizard choices can be re-evaluated correctly.
   - Latest code/deployed state: the shared installer-choice schema now supports extension-owned text groups. Ghost Recon Breakpoint `.data` folder and loose `.data` archive classes use a source-verified `.forge` folder text prompt through the same phone/tablet Action Center and Decky modal flow instead of remaining blocked.
   - Latest live state: deployed package was smoke-tested with a synthetic local Fallout FOMOD. The backend created a `needs_choices` candidate, preserved default option `type`, emitted computed `effective_type`, and the candidate was cleared without touching game files. A real Stardew Nexus/browser FOMOD (`Neural Harvest`) opened the Decky modal, proving the no-phone modal display path works, but the archive is malformed because its `ModuleConfig.xml` references missing `Common/...` files. DMM now blocks that archive with a clear review reason and keeps it visible instead of closing the flow as successful. A valid Nexus/browser FOMOD success fixture is still required.

6. Redesign the phone/tablet mod-management UI around the profile-first model.
    - The primary surface should be the selected profile's enabled/disabled mod list.
    - Captured-install actions should read as local install choices, not network download approvals.
    - Staging, deploy preview, purge, repair, conflicts, and file-level operations should be advanced/power-user views unless they require immediate action.
   - Latest live state: healthy installed mods now report API `status: "installed"` instead of the old `status: "staged"` vocabulary. Web/Decky normal mod rows consume `installed` as the healthy state while staging remains an internal filesystem/deployment concept. Package install, live Stardew API verification, and live profile toggle/deploy smoke passed on the Deck.
   - Latest code/live state: active profile selection, install-to-profile targeting, profile copy/move, and profile-scoped removal are implemented and committed in `b1fde4c`. The latest Deck package reports `b1fde4c`; `live_profile_toggle_check.sh` and `live_stardew_mod_files_check.sh` passed, confirming profile toggles reconcile deployment and live game files are DMM-owned symlinks into staging.
   - Latest live state: after the current package, Deck-side `live_profile_toggle_check.sh` passed and Deck-side `live_stardew_mod_files_check.sh` confirmed six DMM-managed Stardew manifest links plus SMAPI runtime/launch OK.
   - Latest code state: the selected-profile summary tile now labels the zero-count action button as `Action Center` instead of a vague `Open`, so the game-level Action Center remains visible as a command even when there are no open actions.

7. Improve deployment language.
   - Replace file-operation-heavy copy with user-centered state: installed, disabled, enabled, applied, needs restart, blocked, conflicts, failed.
   - Keep exact file operations visible in advanced details and logs.

8. Finish MVP user-facing rollback.
   - Simple users should see a safe primary action such as `Restore last applied state` only when a rollback is available.
   - Power users should get an advanced rollback surface with deployment history, named restore points, preview, retention controls, and rollback job details.
   - Rollback must affect only DMM-owned deployment artifacts tracked by manifests and must not imply unmanaged/manual files can be restored.
   - Rollback actions must be jobs, must log every file operation, and must preserve the profile-first UX.
   - Backend restore/history exists and was live API checked; UI review and failure/toast validation remain.
   - Latest live state: `live_rollback_check.sh` passed against Stardew and completed rollback job `job-249` with no repair issues.

9. Manage mod updates.
   - Users should be able to check installed mods for source-specific updates, see clear current/available/unknown/unsupported/error state, and install available updates through the same captured-install/profile pipeline used by normal downloads.
   - Update installs must preserve source tags, profile targeting, FOMOD/installer-choice pauses, disabled-by-default safety, Action Center visibility, job/toast state, deployment manifests, and rollback behavior.
   - Nexus, GitHub Releases, Modrinth, Thunderstore, GameBanana, mod.io, and CurseForge should use verified source-specific update resolvers where available. Direct, local archive, Steam Workshop, and unsupported providers should show explicit unsupported update state.
   - Current implementation note: update checks and update install routes exist, and live Stardew update-check API validation passed with all installed mods reported current. Still needs Gaming Mode Decky visual review and a real available-update validation with a provider/file that has a newer version.

10. Finish Decky-side game/mod management polish.
   - Game selection should support search, filters, favorites synced with the phone/tablet web app, and recent-game sorting.
   - Mod rows must be controller-navigable, readable at Decky sidebar width, and free of clipped action labels.
   - Users should be able to enable, disable, remove from the selected profile, reinstall, and check updates from the Decky plugin after selecting a game.
   - Latest code state: Decky remove now uses the profile-scoped remove route and keeps cached/staged installed mod files recoverable; the obsolete game-level delete endpoint and bridge path were removed.
   - The Decky route tab is now labeled `Games` because it is the entry point for selecting a game and then managing that game's mods.
   - Latest package must still be validated in Gaming Mode for D-pad behavior, Nexus modal sizing, Manage paste-link spacing, and update-check row display.

11. Validate and polish Nexus browsing/search inside DMM.
   - Decky and phone/tablet surfaces can search/browse Nexus through the backend adapter.
   - Direct HTTPS file installs now show a clear browser-required error for non-premium accounts.
   - Current Decky flow: select a game, choose `Explore Mods` > `Nexus Mods`, open a result page in DMM's controlled BrowserView, then click Nexus' own Mod Manager Download/Vortex link so DMM receives the browser-generated `nxm://` credential.
   - Current phone/tablet paste flow: plain Nexus HTTP/HTTPS page URLs are not direct installs. A single pasted Nexus page opens as a Deck browser handoff with a retryable `Open on Deck` prompt; the actual download starts only after Nexus generates an `nxm://` link in DMM's controlled Deck browser.
   - Current Decky UX note: keep the `Vortex Only` toggle for Nexus because it is useful while Nexus is the only supported browse provider. When additional browse providers exist, redesign it into a broader source/compatibility filter instead of removing the current behavior.
   - Latest live state: Decky Nexus search is controller-usable enough to reach and test a real Nexus FOMOD modal from Gaming Mode.
   - Latest code/deployed state: pasted Nexus mod/file pages now route into that same BrowserView handoff instead of leaving the user at a text-only browser-required message.
   - Do not reintroduce a primary `Show Files` / direct Nexus file-list install path for non-premium Nexus downloads. Direct API attempts are allowed only as advanced/update helpers that fail cleanly with browser-required messaging.
   - Latest deployed state: commits `6c72f6d`, `fb19ddf`, and `9a6dca8` align source copy and update/direct HTTPS browser-required paths with the controlled-browser handoff. Browser-required Nexus update/paste flows no longer create failed captured-install jobs; they return a completed handoff job plus `browser_required: true`.
   - Latest live state: package `9a6dca8` was installed on the Deck, package parity passed, `/api/health` returned OK, live web assets were reachable, and three stale pre-fix failed captured-install probe jobs were cleared through the normal API.
   - Remaining validation: visual Gaming Mode modal review, real browser-generated `nxm://` file-install flow, and whether non-Vortex results belong behind an advanced toggle.

12. Continue Vortex-style extension parity.
   - Maintain the rule that every supported game has an explicit first-party Go extension and game-specific behavior lives in that extension. Core packages provide reusable capabilities only; they must not claim support for a game, domain, archive layout, launch tool, runtime dependency, Workshop behavior, or load-order file without an extension declaring it.
   - Keep verifying Vortex source before implementing game-specific installer/load-order/runtime behavior.
   - Completed locally: F4SE/SKSE dirty-state marker detection moved out of generic Steam discovery and into extension-declared unmanaged marker metadata consumed by the generic discovery/diagnostic path.
   - Completed locally: Bethesda `plugins.txt` / `loadorder.txt` dirty-state detection moved out of generic Steam discovery and into Fallout 4 / Skyrim SE extension-declared unmanaged marker metadata.
   - Completed locally: the generic plugin activation format formerly shaped around Fallout 4 is now named `asterisked`, and Fallout/Skyrim extensions select that neutral format explicitly.
   - TexMod path: model TexMod through the existing extension-declared launch-tool capability where possible, with Prototype/Prototype 2 extensions declaring the TexMod executable, `.tpf` payload installer rules, required files, and primary launch behavior. If TexMod needs enabled-mod package lists or generated config, add a typed optional dynamic launch-input spec to the generic launch-tool API, then have the Prototype extensions use it. Static launch tools such as SMAPI should continue to use the simple executable/arguments/required-files path.
   - Remaining major gaps include advanced Witcher menu/script/config merge behavior, LOOT/load-order sorting, more game-version providers, and live Fallout/Skyrim/Witcher/FF7 validation.
   - Latest code state: Ghost Recon Breakpoint now covers Vortex's free-text `.forge` folder prompt generically. Post-deploy extension messages now become deduped `extension-notice` Action Center jobs with Decky toast coverage, and Ghost Recon Breakpoint registers a Vortex-verified AnvilToolkit repack reminder through its extension `did-deploy` hook. The remaining Breakpoint gap is a generic extension tool-runtime action that can launch AnvilToolkit from the notice, plus representative archive validation.
   - Latest code state: Witcher 3 now mirrors Vortex's post-deploy Script Merger reminder through an extension-owned `did-deploy` notice when enabled DMM-managed Witcher mod files change. DMM still does not run or automate Script Merger; a generic extension tool-runtime action is needed before that notice can become an actionable launch button.
   - Latest code state: Witcher 3 now also declares Vortex's `ignoreConflicts`/`ignoreDeploy` patterns for `README.TXT` and `**/*.PART.TXT`, so DMM's generic deployment planner skips those extension-declared artifacts without Witcher-specific core code.
   - Latest code state: Portal 2 now exposes the Vortex extension page's `portal2_dlc3` setup requirement as an extension-owned runtime diagnostic tied to enabled Portal 2 mods. DMM still deploys through the generic target-root planner; the diagnostic simply tells users why that folder matters.
   - Latest code/deployed state: source review of `https://github.com/Pickysaurus/vortex-jedi-survivor` found that Jedi Survivor multi-PAK archives use checkbox-style selection and preserve the selected PAK variants' original relative destinations. DMM now mirrors that through the extension-owned generic installer-choice path instead of forcing exactly one `.pak`. Package `a7bfb72` was deployed and live extension/provider checks passed.
   - Latest code state: the current Battlefront II Nexus/Vortex extension page says deployed `.fbmod` files still need a Frosty Mod Manager handoff after deployment. The Battlefront extension now emits a `did-deploy` Action Center notice for enabled `.fbmod` deployments instead of hiding that manual step in logs.
   - Latest live state: package `25ecd7d` added narrow Nexus research-blocked extensions for Bastion, Blasphemous, Brawlhalla, Command & Conquer Generals, Dave the Diver, Half-Life, Mirror's Edge, Mr. Prepper, Persona 5 Royal, Potion Craft, Quake 4, Riders Republic, Rome: Total War, Spelunky, STEINS;GATE, The Division 2, and The King is Watching. These expose verified Nexus browse/capture entry points and live diagnostics, but intentionally block archive deployment until source or representative archive behavior is verified.
   - Latest live state: after deploying `25ecd7d`, the visible unsupported game count dropped from 34 to 17. The remaining unsupported games should not receive placeholder extensions until an exact Nexus/Vortex/Workshop/source signal is verified.
   - Latest live state: after the current package, `/api/games` reports zero unsupported installed games. Fallout 4 remains in review only because its extension detected an existing F4SE loader; that marker is extension-owned.

13. Validate Steam Workshop coexistence/actions.
   - Workshop content should not make a game undeployable when the extension declares coexistence safe.
   - Backend/Decky queued Workshop actions exist; live mutation behavior still needs a real Workshop-enabled game validation pass.
   - Latest live state: Decky startup sync uses Steam's verified `GetSubscribedWorkshopItemDetails(appID, publishedFileIDs)` method to enrich downloaded/subscribed IDs. Kenshi now shows real Workshop titles from SteamClient details after startup sync.
   - Latest live state: after the newest package/reboot, `/api/games/233860/workshop` returned 15 Kenshi Workshop rows, 15 titled rows, and known disabled state for the first visible rows.
   - Latest live state: package `f89dd7b` added verified Workshop-only extensions for Besiege, Command & Conquer Generals Zero Hour, Cultist Simulator, DiRT Rally, Plague Inc: Evolved, Stacklands, Transport Fever 2, and We Who Are About To Die. Live `/api/extensions` validation showed five Steam Workshop actions and no Nexus domains for each.
   - Latest live gap: real enable/disable, unsubscribe, and load-order mutation still need safe live validation.
   - MVP Workshop scope is manage-installed content only: disable, enable, unsubscribe/remove, and load-order movement through verified Steam/Decky APIs. Do not build a Steam Workshop search/browser UI for MVP.

14. Verify toast coverage.
   - Re-check capture, downloaded, install waiting, installing, installed, failed, FOMOD-choice-required, deploy success, deploy failure/rollback, and launch-action-required notifications in Gaming Mode.

15. Code-review and remove deprecated UX/code paths.
   - Look specifically for old "approve download", "auto-accept download", and "auto-deploy staged mods" assumptions.
   - Remove dead backend branches, stale UI labels, stale tests, old helper scripts, and outdated docs that no longer match the immediate-download / approve-install model.
   - Keep the current behavior: Nexus captures download immediately, approval gates local install, newly installed mods default disabled, and Decky owns install/enable automation settings.
   - Latest live state: removed the normal `staged` API status path for healthy installed mods and renamed download recovery response/event counts from `staged` to `installed`. Live `/api/games/413150/mods` now reports installed Nexus mods with `status: "installed"`.
   - Latest review state: route/job scan found no active `/api/imports/*` aliases, `pending-import` job paths, or game-level mod delete route in runtime code. Current runtime route family is `/api/captured-installs/*` plus Action Center/installer-choice handling, and removal from normal UI is profile-scoped.

16. Finish planning-doc cleanup.
   - Align README, ROADMAP, TODO, extension framework docs, extension targets, and testing docs with the current route/event/update-check/rollback terminology.
   - Remove claims that scaffolding equals completed product behavior.

17. Final MVP phone/tablet UX overhaul.
   - Treat the current phone/tablet app as functionally useful but not shippable.
   - Redesign the mobile/tablet web app around the same strong product quality now present in the Decky UI: clear game context, profile-first mod management, readable Action Center, polished installer-choice/FOMOD flow, source pills, update state, rollback/recovery, and advanced deployment tools without overwhelming normal users.
   - iPad/tablet layout must be first-class, not a stretched phone view.
   - Complete this as the final MVP gate after the backend/deployment/FOMOD/provider pipeline is stable enough that the UI is not being redesigned around moving targets.

## MVP Polish After Core Pipeline

1. Verify toast coverage after event architecture lands.
   - User-required validation: leave the final Gaming Mode toast pass for morning manual testing.
   - Current manual testing shows download/install toasts working.
   - Re-check capture, downloaded, install waiting, installing, installed, failed, and launch-action-required notifications after replacing polling.

## Post-MVP

1. Add Windows/Proton Stardew support through extension metadata.
2. Use a clean Fallout 4 reinstall as the first Windows/Proton extension test bed.
3. Build extension manifests for installed game targets listed in `extensionTargets.md`.
4. For each game extension, verify Vortex source behavior first and add missing extension-framework APIs as needed for one-for-one feature parity.
5. Add saved installer-choice presets for unattended FOMOD reinstalls.
6. Harden additional upstream/provider support after the MVP provider architecture ships.
7. Add configurable DMM-owned storage locations, including SD-card-friendly downloads/cache/staging/backups, with safe migration, free-space checks, mount validation, and deployment-manifest repair/verification after moving storage.

## Completed / Removed From Active MVP

- Completed: QAM no longer tries to host the dense DMM workflow; `Open DMM` launches the full Decky route.
- Completed: Fallout/Gamebryo native activation files no longer create false DMM-generated deployment conflicts when there are no DMM-managed plugin entries.
- Completed: Direct Nexus HTTPS file install failures for non-premium accounts now show a browser-required message instead of a false success.
- Completed: `Restore last applied state` and deployment history are exposed through backend endpoints and the web advanced recovery panel.
   - Completed: Nexus mod update checks are implemented through backend cache state, web controls, Decky controls, `mod_updates.changed` events, and live Stardew API validation.
   - Latest live state: a current package checked eight Stardew mods through `POST /api/games/413150/mods/check-updates`; Nexus and GitHub-sourced mods all returned `current` without installing changes.
- Completed: Backend queue plus typed domain events and WebSocket delivery are the active realtime architecture.
- Completed: Nexus captures now download/cache immediately while links are fresh.
- Completed: Project Zomboid is registered as a Workshop-only first-party extension with no fake Nexus domain, so Workshop coexistence/actions can be supported without claiming Nexus archive support.
- Completed: Approval now gates local install from cached archive.
- Completed: Decky Settings now owns `Auto-install captured downloads`.
- Completed: Decky Settings now owns `Auto-enable installed mods`.
- Completed: Newly installed mods default disabled unless auto-enable is explicitly on.
- Completed: Extension-driven SMAPI launch setup was live-tested successfully with Stardew launching through SMAPI and loaded mods.
- Completed: Phone/tablet web game list supports search, favorites pinned at the top, and `Recent`/`A-Z`/`Z-A` sorting.
- Completed: Decky navigation uses tabs with Mods as a first-class surface.
- Completed: Decky plugin starts the Go backend automatically when the plugin loads, so the server comes back after a reboot without opening the panel.
- Completed: Decky Mods auto-selects the running supported game, shows search, and provides controller-focusable rows.
- Completed: Decky and phone/tablet mod enable/disable actions apply enabled mods through the backend.
- Completed: Reset managed mods purges DMM-owned deployed files, removes installed rows and staging folders, clears install candidates, and keeps cached downloads.
- Removed: `Auto-accept download requests` as a product concept.
- Removed: `Auto-deploy staged mods` as a normal-user product concept.
- Removed: Phone/tablet control over Deck-side install automation settings.
