# Total War: PHARAOH DYNASTIES Extension Notes

## Identity

- Steam AppID: `2951630`
- DMM extension ID: `totalwarpharaohdynasties`

## Verified Sources

- Live Steam Deck Workshop manifest snapshot: `/home/deck/.local/share/Steam/steamapps/workshop/appworkshop_2951630.acf`
- Live Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content because the Deck has a PHARAOH Workshop manifest.
- DMM does not declare a Nexus domain, archive installer, or Total War pack-file/load-order handling.

## Beta Gaps

- Steam Store appdetails did not expose category `30` during the latest verification, so PHARAOH Workshop support is currently based on observed Steam Deck Workshop state.
- Total War pack-file load order and launcher integration need source/runtime verification before DMM should manage non-Workshop mods.
- Workshop enable/disable, unsubscribe, and order operations need live validation with actual PHARAOH Workshop content.
