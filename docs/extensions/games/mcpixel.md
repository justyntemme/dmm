# McPixel

- Steam AppID: `220860`
- DMM extension ID: `mcpixel`
- Coverage: metadata only

## Verified Sources

- ModDB game mods page: `https://www.moddb.com/games/mcpixel/mods`
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Bundled Vortex game extension source was checked; no reviewed McPixel handler was found.

## DMM Behavior

- DMM declares the installed game and verified source references.
- DMM does not declare a Nexus domain, Steam Workshop actions, or archive installers for McPixel.

## Remaining Work

- Do not add install support until ModDB automation or representative archive behavior is verified.
- If support is added later, keep all McPixel-specific roots and install rules inside this extension.
