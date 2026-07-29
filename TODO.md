# TODO

## Current MVP Priorities

1. Code-review and remove deprecated UX/code paths.
   - Look specifically for old "approve download", "auto-accept download", and "auto-deploy staged mods" assumptions.
   - Remove dead backend branches, stale UI labels, stale tests, old helper scripts, and outdated docs that no longer match the immediate-download / approve-install model.
   - Keep the current behavior: Nexus captures download immediately, approval gates local install, newly installed mods default disabled, and Decky owns install/enable automation settings.

2. Replace aggressive polling with durable live-event architecture.
   - Backend queue owns work, retry, cancel, crash recovery, and durable state transitions.
   - Typed domain events should cover install requests, jobs, profile changes, deployment status, runtime requirements, launch actions, and running-game changes.
   - WebSocket should be the realtime transport for phone/tablet UI and Decky modal/toast flows.
   - Short polling can remain only as a temporary fallback.
   - Do this before major Decky/mobile UI work so those surfaces are built once on the final realtime state model.

3. Fix phone/tablet UI refresh behavior under the event model.
   - Install requests should update immediately after capture, install approval, cancel, retry, failure, and completion.
   - Approval buttons must become disabled/in-flight immediately so the user cannot approve the same request twice.
   - Profile mod lists, deployment status, jobs, runtime requirements, and launch actions should update without manual browser refresh.

4. Convert Decky navigation to tabs and make Mods rows controller-scrollable.
   - Convert cramped Decky click-through/dropdown-heavy navigation into tabs where Decky supports it cleanly.
   - The Mods tab should be a first-class surface, not a nested button view.
   - When a game is running, Decky should auto-select that game for the Mods tab, similar to the Lossless Scaling/frame-generation plugin pattern: show the running game's icon/name at the top and list profile mods below.
   - If no supported game is running, show a compact game selector/fallback state.
   - Mod entries must be focusable/selectable with controller navigation, especially D-pad up/down.
   - Focused mod rows should visibly highlight and scroll without requiring pointer clicks.
   - Long mod names and large mod lists must remain usable in the Decky sidebar.

5. Add phone/tablet web game favorites and sorting.
   - The web game list already has search; this task is ordering and prioritization.
   - Users can favorite games so they remain pinned at the top of the game drawer/list.
   - Add sort options: `Recent` default, `A-Z`, and `Z-A`.
   - Track/select "recent" from the app's own game selection/use history, not only Steam install order.

6. Ensure Decky mod enable/disable applies profile changes automatically.
   - Toggling a mod should update the selected profile and apply DMM-owned deployment changes without exposing manual "apply profile" steps.
   - If automatic apply fails, show a clear error and a retry/apply action.
   - The UI must explain that a running game may require restart before changes take effect.

7. Verify and fix stale purge/reset cleanup.
   - Removing all mods or purging/resetting should reconcile the active deployment manifest and remove only DMM-owned deployed artifacts.
   - No stale installed-mod rows, profile rows, deployment manifests, or orphaned staging paths should remain after a full reset path.

8. Implement MVP FOMOD / installer-choice support.
   - Download/cache happens immediately as normal.
   - Installer-choice requests must persist from local cached archive state.
   - Phone/tablet UI must show touch-friendly choices and apply selected files.
   - Decky modal flow is required for no-phone installs where Decky can safely present the modal.
   - Auto-install/auto-enable must pause for FOMOD unless a compatible saved preset exists.

9. Redesign the phone/tablet mod-management UI around the profile-first model.
    - The primary surface should be the selected profile's enabled/disabled mod list.
    - Install requests should read as local install approvals, not network download approvals.
    - Staging, deploy preview, purge, repair, conflicts, and file-level operations should be advanced/power-user views unless they require immediate action.

10. Improve deployment language.
   - Replace file-operation-heavy copy with user-centered state: installed, disabled, enabled, applied, needs restart, blocked, conflicts, failed.
   - Keep exact file operations visible in advanced details and logs.

## MVP Polish After Core Pipeline

1. Verify toast coverage after event architecture lands.
   - Current manual testing shows download/install toasts working.
   - Re-check capture, downloaded, install waiting, installing, installed, failed, and launch-action-required notifications after replacing polling.

## Post-MVP

1. Add Windows/Proton Stardew support through extension metadata.
2. Build extension manifests for installed game targets listed in `extensionTargets.md`.
3. For each game extension, verify Vortex source behavior first and add missing extension-framework APIs as needed for one-for-one feature parity.
4. Add saved installer-choice presets for unattended FOMOD reinstalls.
5. Add more upstream/provider support after the Nexus MVP path is stable.

## Completed / Removed From Active MVP

- Completed: Nexus captures now download/cache immediately while links are fresh.
- Completed: Approval now gates local install from cached archive.
- Completed: Decky Settings now owns `Auto-install captured downloads`.
- Completed: Decky Settings now owns `Auto-enable installed mods`.
- Completed: Newly installed mods default disabled unless auto-enable is explicitly on.
- Completed: Extension-driven SMAPI launch setup was live-tested successfully with Stardew launching through SMAPI and loaded mods.
- Removed: `Auto-accept download requests` as a product concept.
- Removed: `Auto-deploy staged mods` as a normal-user product concept.
- Removed: Phone/tablet control over Deck-side install automation settings.
