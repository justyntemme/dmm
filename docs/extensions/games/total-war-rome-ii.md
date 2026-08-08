# Total War: ROME II Extension Notes

## Identity

- Steam AppID: `214950`
- DMM extension ID: `totalwarrome2`
- Nexus domain: `totalwarrome2`

## Verified Sources

- Nexus game page/domain: `https://www.nexusmods.com/totalwarrome2`
- Vortex Total War: Three Kingdoms pack installer source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games/game-totalwarthreekingdoms/src/index.js`
- Total War official Workshop pack-location note: `https://wiki.totalwar.com/w/Steam_Workshop_and_How_to_Make_Mods.html`
- Total War ROME II user.script loading note: `https://www.totalwar.com/news/improving-game-and-mod-interaction-with-desert-kingdoms`
- Steam appdetails category check: `https://store.steampowered.com/api/appdetails?appids=214950&filters=categories`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Total War Rome II`

## Current DMM Capability

- DMM declares the verified Nexus domain so game-scoped Nexus browsing works.
- DMM checks for `Rome2.exe`, `data/manifest.txt`, and `data/data_rome2.pack`.
- DMM supports `.pack` archives intended for the game `data` folder, including same-root `.png` and `.txt` sidecars.
- DMM emits a post-deploy notice because movie-format packs may load automatically, while mod/release packs may still need Rome II launcher or `user.script` activation.
- Steam Workshop is not advertised for this game because Steam appdetails does not declare the Workshop category and the Deck has no local `214950` Workshop manifest/content.

## Beta Gaps

- Live-test a safe `.pack` archive and verify whether it is movie-format auto-load or launcher/user-script activated.
- The extension still needs verified rules for launcher-managed mod activation, data-folder loose files, and any external Total War mod-manager workflow.
- If support requires patching or generating Total War launcher/load-order state, DMM needs a generic extension-framework capability with preview, rollback, and manifest ownership before enabling installs.
