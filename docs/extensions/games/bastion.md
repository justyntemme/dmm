# Bastion Extension Notes

## Identity

- Steam AppID: `107100`
- DMM extension ID: `bastion`
- Nexus domain: `bastion`

## Verified Sources

- Nexus API game list verified the `bastion` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Bastion handler was found.

## Current DMM Capability

- DMM declares the verified Nexus domain so Bastion can be discovered and browsed through the normal DMM Nexus flow.
- Archive installs are blocked until game-specific layouts are source-reviewed.

## Beta Gaps

- Review representative Nexus archives and any available extension/client source.
- Add only extension-owned installer rules for verified mod roots, file replacements, and rollback behavior.
