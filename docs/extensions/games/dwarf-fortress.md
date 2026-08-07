# Dwarf Fortress Extension Notes

## Identity

- Steam AppID: `975370`
- DMM extension ID: `dwarffortress`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=975370&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not treat DFHack (`2346660`) as a standalone modded game target yet.
- DMM does not declare Nexus domains or archive installers for Dwarf Fortress.

## Beta Gaps

- DFHack integration must be designed as a paired tool/runtime capability if we manage DFHack-specific mods later.
- Steam Workshop enable/disable and ordering need live validation with actual subscribed Dwarf Fortress Workshop content.
- Local/raw mod archive support is deferred until representative layouts and Dwarf Fortress/DFHack boundaries are verified.
