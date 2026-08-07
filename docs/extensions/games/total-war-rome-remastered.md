# Total War: ROME REMASTERED Extension Notes

## Identity

- Steam AppID: `885970`
- DMM extension ID: `totalwarromeremastered`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=885970&filters=categories`
- Live Steam Deck Workshop manifest snapshot: `/home/deck/.local/share/Steam/steamapps/workshop/appworkshop_885970.acf`
- Live Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM does not declare a Nexus domain, archive installer, or Total War pack-file/load-order handling.

## Beta Gaps

- Total War pack-file load order and launcher integration need source/runtime verification before DMM should manage non-Workshop mods.
- Workshop enable/disable, unsubscribe, and order operations need live validation with actual ROME REMASTERED Workshop content.
