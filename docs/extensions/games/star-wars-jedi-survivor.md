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
- Multi-PAK archives now pause as DMM installer-choice actions. The extension emits a Vortex-style "choose PAK" wizard, and the selected PAK plus matching sidecar files are staged through the normal profile pipeline.
- R457 loader packages containing `zR457ModLoader.pak` are represented as game-root relative installs.
- Unreal pak load-order prefix generation is registered through the shared DMM Unreal helper.

## Beta Gaps

- Live Nexus archive validation is required.
- Multi-PAK installer-choice UI needs live Decky and phone/tablet validation with a real archive.
- Load-order UI and prefix behavior need live validation with multiple installed pak mods.
- R457 loader package behavior needs live validation against the real Nexus archive.

## Validation Targets

- Simple single-PAK Nexus archive.
- Multi-PAK archive to confirm the installer-choice prompt and selected-file staging.
- R457 loader archive.
- Two pak mods requiring order changes.
