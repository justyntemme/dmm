# Baldur's Gate 3 Extension Notes

## Identity

- Steam AppID: `1086940`
- DMM extension ID: `baldursgate3`
- Nexus domain: `baldursgate3`

## Verified Sources

- Vortex BG3 game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-baldursgate3/src/index.tsx`
- Vortex BG3 installers: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-baldursgate3/src/installers.ts`
- Vortex BG3 LSLib downloader: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-baldursgate3/src/githubDownloader.ts`
- LSLib upstream releases: `https://github.com/Norbyte/lslib/releases`

## Current DMM Capability

- Nexus domain, Steam AppID, and BG3 local data target roots are registered.
- `.pak`, BG3 Script Extender, engine-injector, loose `Data`, and LSLib tool archive planning are extension-owned.
- LSLib/Divine is modeled as a DMM-managed tool-only provider mod type.
- The BG3 pak metadata runtime requirement can be satisfied by either files already present in the game path or an enabled DMM-managed LSLib/Divine provider mod.
- DMM can auto-queue the source-verified Norbyte/lslib latest `ExportTool-vX.Y.Z.zip` archive through runtime acquisition when BG3 pak metadata support is required.

## Beta Gaps

- DMM does not yet execute LSLib/Divine to read pak `meta.lsx` data.
- `modsettings.lsx` generation uses the extension-owned Divine process bridge to list pak contents, extract `meta.lsx`, and generate the Public profile load-order file during deployment. BG3 load-order import/migration UI remains future parity work.
- Vortex `check-mods-version` parity remains missing; DMM can acquire the tool, but does not yet run BG3's source-backed update check event.
- Live BG3 archive validation is still required on Steam Deck.

## Validation Targets

- LSLib/Divine GitHub acquisition and tool-only install.
- Simple `.pak` mod that requires metadata extraction.
- BG3 Script Extender archive.
- Engine injector archive.
