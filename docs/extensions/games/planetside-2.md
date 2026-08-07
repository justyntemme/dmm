# PlanetSide 2

- Steam AppID: `218230`
- DMM extension ID: `planetside2`
- Coverage: metadata only

## Verified Sources

- ModDB game addons page: `https://www.moddb.com/games/planetside-2/addons`
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Bundled Vortex game extension source was checked; no reviewed PlanetSide 2 handler was found.

## DMM Behavior

- DMM declares the installed game and verified source references.
- DMM does not declare a Nexus domain, Steam Workshop actions, or archive installers for PlanetSide 2.
- DMM must not add file mutation support for this online game without explicit source review of anti-cheat and allowed customization boundaries.

## Remaining Work

- Keep this metadata-only until an official or clearly safe mod/addon management path is verified.
- If support is added later, keep all PlanetSide 2-specific roots, risk warnings, and install rules inside this extension.
