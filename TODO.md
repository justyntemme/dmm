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
   - Routine polling should not be part of the normal UX; one-shot startup snapshots and reconnect recovery fetches are acceptable.
   - Do this before major Decky/mobile UI work so those surfaces are built once on the final realtime state model.

3. Fix phone/tablet UI refresh behavior under the event model.
   - Install requests should update immediately after capture, install approval, cancel, retry, failure, and completion.
   - Approval buttons must become disabled/in-flight immediately so the user cannot approve the same request twice.
   - Profile mod lists, deployment status, jobs, runtime requirements, and launch actions should update without manual browser refresh.

4. Implement MVP FOMOD / installer-choice support.
   - Download/cache happens immediately as normal.
   - Installer-choice requests must persist from local cached archive state.
   - Phone/tablet UI must show touch-friendly choices and apply selected files.
   - Decky modal flow is required for no-phone installs where Decky can safely present the modal.
   - Auto-install/auto-enable must pause for FOMOD unless a compatible saved preset exists.

5. Redesign the phone/tablet mod-management UI around the profile-first model.
    - The primary surface should be the selected profile's enabled/disabled mod list.
    - Install requests should read as local install approvals, not network download approvals.
    - Staging, deploy preview, purge, repair, conflicts, and file-level operations should be advanced/power-user views unless they require immediate action.

6. Improve deployment language.
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
- Completed: Phone/tablet web game list supports search, favorites pinned at the top, and `Recent`/`A-Z`/`Z-A` sorting.
- Completed: Decky navigation uses tabs with Mods as a first-class surface.
- Completed: Decky Mods auto-selects the running supported game, shows search, and provides controller-focusable rows.
- Completed: Decky and phone/tablet mod enable/disable actions apply profile changes through the backend.
- Completed: Reset managed mods purges DMM-owned deployed files, removes installed rows and staging folders, clears install candidates, and keeps cached downloads.
- Removed: `Auto-accept download requests` as a product concept.
- Removed: `Auto-deploy staged mods` as a normal-user product concept.
- Removed: Phone/tablet control over Deck-side install automation settings.
