# Metro Exodus Extension Notes

## Identity

- Steam AppID: `1449560`
- DMM extension ID: `metroexodus`
- Nexus domain: `metroexodus`
- Vortex manifest package: Nexus site mod `907`, file `8800`, version `0.2.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Metro Exodus Vortex extension page: `https://www.nexusmods.com/site/mods/907`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Metro Exodus Enhanced Edition`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the executable and VFS archive layout in the Enhanced Edition install.
- Blocks archive installs until the Vortex package and representative Metro archive layouts are inspected.

## Beta Gaps

- Confirm whether the Vortex extension targets classic Metro Exodus, Enhanced Edition, or both.
- Add only verified root/VFS/loose-file installer rules.
