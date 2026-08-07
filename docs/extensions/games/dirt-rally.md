# DiRT Rally Extension Notes

## Identity

- Steam AppID: `310560`
- DMM extension ID: `dirtrally`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=310560&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare Nexus domains or archive installers for DiRT Rally.

## Beta Gaps

- Workshop enable/disable, unsubscribe, and order operations need live validation with actual subscribed DiRT Rally Workshop content.
- Any loose-file or external-manager mod flow needs source review before it is added to this extension.
