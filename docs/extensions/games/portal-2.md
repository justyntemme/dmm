# Portal 2 Extension Notes

## Identity

- Steam AppID: `620`
- DMM extension ID: `portal2`
- Nexus domain: `portal2`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/109`

## Current DMM Capability

- Registers the Steam AppID and Nexus domain from the Vortex manifest.
- Installs regular archive contents into `portal2_dlc3`, matching the Vortex extension page guidance.
- Strips a single harmless download-wrapper folder before deploying into `portal2_dlc3`.

## Beta Gaps

- Needs live validation with representative Portal 2 Nexus mods.
- Needs actual extension archive/source inspection if it becomes available without relying on the Nexus package download flow.
- No load-order, tool, or special package semantics are claimed yet.

## Validation Targets

- Basic materials/scripts replacement archive.
- Wrapped archive with one top-level folder.
