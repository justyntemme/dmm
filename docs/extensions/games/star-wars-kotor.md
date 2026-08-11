# Star Wars: Knights of the Old Republic

## Source Review

- Vortex game extension source: `https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sw-kotor/src`

## DMM Runtime Support

- DMM registers first-party extensions for KOTOR (`32370`) and KOTOR II (`208580`).
- Game-root archives install to recognized KOTOR root folders such as `modules`, `override`, `rims`, `streammusic`, and related game data folders.
- Default archives install to `override`, matching Vortex's query mod path behavior.
- TSLPatcher utility and patcher-mod archives are no longer unsupported placeholders. DMM stages them under `DMM/TSLPatcher`, declares a `TSLPatcher` launch tool, checks for the staged executable as a runtime requirement, and queues a post-deploy action to run the patcher explicitly.
- The TSLPatcher action remains visible because patchers mutate real game files outside DMM's normal symlink deployment model.

## Remaining Validation

- Live Proton execution of `TSLPatcher.exe` from the Decky launch-tool bridge needs device testing with representative KOTOR mods.
- If a patcher requires non-default working directory behavior, add it as a generic launch-tool working-directory capability and keep the KOTOR-specific declaration in this extension.
