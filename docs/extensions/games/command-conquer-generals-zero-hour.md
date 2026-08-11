# Command & Conquer Generals Zero Hour Extension Notes

## Identity

- Steam AppID: `2732960`
- DMM extension ID: `commandconquergeneralszerohour`

## Verified Sources

- Steam Store appdetails category verification: `https://store.steampowered.com/api/appdetails?appids=2732960&filters=categories`
- Live Steam Deck manifest snapshot: `extensionTargets.md#installed-games-snapshot`

## Current DMM Capability

- DMM declares Steam Workshop coexistence and the standard Workshop action contract for installed/subscribed content.
- DMM supports narrow local/archive `.big` package installs into the Zero Hour game root.
- DMM does not declare a Nexus domain for Zero Hour because a verified Nexus automatic-download domain has not been confirmed for this Steam app.

## Beta Gaps

- Workshop enable/disable, unsubscribe, and order operations need live validation with actual subscribed Workshop content.
- Representative `.big` package installs need live validation in-game.
- GenLauncher packages, patchers, loose INI/data replacements, and full conversion launch flows remain blocked until source-reviewed extension rules exist.
