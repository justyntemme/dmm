# Civilization VII Extension Notes

## Identity

- Steam AppID: `1295660`
- DMM extension ID: `civilizationvii`
- Nexus domain: `civilizationvii`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/1182`
- Live Steam Deck install:
  - Game path: `steamapps/common/Sid Meier's Civilization VII`
  - Proton user mod path: `steamapps/compatdata/1295660/pfx/drive_c/users/steamuser/AppData/Local/Firaxis Games/Sid Meier's Civilization VII/Mods`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Installs `.modinfo` packages into the Proton LocalAppData `Mods` folder through an extension-owned target-root resolver.
- Uses copy deployment for this extension's files because the target is a Windows/Proton user-data folder.
- Reads the `.modinfo` `id`, `version`, and display `Name` for installer metadata.
- Rejects vanilla `Base/` and `DLC/` game-folder packages instead of treating game assets as normal user mods.

## Beta Gaps

- Needs live archive validation with at least one Nexus Vortex-compatible Civ VII mod.
- Needs verification of any load-order, in-game enablement, or external launcher semantics from the actual Vortex extension archive when the package source is inspected.
- Needs confirmation that multiple `.modinfo` modules in one archive should remain split by manifest id for all common Civ VII mod packages.

## Validation Targets

- Single `.modinfo` mod package.
- Multi-module archive with disjoint `.modinfo` folders.
- Archive wrapped in one download folder.
