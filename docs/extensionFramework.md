# Extension Framework

This document tracks the extension/runtime boundary for Decky Mod Manager. The first target is a one-for-one functional clone of the Stardew Valley Vortex extension behavior needed for Steam Deck, implemented in DMM's extension framework rather than as generic hardcoded application logic.

## Current Direction

- Extensions are compiled Go packages for MVP.
- Extension specs describe game identity, Nexus domains, Steam app IDs, supported installer rules, mod types, deployment roots, runtime requirements, and launch tools.
- Generic backend code executes common pipeline stages, but it does not own game-specific rules.
- The Decky frontend is a capability adapter for Steam frontend APIs that are unavailable to the Go backend.

## Stardew Valley MVP

- Support Steam Deck-native Linux Stardew for MVP.
- Windows/Proton Stardew is post-MVP, but the eventual implementation must use the same extension-driven runtime/platform model.
- Detect runtime platform from the actual Steam install and compatibility state, not from the host OS alone.
- Native Linux Stardew:
  - Game marker: `StardewValley`.
  - SMAPI payload: Linux `install.dat`.
  - SMAPI launch target: `StardewModdingAPI`.
- Post-MVP Windows/Proton Stardew:
  - Game marker: `Stardew Valley.exe` and/or forced Proton compatibility state.
  - SMAPI payload: Windows `install.dat`.
  - SMAPI launch target: `StardewModdingAPI.exe`.
- SMAPI runtime support must come from the Stardew extension's launch-tool metadata, not from generic server branches.
- The Stardew extension should declare that enabled SMAPI mod metadata/mod types require `smapi` as the primary launch tool.
- The backend should publish a required launch-tool action when extension rules require SMAPI and the active Steam launch options do not reference the correct SMAPI executable.
- The Decky frontend should apply that action through Steam's frontend API, then report the observed result back to the backend for diagnostics.

## Decky Capability Contract

Backend responsibilities:

- Determine when a game/profile requires a launch tool.
- Evaluate extension-declared launch-tool rules against enabled profile mods and staged metadata.
- Produce a desired launch action with app ID, extension ID, tool ID, executable path, desired launch options, current diagnostics, and user-facing explanation.
- Persist the action result and logs.
- Re-read game/Steam state after the action to verify it.
- Honor extension-declared per-file deployment strategy for launch/runtime files, such as copying launcher-root files that cannot safely resolve through staging symlinks.

Decky frontend responsibilities:

- Present the action in a Decky settings/debug/runtime view.
- Run a plugin-level background monitor while Decky Loader has DMM loaded; this monitor is started outside the React panel component, so it continues when the Decky sidebar panel is closed.
- Read current launch options through Steam frontend state when available.
- Apply launch options through `SteamClient.Apps.SetAppLaunchOptions`.
- Report success, failure, and observed launch options to the backend.

The Decky panel does not need to be open for launch actions to run, but the DMM Decky plugin must remain loaded and the Go backend must be running. If Decky Loader is stopped or the plugin is unloaded, the Steam frontend API bridge is unavailable and pending actions remain queued for retry.

Out of scope for the Decky frontend:

- Archive parsing.
- Nexus provider logic.
- Installer planning.
- Deployment planning.
- Profile state mutation beyond calling backend APIs.
- Game-specific launch rules.

## Source Alignment

- Vortex Stardew registers SMAPI as a supported/default primary tool.
- Vortex Stardew chooses `StardewModdingAPI.exe` on Windows and `StardewModdingAPI` on Linux/macOS.
- Vortex SMAPI installer support uses platform-specific payload metadata.
- Decky launch option changes must use verified Steam frontend APIs; direct Steam config edits are not a normal runtime path.

## Remaining Stardew Extension Steps

1. Add runtime platform detection to the extension/game discovery path.
2. Select the SMAPI installer payload by detected game runtime platform.
3. Persist platform-specific launch-tool metadata in staged manifests or diagnostics.
4. Add extension-declared primary launch-tool rules for SMAPI-required mods.
5. Add backend runtime action endpoints for launch-tool configuration.
6. Add Decky frontend action execution using Steam client launch-option APIs.
7. Verify native Linux Stardew end to end for MVP sign-off.
8. Implement and verify Windows/Proton Stardew end to end after MVP.

## Next MVP Feature After Stardew Extension Parity

- Implement FOMOD/installer-choice support as an MVP-required feature after the Stardew Valley extension reaches parity with the required Vortex extension behavior.
- FOMOD support must use the shared install pipeline: download, inspect, pause for choices, persist the choice request, produce install instructions, stage outputs, and then deploy according to profile settings.
- Deck-only FOMOD flows should use Decky modals, not the sidebar as the primary option UI.
- Phone/tablet FOMOD flows should use the same backend choice-state API as Decky modals.
- Auto-enable may open a Decky modal for first-time installer choices, but it must not silently choose options.
- Saved installer-choice presets may enable future headless reinstall/update flows only when the FOMOD structure still matches the stored preset.

## Future Extension Work

- External extension loading.
- Script extender and toolchain launch management for other games.
- Per-game load order engines.
- Provider-specific update checks.
- Vortex extension translation or compatibility shims.
