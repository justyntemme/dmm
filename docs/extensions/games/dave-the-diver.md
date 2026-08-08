# Dave the Diver Extension Notes

## Identity

- Steam AppID: `1868140`
- DMM extension ID: `davethediver`
- Nexus domain: `davethediver`

## Verified Sources

- Nexus API game list verified the `davethediver` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Vortex has no bundled Dave the Diver game handler, so DMM uses the source-reviewed shared Vortex `modtype-bepinex` behavior.
- Representative Nexus install instructions show BepInEx 6 IL2CPP plugin archives targeting `Dave the Diver/BepInEx/plugins`.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM detects the Steam Deck Windows/Proton install through `DaveTheDiver.exe`, `UnityPlayer.dll`, `GameAssembly.dll`, and `DaveTheDiver_Data/globalgamemanagers`.
- DMM supports Vortex-compatible BepInEx IL2CPP runtime packages, BepInEx root/config packages, ConfigurationManager archives, and loose BepInEx plugin DLL archives.
- Unknown archive layouts remain blocked by an extension-owned catch-all until a specific install rule is verified.

## Beta Gaps

- Live-test a Dave the Diver Nexus BepInEx IL2CPP runtime package and at least one plugin archive on the Steam Deck.
- Add non-BepInEx install rules only after source or representative archive review proves the target roots.
