# WARNO Extension Notes

## Identity

- Steam AppID: `1611600`
- DMM extension ID: `warno`

## Verified Sources

- Live Steam Deck Workshop manifest snapshot: `/home/deck/.local/share/Steam/steamapps/workshop/appworkshop_1611600.acf`
- Live Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content because the Deck has a WARNO Workshop manifest.
- DMM does not declare a Nexus domain or archive installer.

## Beta Gaps

- Steam Store appdetails did not expose category `30` during the latest verification, so WARNO Workshop support is currently based on observed Steam Deck Workshop state.
- Workshop enable/disable, unsubscribe, and order operations need live validation with actual WARNO Workshop content.
- Any non-Workshop mod source needs source review before it is added to this extension.
