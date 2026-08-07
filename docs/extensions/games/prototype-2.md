# Prototype 2 Extension Notes

## Identity

- Steam AppID: `115320`
- DMM extension ID: `prototype2`
- Nexus domain: `prototype2`

## Verified Sources

- Nexus game domain: `https://www.nexusmods.com/prototype2`
- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Representative root ASI fix: `https://www.nexusmods.com/prototype2/mods/42`
- Representative standalone patcher: `https://www.nexusmods.com/prototype2/mods/94`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Prototype 2`

## Current DMM Capability

- DMM registers the Nexus domain so Prototype 2 can be browsed through the existing browser-first Nexus flow.
- All archives are blocked with a clear research message because no Vortex extension was found in the checked sources and observed mods use conflicting manual install models.
- Required-file diagnostics verify `prototype2.exe`, `art.rcf`, and `scripts.rcf`.

## Beta Gaps

- Add pattern-specific installers only after source/archive validation:
  - root ASI/fix drops beside `prototype2.exe`
  - extracted `art`/RCF folder replacements
  - TexMod package handling and launch flow
  - standalone tools or patchers
- Do not add a broad root-copy installer until representative archives prove it is safe for the common Prototype 2 Nexus flow.
