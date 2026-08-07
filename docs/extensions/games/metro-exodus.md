# Metro Exodus Extension Notes

## Identity

- Steam AppID: `1449560`
- DMM extension ID: `metroexodus`
- Nexus domain: `metroexodus`
- Vortex manifest package: Nexus site mod `907`, file `8800`, version `0.2.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Metro Exodus Vortex extension page: `https://www.nexusmods.com/site/mods/907`
- Verified Vortex extension package file: `https://www.nexusmods.com/site/mods/907?tab=files&file_id=8800`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Metro Exodus Enhanced Edition`

## Current DMM Capability

- Registers Enhanced Edition Steam AppID `1449560`, legacy Steam AppID `412020`, and Nexus domain `metroexodus`.
- Mirrors the verified Vortex package's basic-game setup:
  - executable: `MetroExodus.exe`
  - mod path: game root `.`
  - mod merging enabled
  - required files: `MetroExodus.exe`
  - supported tool: `SDK/bin_x64/Exodus_SDK.exe`
- Provides a root archive installer that strips a single common archive wrapper before staging files into the game root.
- Publishes Vortex-equivalent readme/changelog conflict ignores and deploy ignores:
  - `**/changelog*`
  - `**/readme*`

## Beta Gaps / Live Validation

- Live-test a safe Metro Exodus mod from Nexus through the BrowserView `nxm://` flow.
- Confirm whether legacy Steam AppID `412020` installs into a different folder shape on Deck/Proton before promoting it as a manual target.
