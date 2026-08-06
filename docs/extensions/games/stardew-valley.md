# Stardew Valley Extension Notes

## Identity

- Steam AppID: `413150`
- DMM extension ID: `stardewvalley`
- Nexus domain: `stardewvalley`

## Verified Sources

- Vortex Stardew game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/game/StardewValleyGame.ts`
- Vortex Stardew installers: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/installers`
- Vortex config-mod behavior: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-stardewvalley/src/configMod`
- Steam Deck SMAPI guide: `https://stardewvalleywiki.com/Modding:Installing_SMAPI_on_Steam_Deck`

## Current DMM Capability

- Nexus browsing and captured install pipeline are supported.
- SMAPI installer and SMAPI mod folder planning are implemented by the extension.
- SMAPI launch tool metadata is extension-owned and live-tested on the Deck.
- Generated SMAPI config files are preserved through an extension event hook.
- Native Linux and Windows/Proton install platform matching are represented in the extension model.

## Beta Gaps

- Continue validating broad Stardew Nexus archive shapes, especially config-menu-heavy mods and content packs.
- Re-test FOMOD or installer-choice flow if a Stardew archive needs choices.
- Keep launch-tool configuration surfaced as extension metadata, not core Stardew-specific code.

## Validation Targets

- SMAPI itself.
- Generic Mod Config Menu.
- A SMAPI content pack with dependencies.
- A duplicate/update install path into a non-default profile.
