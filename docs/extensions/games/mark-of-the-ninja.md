# Mark of the Ninja

- Steam AppID: `214560`
- DMM extension ID: `markoftheninja`
- Coverage: metadata only

## Verified Sources

- ModDB game mods page: `https://www.moddb.com/games/mark-of-the-ninja/mods`
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Bundled Vortex game extension source was checked; no reviewed Mark of the Ninja handler was found.

## DMM Behavior

- DMM declares the installed game and verified source references.
- DMM does not use the Nexus `markoftheninjaremastered` domain for this Steam app because the installed app is the original Mark of the Ninja, not the Remastered release.
- DMM does not declare a Nexus domain, Steam Workshop actions, or archive installers for this extension.

## Remaining Work

- Do not add install support until ModDB automation or representative archive behavior is verified.
- Treat Remastered support as a separate extension if the Remastered Steam app is installed later.
