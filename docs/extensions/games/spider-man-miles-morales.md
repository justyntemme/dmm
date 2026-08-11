# Marvel's Spider-Man: Miles Morales Extension Notes

## Identity

- Steam AppID: `1817190`
- DMM extension ID: `spidermanmilesmorales`
- Nexus domain: `spidermanmilesmorales`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Spider-Man Remastered and Miles Morales Vortex extension page: `https://www.nexusmods.com/site/mods/443`
- Verified Vortex extension package file: `https://www.nexusmods.com/site/mods/443?tab=files&file_id=1831`
- Miles Morales MMPC Modding Tool page: `https://www.nexusmods.com/spidermanmilesmorales/mods/8`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Marvel's Spider-Man Miles Morales`

## Current DMM Capability

- `.mmpcmod` archives are installed into `SMPCTool/ModManager/MMPCMods`, matching the Vortex extension's Miles Morales mod path.
- Archives with multiple `.mmpcmod` files use DMM's installer-choice flow to mirror Vortex's selection prompt.
- DMM writes Vortex-style `SMPCTool/ModManager/ModManager.txt` entries for enabled DMM-managed `.mmpcmod` files during deployment.
- Runtime diagnostics verify `MilesMorales.exe`, `asset_archive/toc`, and `SMPCTool/MMPCTool.exe`.
- DMM installs MMPC Modding Tool archives as managed tool-only payloads under `SMPCTool/`, writes `SMPCTool/assetArchiveDir.txt` from the discovered game path when available, and registers a non-primary MMPC tool entry so the tool is visible in extension capabilities.

## Beta Gaps

- Vortex runs `MMPCTool.exe -install` after deploy to merge staged `.mmpcmod` files into game archives. DMM does not yet have the generic Proton/Windows tool-execution contract needed to run that automatically.
- `.mmpcmodpack` support is blocked until DMM has a Vortex submodule-equivalent archive expansion step.
- Suit Adder Tool support is not implemented yet.
