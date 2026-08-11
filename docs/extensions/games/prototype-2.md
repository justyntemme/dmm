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
- Representative TexMod instructions: `https://steamcommunity.com/sharedfiles/filedetails/?id=715011335`
- TexMod automation limitation reference: `https://www.nexusmods.com/tombraiderlegend/mods/92`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Prototype 2`

## Current DMM Capability

- DMM registers the Nexus domain so Prototype 2 can be browsed through the existing browser-first Nexus flow.
- DMM supports narrow root ASI plugin archives, including `.asi`, matching `.ini`, and optional `dinput8.dll` sidecars.
- DMM supports TexMod `.tpf` package archives by staging them under `DMM/TexMod/Packages`, supports TexMod tool archives containing `Texmod.exe`, reports a TexMod runtime requirement for enabled `.tpf` packages, and queues an extension launch action to open TexMod after deployment.
- Extracted RCF folders, standalone patchers, and broad root-copy archives remain blocked with a clear extension-owned message.
- Required-file diagnostics verify `prototype2.exe`, `art.rcf`, and `scripts.rcf`.

## Beta Gaps

- Live-test Prototype2Fix or another safe ASI plugin archive.
- Live-test the TexMod flow on Deck with a small `.tpf` package and confirm Proton passes keyboard/mouse input to the GUI tool acceptably.
- Add pattern-specific installers only after source/archive validation:
  - extracted `art`/RCF folder replacements
  - standalone tools or patchers
- Do not add a broad root-copy installer until representative archives prove it is safe for the common Prototype 2 Nexus flow.
