# Nuclear Option Extension Notes

## Identity

- Steam AppID: `2168680`
- DMM extension ID: `nuclearoption`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=2168680&filters=categories`
- Live Steam Deck Workshop manifest snapshot: `/home/deck/.local/share/Steam/steamapps/workshop/appworkshop_2168680.acf`
- Live Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare a Nexus domain or archive installer.

## Beta Gaps

- Workshop enable/disable, unsubscribe, and order operations need live validation with actual Nuclear Option Workshop content.
- Any non-Workshop mod source needs source review before it is added to this extension.
