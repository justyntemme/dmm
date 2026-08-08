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
- DMM supports narrow root ASI plugin archives, including `.asi`, matching `.ini`, and optional `dinput8.dll` sidecars.
- TexMod packages, extracted RCF folders, standalone patchers, and broad root-copy archives remain blocked with a clear extension-owned message.
- Required-file diagnostics verify `prototype2.exe`, `art.rcf`, and `scripts.rcf`.

## Beta Gaps

- Live-test Prototype2Fix or another safe ASI plugin archive.
- Add pattern-specific installers only after source/archive validation:
  - extracted `art`/RCF folder replacements
  - TexMod package handling and launch flow
  - standalone tools or patchers
- Do not add a broad root-copy installer until representative archives prove it is safe for the common Prototype 2 Nexus flow.
