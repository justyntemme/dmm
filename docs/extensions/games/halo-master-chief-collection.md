# Halo: The Master Chief Collection Extension Notes

## Identity

- Steam AppID: `976730`
- DMM extension ID: `halothemasterchiefcollection`
- Nexus domain: `halothemasterchiefcollection`

## Verified Sources

- Vortex Halo MCC game extension: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-masterchiefcollection/src`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Plug-and-play `modinfo.json`, `modpack_config.cfg`, and recognized Halo game-folder installers are represented.
- Assembly tool metadata is registered.
- Game version discovery reads `build_tag.txt`.
- Extension deploy hook writes managed `ModManifest.txt` entries through DMM deployment.

## Beta Gaps

- Live Nexus archive validation is required.
- Launcher expectations and anti-cheat-safe launch flow need UX review.
- `ModManifest.txt` interaction with non-DMM mods needs validation.

## Validation Targets

- Plug-and-play mod with `modinfo.json`.
- Modpack config archive.
- Recognized Halo game-folder archive.
