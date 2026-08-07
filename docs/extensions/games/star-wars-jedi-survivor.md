# Star Wars Jedi: Survivor Extension Notes

## Identity

- Steam AppID: `1774580`
- DMM extension ID: `starwarsjedisurvivor`
- Nexus domain: `starwarsjedisurvivor`

## Verified Sources

- Vortex Star Wars Jedi: Survivor extension source: `https://github.com/Pickysaurus/vortex-jedi-survivor`
- Live Steam Deck install path verification:
  - Executable: `SwGame/Binaries/Win64/JediSurvivor.exe`
  - Pak folder: `SwGame/Content/Paks`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Copy-only deployment is the default strategy because the Vortex extension declares `supportsSymlinks: false`.
- Single `.pak` archive planning is implemented and deploys files from the matched pak folder into `SwGame/Content/Paks/~mods`.
- R457 loader packages containing `zR457ModLoader.pak` are represented as game-root relative installs.
- Unreal pak load-order prefix generation is registered through the shared DMM Unreal helper.

## Beta Gaps

- Multi-PAK archives are blocked until DMM has a generic installer-choice UI for Vortex-style PAK selection.
- Live Nexus archive validation is required.
- Load-order UI and prefix behavior need live validation with multiple installed pak mods.
- R457 loader package behavior needs live validation against the real Nexus archive.

## Validation Targets

- Simple single-PAK Nexus archive.
- Multi-PAK archive to confirm the unsupported choice message.
- R457 loader archive.
- Two pak mods requiring order changes.
