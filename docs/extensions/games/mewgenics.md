# Mewgenics Extension Notes

## Identity

- Steam AppID: `686060`
- DMM extension ID: `mewgenics`
- Nexus domain: `mewgenics`
- Vortex manifest package: Nexus site mod `1691`, file `8709`, version `0.3.2`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Mewgenics Vortex extension package `0.3.2`: `https://www.nexusmods.com/site/mods/1691`
- Vortex package source file: `index.js` from Nexus site mod `1691`, file `8709`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Mewgenics`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the verified Vortex extension package.
- Checks for `Mewgenics.exe` and `resources.gpak`.
- Supports Vortex-backed installers for normal `description.json`/content-folder mods, Mewjector DLL mods, Mewjector, Mewtator, and Mewgenics Save Editor archives.
- Generates DMM-owned `mods/modlist.txt` and root `launch.bat` during deployment from enabled profile mappings, matching Vortex's load-order-driven launch-file model.
- Registers `launch.bat` as the extension primary launch tool for enabled Mewgenics mods.
- Blocks the Vortex fallback installer instead of copying arbitrary unknown root files.

## Beta Gaps

- Live-test representative Nexus Mewgenics archives.
- Live-test Steam Deck launch behavior for generated `launch.bat`. Vortex marks it as a shell launch tool; DMM's current launch-tool API does not yet model a separate shell flag.
- Add a generic launch-tool shell/argument contract if `launch.bat` cannot be executed through the current Steam launch-option bridge.
