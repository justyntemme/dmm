# Command & Conquer: Generals Extension Notes

## Identity

- Steam AppID: `2229870`
- DMM extension ID: `cncgenerals`
- Nexus domain: `cncgenerals`

## Verified Sources

- Nexus API game list verified the `cncgenerals` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Generals handler was found.
- C&C Labs documents the simple `.big` package flow: drop the BIG file in the Generals or Zero Hour game directory.
- Steam community Generals guide treats GenLauncher as a separate external launcher flow.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM supports narrow `.big` archive installs into the game root.
- GenLauncher packages, patchers, loose INI/data replacements, and full conversion launch flows remain blocked until source-reviewed extension rules exist.
- Runtime diagnostics verify `Generals.exe`, `Game.dat`, and `INI.big`.

## Beta Gaps

- Review Generals and Zero Hour interactions before sharing any install rules.
- Encode installer rules in the extension rather than in backend branches.
- Add external launcher/tool support only through a generic extension-framework capability, not a Generals-specific backend path.
