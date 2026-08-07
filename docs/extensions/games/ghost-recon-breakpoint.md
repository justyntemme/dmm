# Ghost Recon Breakpoint Extension Notes

## Identity

- Steam AppID: `2231380`
- DMM extension ID: `ghostreconbreakpoint`
- Nexus domain: `ghostreconbreakpoint`
- Vortex manifest package: Nexus site mod `972`, file `7463`, version `0.2.8`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Ghost Recon Breakpoint Vortex extension page: `https://www.nexusmods.com/site/mods/972`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Ghost Recon Breakpoint`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the DirectX/Vulkan executables, a base forge archive, and `sounddata/pc`.
- Blocks archive installs because the manifest describes AnvilToolkit, forge-file repacking, and rename prompts that need dedicated extension-framework support.

## Beta Gaps

- Inspect the Vortex package and representative archives.
- Add generic external-tool/rename-prompt/repack lifecycle support before enabling Breakpoint installs.
