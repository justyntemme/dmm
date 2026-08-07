# Prototype Extension Notes

## Identity

- Steam AppID: `10150`
- DMM extension ID: `prototype`
- Nexus domain: `prototype`

## Verified Sources

- Nexus game domain: `https://www.nexusmods.com/prototype`
- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Representative root ASI fix: `https://www.nexusmods.com/prototype/mods/52`
- Representative RCF/extractor mod discussion: `https://www.nexusmods.com/prototype/mods/81`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Prototype`

## Current DMM Capability

- DMM registers the Nexus domain so Prototype can be browsed through the existing browser-first Nexus flow.
- All archives are blocked with a clear research message because no Vortex extension was found in the checked sources and observed mods use conflicting manual install models.
- Required-file diagnostics verify `prototypef.exe`, `art.rcf`, and `scripts.rcf`.

## Beta Gaps

- Add pattern-specific installers only after source/archive validation:
  - root ASI/fix drops beside `prototypef.exe`
  - extracted `art`/RCF folder replacements
  - TexMod package handling and launch flow
  - standalone tools or patchers
- Do not add a broad root-copy installer until representative archives prove it is safe for the common Prototype Nexus flow.
