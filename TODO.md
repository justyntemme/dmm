# TODO

## Current MVP Fixes

1. Change Nexus approval from "approve download" to "download now, approve install."
   - Nexus automatic download links can be short-lived, so DMM should consume the captured `nxm://` request immediately while it is fresh.
   - The backend should download/cache the archive, inspect it, and create a local install request from durable local state.
   - User approval should gate install/stage/profile/deploy decisions, not the network download itself.
   - This install approval UX should hide staging/install-plan details by default and present the user-centered outcome: ready to install, needs choices, blocked, installed disabled, or failed.
   - If `Auto-accept download requests` remains the setting label, its implementation should map to auto-accepting install requests after the archive is cached; consider renaming the user-facing label during UI redesign.
2. Make the existing Decky Mods rows controller-scrollable.
   - Mod entries must be focusable/selectable with controller navigation, especially D-pad up/down.
   - Focused mod rows should visibly highlight and allow scrolling through the list without requiring pointer clicks.
   - The list must remain usable with long mod names and many entries in the Decky sidebar.
3. Add Decky Mods search/filter.
   - Add a compact search/filter field for long mod lists.
   - Keep it controller-friendly and visually quiet.
4. Add Deck-owned fast install settings for Deck-only use.
   - Add `Auto-accept download requests` as a Decky Settings checkbox, or rename it to match the new install-approval model if the UI redesign lands first.
   - Add `Auto-deploy staged mods` as a Decky Settings checkbox.
   - Both settings should be enabled by default for MVP so a Deck-only user can click Nexus Mod Manager Download and have DMM capture, download, inspect, stage, and deploy without opening the phone/tablet UI.
   - Auto-deploy must not auto-enable the mod. Newly installed mods stay disabled until the user explicitly enables them in the current profile.
   - If a mod needs FOMOD/installer choices or is blocked by install planning, auto-deploy must pause and surface an actionable Decky/phone request instead of guessing.
   - These controls belong in the Decky plugin settings, not the phone/tablet web app.
5. Polish the existing Decky `Mods` UI into a first-class tab-level surface where possible.
   - The current Mods sidebar view looks good visually, so keep its direction and improve interaction rather than replacing it.
   - Convert Mods access from a click-through button into a tab-level surface where possible, to preserve sidebar space and make it feel like a first-class Decky workflow.
   - Each mod row should keep a controller-friendly enable/disable toggle.
   - Enabling/disabling from Decky should update the profile and apply the profile changes automatically.
   - The row should communicate when a game restart is required for changes to affect a running game.
   - Keep the UI compact and polished for the Decky sidebar: restrained text, clear status chips, and no staging/file-operation noise in the primary view.
6. Verify Decky toast notifications only if logs show missing captured/download/install transition toasts.
   - Manual testing now shows download toast notifications working again.
   - Remove this as an active blocker unless logs show that specific toast classes are still failing.
7. Add mobile web game favorites and sorting.
   - Users can favorite games so they appear at the top of the game drawer/list.
   - Add a sort dropdown with `Recent` as the default, plus `A-Z` and `Z-A`.
8. Replace aggressive polling with the durable live-event architecture.
   - Backend queue owns work, retry, cancel, and crash recovery.
   - Typed domain events are emitted for install requests, jobs, profile changes, deployment status, runtime requirements, and launch actions.
   - WebSocket is the realtime transport for phone/tablet UI and Decky modal/toast flows.
   - Short polling can remain only as a temporary fallback while the event bus lands.
9. Verify mobile web state updates after the newest staged package is installed:
   - Install requests update immediately after approval/cancel/retry.
   - Approved requests cannot be approved twice while the job is in flight.
   - Job events, install completion, profile changes, deployment status, and runtime/launch-tool actions update the current screen automatically.
   - Discuss the event-driven system before implementation; short polling is acceptable only as a temporary fallback.
   - Current package forces an immediate job/game refresh after approve/cancel/retry and keeps SSE/short polling as a fallback.
10. Change Nexus install behavior so newly downloaded/staged mods are disabled by default. Completed locally; verify on the Deck.
11. Hide the normal "apply profile changes" workflow behind simple mod enable/disable actions. Mostly complete locally:
   - Enabling a mod saves it to the current profile and applies the profile automatically.
   - Disabling a mod saves it to the current profile and applies the profile automatically.
   - Keep staging, preview, purge, repair, and manual apply controls in an advanced/power-user area.
   - Current package adds a clear primary "Apply Changes" action if automatic apply does not finish cleanly.
12. Investigate the "remove 24" apply result and make the primary UI communicate deployment changes in user-centered terms.
13. Verify extension-driven SMAPI launch-option configuration after the newest staged package is installed:
   - Backend should publish the required launch action when enabled Stardew/SMAPI mods require it.
   - Decky should apply the Steam launch option through the Steam client API.
   - Diagnostics should show Steam is configured to launch Stardew through SMAPI after the action succeeds.
   - Direct Steam `localconfig.vdf` mutation is removed from product flow; `/launch/apply` is a request/status endpoint for the Decky Steam API bridge.
   - SMAPI launch-root files must deploy as copies, not symlinks, so .NET resolves Stardew's bundled runtime from the game folder.
14. Package and SCP a new test build when these fixes are ready.
   - Completed: `/home/deck/.testing/decky-mod-manager.tar.gz` has been updated and passes package shape smoke locally and on the Deck.

## MVP Polish

1. Polish the profile-first mod UI:
   - Ordinary users should see mods in a profile with enable/disable state and clear next actions.
   - Staging, preview, purge, repair, blocked install plans, and file-level details should be advanced controls.
2. Implement FOMOD / installer-choice support:
   - Detect installer-choice archives.
   - Pause after download/archive inspection.
   - Persist the installer-choice request.
   - Show choices in the phone/tablet UI.
   - Support a Decky modal flow for no-phone installs where possible.
3. Last MVP polish item: add game list favorites and sorting.
   - Users can favorite games so they appear at the top of the game drawer/list.
   - Add a sort dropdown with `A-Z`, `Z-A`, and `Recent`.
   - Keep this below deployment/FOMOD/live-validation work; it improves navigation but does not block the core mod install pipeline.

## Decky Mod Management

1. MVP: Promote the existing Decky Mods view into a tab-level surface where possible.
2. MVP: Make installed mod rows focusable with D-pad/controller navigation.
3. MVP: Ensure focused rows highlight and scroll correctly without pointer clicks.
4. MVP: Add a compact mod search/filter field.
5. MVP: Show installed mods for the current game/profile in Decky without staging terminology.
6. MVP: Add/keep enable/disable toggles for mods in Decky.
7. MVP: Add a compact profile selector in Decky for the current/running game.
8. MVP: Note clearly that game restart is required for profile/mod changes to affect a running game.
9. MVP: Convert Decky navigation away from cramped dropdown-heavy controls where tabs provide more usable sidebar space.

## Post-MVP After Current MVP Completion

1. Add Windows/Proton Stardew support through extension metadata, not generic code branches.
2. Build extension manifests for installed game targets from `extensionTargets.md`.
3. For each game extension, verify Vortex source behavior first and add missing extension-framework APIs as needed for one-for-one feature parity.

## Latest Launch Tool Feedback

- Runtime warnings for missing primary launch tools should include a retry/request action.
- The action must be generic: extensions declare primary launch tool requirements; backend exposes required launch actions; Decky applies Steam launch options through Steam APIs.
- Stardew/SMAPI should only be one extension instance of this mechanism, not a generic app branch.
- If Decky was offline or not rendering when the action was first needed, the action should remain discoverable and retryable from the phone/tablet warning and the Decky plugin.
