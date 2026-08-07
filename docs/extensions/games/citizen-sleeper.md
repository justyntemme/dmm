# Citizen Sleeper Extension Notes

## Identity

- Steam AppID: `1578650`
- DMM extension ID: `citizensleeper`
- Nexus domain: `citizensleeper`
- Vortex manifest package: Nexus site mod `444`, file `1656`, version `1.0.0`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Citizen Sleeper Vortex extension page: `https://www.nexusmods.com/site/mods/444`
- Linked extension repository from manifest description: `https://github.com/BluesKutya/VortexExtensions`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/Citizen Sleeper`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Checks for the Unity executable and data folders on the Deck install.
- Blocks archive installs until the linked source repository is inspected and installer rules are classified.

## Beta Gaps

- Inspect the linked extension source and representative Nexus archives.
- Add only verified Unity/BepInEx/root-file installers.
