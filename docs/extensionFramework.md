# Extension Framework

This document tracks the extension/runtime boundary for Decky Mod Manager. The first target is a one-for-one functional clone of the Stardew Valley Vortex extension behavior needed for Steam Deck, implemented in DMM's extension framework rather than as generic hardcoded application logic.

## Current Direction

- Extensions are first-party compiled Go packages for MVP.
- Extension specs register through `gameext.Registrar`, using Go structs and registration code rather than JSON/YAML/Lua/Starlark behavior.
- Extension specs describe game identity, Nexus domains, Steam app IDs, supported installer rules, mod types, deployment roots, runtime requirements, launch tools, merge hooks, load-order hooks, and lifecycle event hook declarations.
- Generic backend code executes common pipeline stages, but it does not own game-specific rules.
- The core validates extension outputs before filesystem writes: staging paths, target mappings, generated files, launch-tool files, deploy strategies, unmanaged overwrite risk, and deployment manifest consistency.
- The Decky frontend is a capability adapter for Steam frontend APIs that are unavailable to the Go backend.

## Stardew Valley MVP

- Support Steam Deck-native Linux Stardew for MVP.
- Windows/Proton Stardew is post-MVP, but the eventual implementation must use the same extension-driven runtime/platform model.
- Native Linux Stardew:
  - Game marker: `StardewValley`.
  - SMAPI payload: Linux `install.dat`.
  - SMAPI launch target: `StardewModdingAPI`.
- Post-MVP Windows/Proton Stardew:
  - Game marker: `Stardew Valley.exe` and/or forced Proton compatibility state.
  - SMAPI payload: Windows `install.dat`.
  - SMAPI launch target: `StardewModdingAPI.exe`.
- SMAPI runtime support must come from the Stardew extension's launch-tool metadata, not from generic server branches.
- The Stardew extension declares that enabled SMAPI mod metadata/mod types require `smapi` as the primary launch tool.
- The backend publishes a required launch-tool action when extension rules require SMAPI and the active Steam launch options do not reference the correct SMAPI executable.
- The Decky frontend applies that action through Steam's frontend API, then reports the observed result back to the backend for diagnostics.

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

## Dynamic Launch Inputs

The existing launch-tool contract already covers static launch tools such as SMAPI: an extension declares an executable, arguments, required files, provider mod types, and the mod types that require the tool as primary. Generic backend code evaluates those rules and Decky applies the resulting Steam launch options.

TexMod-style tools should extend that same contract instead of introducing a game-specific path. The likely API shape is a typed optional field on `LaunchToolSpec`, not a stringly `dynamic` mode:

```go
type LaunchToolSpec struct {
    ID                 string
    Name               string
    ExecutableRelative string
    Arguments          []string
    RequiredFiles      []string
    DefaultPrimary     bool
    ModTypes           []string
    ProviderModTypes   []string
    DynamicInputs      []LaunchToolDynamicInputSpec
}

type LaunchToolDynamicInputSpec struct {
    ID              string
    Kind            string // generated-config, enabled-mod-file-list, or another reviewed generic kind.
    SourceModTypes  []string
    OutputRelative  string
    ArgumentToken   string
}
```

The exact struct names can change during implementation, but the boundary should not: extensions declare the dynamic input, the core builds it from the selected profile's enabled mods, validates the generated output and paths, stores it as DMM-owned profile state, and then passes it to the launch tool through normal launch arguments or generated files. Static launch tools leave `DynamicInputs` empty and keep the current SMAPI behavior.

For Prototype/Prototype 2, TexMod support should likely be modeled as:

- A `.tpf` installer/mod type declared by the Prototype extension.
- A TexMod launch tool declared by the Prototype extension.
- A dynamic enabled-mod file list or generated config input containing the active profile's `.tpf` package paths/order.
- Steam launch options pointing at the TexMod/autoload executable or helper, not at a hardcoded core Prototype path.
- DMM-managed generated files and rollback state so profile changes can add/remove active `.tpf` packages cleanly.

## Completed Native Stardew Extension Capabilities

- Stardew lives in `internal/extensions/stardewvalley` and registers through the Go extension registrar.
- Native Linux SMAPI installer archives are detected from Vortex-modeled installer metadata and install the embedded Linux payload.
- SMAPI mods are detected from `manifest.json`, installed under `Mods/`, and expose dependency metadata from the manifest.
- Root-folder `Content/` archives are routed through the root-folder installer path.
- Launch-tool requirements are extension-declared and executed by the Decky Steam API bridge.
- Native Linux Stardew plus SMAPI mods have been manually verified on the Steam Deck.

## FOMOD / Installer Choice Support

- FOMOD support must use the shared install pipeline: download, inspect, pause for choices, persist the installer-choice action, produce install instructions, stage outputs, and then deploy according to profile settings.
- Non-FOMOD extension prompts should use the same installer-choice/action-center pipeline when Vortex source shows that a game extension asks the user for installer input.
- The shared choice schema supports option groups and extension-owned text groups. Storage remains `choices_json` as `map[groupID][]string`; text choices store the entered value as the first string for the group so presets and re-runs use the same path as FOMOD choices.
- Deck-only FOMOD flows should use Decky modals, not the sidebar as the primary option UI.
- Phone/tablet FOMOD flows should use the same backend choice-state API as Decky modals.
- Auto-enable may open a Decky modal for first-time installer choices, but it must not silently choose options.
- Saved installer-choice presets may enable future headless reinstall/update flows only when the FOMOD structure still matches the stored preset.

## Lifecycle Hooks And Notices

- Extensions may register typed lifecycle hooks such as `will-deploy`, `did-deploy`, and `did-purge`.
- `will-deploy` may generate additional deployment mappings; those mappings still flow through core validation, preview, deployment manifests, and rollback.
- `did-deploy` and `did-purge` are observational hooks. They must not write files directly or bypass the deployment manifest.
- Post-event hook messages are promoted to deduped `extension-notice` jobs. This lets game extensions surface Vortex-style manual follow-up work through Action Center, WebSocket updates, and Decky toasts without adding game-specific backend code.
- Current examples: the Ghost Recon Breakpoint extension mirrors Vortex's `did-deploy` AnvilToolkit repack warning, and the Witcher 3 extension mirrors Vortex's Script Merger reminder after managed mod changes. Extension notices can declare a generic `run-launch-tool` action that resolves only against extension-registered launch tools or managed tool providers and is executed by Decky through Steam's tool-launch boundary.

## Extension Boundary Rules For New Games

- Keep the Stardew extension as the only game extension until the Go registration boundary is stable enough that new games do not need rewrites.
- Keep lifecycle-hook behavior generic: game extensions declare hook behavior and messages; the core validates generated mappings and promotes post-event messages to jobs, but it does not contain game-specific hook rules.
- Keep runtime-loaded external `.so` plugins out of MVP unless first-party packaging needs it. First-party extensions are compiled into the DMM binary for now, while preserving a single registration function per extension package.
- Windows/Proton Stardew support remains post-MVP and must add platform detection and payload selection through the same extension API.

## Future Extension Work

- Runtime-loaded external extension bundles.
- Script extender and toolchain launch management for other games.
- Per-game load order engines.
- Provider-specific update checks.
- Vortex extension translation or compatibility tools.
