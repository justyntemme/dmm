# Ghost Recon Breakpoint Extension Notes

## Identity

- Steam AppID: `2231380`
- DMM extension ID: `ghostreconbreakpoint`
- Nexus domain: `ghostreconbreakpoint`
- Vortex manifest package: Nexus site mod `972`, file `7463`, version `0.2.8`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Ghost Recon Breakpoint Vortex extension page: `https://www.nexusmods.com/site/mods/972`
- Verified Vortex extension package file: `https://www.nexusmods.com/site/mods/972?tab=files&file_id=7463`
- Vortex bundled game extension source check: `https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Ghost Recon Breakpoint`

## Current DMM Capability

- Registers Steam AppID `2231380` and Nexus domain `ghostreconbreakpoint`.
- Mirrors the verified Vortex package's game registration:
  - executable: `GRB.exe`
  - mod path: game root `.`
  - required files: `GRB.exe`
  - mod merging and cleanup enabled
- Registers Vortex-declared tools:
  - Launch Game Ubisoft Plus: `GRB_UPP.exe`
  - Launch Vulkan Game: `GRB_vulkan.exe`
  - Custom Launch: `GRB.exe`
  - AnvilToolkit: `anviltoolkit.exe`
- Implements source-verified copy-only installers for:
  - AnvilToolkit archives containing `anviltoolkit.exe`
  - sound `.pck` archives into `sounddata/pc`
  - individual `.buildtable` archives into `Extracted/DataPC_patch_01.forge/Extracted/23_-_TEAMMATE_Template.data`
  - `Extracted` folder archives into the game root
  - `.forge` folder archives into `Extracted/<forge-folder>/...`
  - `.data` folder archives into `Extracted/<user-entered-forge-folder>/...` after an extension-owned text installer choice
  - loose `.data` file archives into `Extracted/<user-entered-forge-folder>/...` after an extension-owned text installer choice
  - `.forge` file replacement archives into the game root
  - root-folder archives containing `videos`
- Publishes Vortex-equivalent readme conflict/deploy ignores for `**/readme.txt`.
- Adds runtime diagnostics for required game files and AnvilToolkit presence.
- Queues a source-backed post-deploy AnvilToolkit action through DMM's generic extension `run-launch-tool` contract. When `anviltoolkit.exe` is present as an enabled managed tool/provider, Decky can launch it through the same Steam-owned tool execution path used by other extension tools.

## Beta Gaps

- `.data` folder archives and loose `.data` archives now use DMM's generic extension text-choice flow instead of Vortex's post-install rename dialog. This still needs live validation with representative Nexus archives.
- The Vortex fallback installer remains blocked because arbitrary root placement is not safe without a specific extension-owned rule.
- The AnvilToolkit action still needs live Deck validation with the actual Nexus tool archive and a representative data-mod deployment.
- Live-test representative Breakpoint archives through the BrowserView `nxm://` flow before treating this as release-ready.
