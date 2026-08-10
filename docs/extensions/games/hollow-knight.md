# Hollow Knight Extension Notes

## Identity

- Steam AppID: `367520`
- DMM extension ID: `hollowknight`
- Nexus domain: `hollowknight`
- Vortex manifest package: Nexus site mod `376`, file `7365`, version `2.1.1`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Hollow Knight Vortex extension page: `https://www.nexusmods.com/site/mods/376`
- Hollow Knight Vortex extension package v2.1.1, Nexus site mod `376`, file `7365`, downloaded through the configured Nexus API key for source review.
- Vortex shared `modtype-bepinex` source: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/Hollow Knight`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the executable, Unity data folder, and managed assembly folder.
- Implements source-backed Vortex installer behavior for:
  - `hollow_knight_Data` root-folder archives.
  - BepInEx runtime packages.
  - BepInEx root folders (`plugins`, `config`, `patchers`).
  - BepInEx plugin DLL archives.
  - BepInEx Configuration Manager archives.
  - `Assembly-CSharp.dll` replacements under `hollow_knight_Data/Managed`.
  - Unity `.assets`, `.resource`, and `.ress` files under `hollow_knight_Data`.
- Blocks the Vortex fallback installer rather than writing unknown files to the game root without a specific extension-owned rule.
- Reports a BepInEx runtime requirement when BepInEx mods are enabled.
- Declares Vortex's `autoDownloadBepInEx: true`, `forceGithubDownload: true`, and pinned `5.4.23.5` behavior through DMM's runtime acquisition pipeline using the source-verified `BepInEx_win_x64_5.4.23.5.zip` GitHub release asset.

## Beta Gaps

- Live archive validation with representative Nexus Hollow Knight mods.
- The Vortex fallback notification flow is intentionally blocked in DMM until the UI can present a clear manual-review path.
