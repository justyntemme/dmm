# Transport Fever 2 Extension Notes

## Identity

- Steam AppID: `1066780`
- DMM extension ID: `transportfever2`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=1066780&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare Nexus domains or archive installers for Transport Fever 2.

## Beta Gaps

- Workshop enable/disable, unsubscribe, and order operations need live validation with actual subscribed Transport Fever 2 Workshop content.
- Local mod-folder semantics and load-order behavior need source or representative runtime review before DMM manages non-Workshop mods.
