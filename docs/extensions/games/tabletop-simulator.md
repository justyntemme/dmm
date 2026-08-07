# Tabletop Simulator Extension Notes

## Identity

- Steam AppID: `286160`
- DMM extension ID: `tabletopsimulator`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=286160&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare a Nexus domain, archive installer, local save-object importer, or Tabletop Simulator cloud/cache handling yet.

## Beta Gaps

- Tabletop Simulator Workshop objects have game-specific save/cache semantics that need verified Steam/Deck runtime testing before DMM should expose deeper ordering or profile behavior.
- Nexus/direct archive support is deferred until a verified source and representative archive layouts are reviewed.
