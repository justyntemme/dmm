# Mr. Prepper Extension Notes

## Identity

- Steam AppID: `761830`
- DMM extension ID: `mrprepper`
- Nexus domain: `mrprepper`

## Verified Sources

- Nexus API game list verified the `mrprepper` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Vortex has no bundled Mr. Prepper game handler, so DMM uses the source-reviewed shared Vortex `modtype-bepinex` behavior.
- Representative Nexus install instructions identify BepInEx 5 as the required runtime for plugin DLL mods.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM detects the Steam Deck Windows/Proton install through `MrPrepper.exe` and `MrPrepper_Data/globalgamemanagers`.
- DMM supports Vortex-compatible BepInEx runtime packages, BepInEx root/config packages, ConfigurationManager archives, and loose BepInEx plugin DLL archives.
- Unknown archive layouts remain blocked by an extension-owned catch-all until a specific install rule is verified.

## Beta Gaps

- Live-test a Mr. Prepper Nexus BepInEx runtime package and at least one plugin archive on the Steam Deck.
- Add non-BepInEx install rules only after source or representative archive review proves the target roots.
