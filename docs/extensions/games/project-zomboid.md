# Project Zomboid Extension Notes

## Identity

- Steam AppID: `108600`
- DMM extension ID: `projectzomboid`
- Nexus domain: `projectzomboid`

## Verified Sources

- Vortex central extension manifest: `https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/692`
- Live Steam Deck user data path: `/home/deck/Zomboid/mods`

## Current DMM Capability

- Nexus domain and `mod.info` installer are registered.
- Project Zomboid mod archives deploy into `/home/deck/Zomboid/mods` through an extension-owned target-root resolver.
- The installer supports archives shaped like `Mod Name/mod.info`, `Mod Name/media/...`, and optional `poster.png`.
- Steam Workshop coexistence/actions are declared.

## Beta Gaps

- Needs live archive validation with a Nexus Project Zomboid mod.
- Needs actual extension archive/source inspection if it becomes available without relying on the Nexus package download flow.
- Needs profile/load-order validation against the in-game mod list and Workshop subscriptions.

## Validation Targets

- Existing Workshop subscriptions.
- Disable/enable/unsubscribe action path.
- Nexus/manual archive with `mod.info`, `media/`, and `poster.png`.
