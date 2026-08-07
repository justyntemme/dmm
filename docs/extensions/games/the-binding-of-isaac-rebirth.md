# The Binding of Isaac: Rebirth Extension Notes

## Identity

- Steam AppID: `250900`
- DMM extension ID: `thebindingofisaacrebirth`
- Nexus domain: `thebindingofisaacrebirth`
- Vortex manifest package: Nexus site mod `516`, file `4127`, version `1.0.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- The Binding of Isaac Vortex extension page: `https://www.nexusmods.com/site/mods/516`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/The Binding of Isaac Rebirth`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the executable and resources folders.
- Blocks archive installs until Afterbirth+/Repentance mod-folder and resources rules are verified.

## Beta Gaps

- Inspect the Vortex package and representative archives.
- Add extension-owned mod-folder/resources installers and any load-order/profile semantics needed for Isaac mods.
