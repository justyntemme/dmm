# Mewgenics Extension Notes

## Identity

- Steam AppID: `686060`
- DMM extension ID: `mewgenics`
- Nexus domain: `mewgenics`
- Vortex manifest package: Nexus site mod `1691`, file `8709`, version `0.3.2`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Mewgenics Vortex extension page: `https://www.nexusmods.com/site/mods/1691`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Mewgenics`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for `Mewgenics.exe` and `resources.gpak`.
- Blocks archive installs because the manifest describes generated launch commands, load-order-driven behavior, and Mewtator/root installs.

## Beta Gaps

- Inspect the Vortex package and representative archives.
- Add extension lifecycle support for generated launch scripts and load-order-derived launch arguments before enabling installs.
