# Stellaris Extension Notes

## Identity

- Steam AppID: `281990`
- DMM extension ID: `stellaris`
- Nexus domain: none verified yet.

## Verified Sources

- Steam Deck installed app manifest snapshot; no official Vortex Stellaris extension found in checked source: `extensionTargets.md#priority-queue`

## Current DMM Capability

- Workshop-only first-party extension is registered.
- Steam Workshop coexistence/actions are declared.
- No Nexus domain, launcher descriptor handling, archive installer, or profile mapping is declared.

## Beta Gaps

- Verify Vortex/source behavior before adding Nexus or archive support.
- Define launcher descriptor generation and profile mapping.
- Determine whether Steam Workshop load order can be managed through Steam APIs alone or needs launcher file integration.

## Validation Targets

- Existing Workshop subscriptions.
- Disable/enable/unsubscribe action path.
- Future descriptor/load-order behavior after source verification.
