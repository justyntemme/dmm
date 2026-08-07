# Megabonk Extension Notes

## Identity

- Steam AppID: `3405340`
- DMM extension ID: `megabonk`
- Nexus domain: `megabonk`
- Vortex manifest package: Nexus site mod `1495`, file `7663`, version `0.1.3`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Megabonk Vortex extension package v0.1.3: `https://www.nexusmods.com/site/mods/1495?tab=files`
- Inspected package source: `index.js` from Nexus site mod `1495`, file `7663`
- Live Steam Deck path check: `/run/media/deck/games/steamapps/common/Megabonk`

## Current DMM Capability

- Registers the full Steam AppID `3405340`, demo AppID `3520070`, and Nexus domain `megabonk`.
- Detects native Linux (`Megabonk.x86_64`, `GameAssembly.so`) and Windows/Proton (`Megabonk.exe`, `GameAssembly.dll`) install shapes.
- Implements source-verified Vortex installer rules for:
  - root archives containing `Megabonk_Data`;
  - Unity asset/resource archives targeting `Megabonk_Data`;
  - BepInEx ConfigurationManager archives targeting canonical `BepInEx`;
  - BepInEx and MelonLoader plugin DLL archives with marker-string detection;
  - custom-character archives through a persisted loader-choice prompt;
  - Windows/Proton `GameAssembly.dll` replacement archives;
  - Windows/Proton BepInEx and MelonLoader runtime package placement.
- Blocks mixed BepInEx/MelonLoader DLL archives instead of choosing a loader silently.
- Blocks arbitrary fallback/root-file placement until a narrower extension-owned rule can classify the archive.

## Platform Boundary

- The verified Vortex package downloads Windows x64 BepInEx and MelonLoader payloads.
- The live Steam Deck install is native Linux. DMM intentionally blocks those Windows runtime installers on native Linux until a Linux Megabonk loader package and launch behavior are source-verified.
- Runtime requirements still detect manually installed BepInEx/MelonLoader markers so DMM can warn when enabled loader-specific mods cannot load.

## Beta Gaps

- Verify native Linux BepInEx/MelonLoader packages and launch behavior for Megabonk before enabling automatic loader installation on the Deck-native install.
- Validate representative Nexus Megabonk archives through the browser `nxm://` capture path.
- Add a generic loader-selection/loader-conflict review surface so games with mutually exclusive mod loaders can show a clearer profile-level warning.
