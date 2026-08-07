# Total War: ROME II Extension Notes

## Identity

- Steam AppID: `214950`
- DMM extension ID: `totalwarrome2`
- Nexus domain: `totalwarrome2`

## Verified Sources

- Nexus game page/domain: `https://www.nexusmods.com/totalwarrome2`
- Vortex bundled game extensions: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Steam appdetails category check: `https://store.steampowered.com/api/appdetails?appids=214950&filters=categories`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Total War Rome II`

## Current DMM Capability

- DMM declares the verified Nexus domain so game-scoped Nexus browsing works.
- DMM checks for `Rome2.exe`, `data/manifest.txt`, and `data/data_rome2.pack`.
- Archive installs are intentionally blocked because no verified Vortex extension was found and representative Nexus layouts have not been classified.
- Steam Workshop is not advertised for this game because Steam appdetails does not declare the Workshop category and the Deck has no local `214950` Workshop manifest/content.

## Beta Gaps

- Representative Nexus archives must be reviewed before install rules are added.
- The extension needs verified rules for pack-file placement, launcher-managed mod activation, data-folder loose files, and any external Total War mod-manager workflow.
- If support requires patching or generating Total War launcher/load-order state, DMM needs a generic extension-framework capability with preview, rollback, and manifest ownership before enabling installs.
