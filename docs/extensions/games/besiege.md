# Besiege Extension Notes

## Identity

- Steam AppID: `346010`
- DMM extension ID: `besiege`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=346010&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare Nexus domains or archive installers for Besiege.

## Beta Gaps

- Workshop enable/disable, unsubscribe, and order operations need live validation with actual subscribed Besiege Workshop content.
- Any non-Workshop archive or Nexus flow needs source review before it is added to this extension.
