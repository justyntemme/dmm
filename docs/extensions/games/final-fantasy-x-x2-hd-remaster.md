# Final Fantasy X/X-2 HD Remaster Extension Notes

## Identity

- Steam AppID: `359870`
- DMM extension ID: `finalfantasyxx2hdremaster`
- Nexus domain: `finalfantasyxx2hdremaster`

## Verified Sources

- Nexus API game-domain verification: `https://api.nexusmods.com/v1/games.json`
- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Vortex bundled game extension source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/FINAL FANTASY FFX&FFX-2 HD Remaster`

## Current DMM Capability

- DMM declares the verified Nexus domain so game-scoped Nexus browsing works.
- DMM checks for the launcher, `FFX.exe`, `FFX-2.exe`, and the main VBF archives.
- Archive installs are intentionally blocked because no verified Vortex extension was found and representative layouts have not been classified.

## Beta Gaps

- Representative Nexus archives must be reviewed before install rules are added.
- The extension needs verified rules for VBF patching, loose binary/root replacement, loader-specific folders, or other observed mod patterns.
- Any future VBF mutation must use a generic extension-framework capability with manifest-aware backup/rollback instead of direct game-specific writes in core code.
