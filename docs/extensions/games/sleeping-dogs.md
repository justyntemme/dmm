# Sleeping Dogs

- Steam AppID: `202170`
- DMM extension ID: `sleepingdogs`
- Coverage: metadata only

## Verified Sources

- ModDB game mods page: `https://www.moddb.com/games/sleeping-dogs/mods`
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Bundled Vortex game extension source was checked; no reviewed Sleeping Dogs handler was found.

## DMM Behavior

- DMM declares the installed game and verified source references.
- DMM does not declare a Nexus domain, Steam Workshop actions, or archive installers for Sleeping Dogs.
- Sleeping Dogs: Definitive Edition has a separate Nexus domain and should not be mapped to Steam AppID `202170` without verifying compatibility with the original Steam install.

## Remaining Work

- Do not add install support until ModDB automation or representative archive behavior is verified.
- If support is added later, keep all Sleeping Dogs-specific roots and install rules inside this extension.
