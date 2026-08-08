# Final Fantasy X/X-2 HD Remaster Extension Notes

## Identity

- Steam AppID: `359870`
- DMM extension ID: `finalfantasyxx2hdremaster`
- Nexus domain: `finalfantasyxx2hdremaster`

## Verified Sources

- Final Fantasy X/X-2 External File Loader Nexus page: `https://www.nexusmods.com/finalfantasyxx2hdremaster/mods/150`
- External File Loader source repository: `https://gitlab.com/ffgriever/ffx-x-2-hd-external-file-loader`
- Representative External File Loader content mod: `https://www.nexusmods.com/finalfantasyxx2hdremaster/mods/244`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/FINAL FANTASY FFX&FFX-2 HD Remaster`

## Current DMM Capability

- DMM declares the verified Nexus domain so game-scoped Nexus browsing works.
- DMM checks for the launcher, `FFX.exe`, `FFX-2.exe`, and the main VBF archives.
- DMM supports ffgriever's External File Loader root package when the archive contains `dinput8.dll`, `hook.ini`, and `modules/...`.
- DMM supports content archives rooted at `data/mods`, `ffx_data`, or `ffx2_data`, deploying content under `data/mods`.
- DMM generates `modules/config/ff10-file-loader.ini` during deploy when External File Loader content mods are enabled. The generated config keeps DMM's `data/mods` path first, preserves existing user paths after it, and preserves non-`[Paths]` sections such as `[General]` and `[Logging]`.
- Runtime diagnostics warn when External File Loader is missing before enabling external-file content mods.

## Beta Gaps

- Live-test the External File Loader package and a safe `ffx_data`/`ffx2_data` content mod.
- Add profile-aware per-mod path isolation and load-order editing for External File Loader only if live testing proves the single DMM-managed `data/mods` path is not sufficient.
- Any future VBF mutation must use a generic extension-framework capability with manifest-aware backup/rollback instead of direct game-specific writes in core code.
