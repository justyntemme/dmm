# Bastion Extension Notes

## Identity

- Steam AppID: `107100`
- DMM extension ID: `bastion`
- Nexus domain: `bastion`

## Verified Sources

- Nexus API game list verified the `bastion` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Bastion handler was found.
- Representative Nexus instructions verify copying mod repository files into `Content/Game`.
- Representative Nexus executable-patch mod exists and is intentionally blocked because DMM does not patch game executables without a verified transform/rollback capability.
- Live Steam Deck inspection verified the native Linux equivalent path is `Linux/Content/Game`.

## Current DMM Capability

- DMM declares the verified Nexus domain so Bastion can be discovered and browsed through the normal DMM Nexus flow.
- DMM supports platform-aware `Content/Game` config replacement archives.
- Native Linux installs target `Linux/Content/Game`; Windows/Proton installs target `Content/Game`.
- Executable patches, binary tools, save edits, and unclassified payloads remain blocked.

## Beta Gaps

- Live-test one Nexus `Content/Game` config replacement archive on the Steam Deck native Linux install.
- Add executable patch support only after a generic binary patch/restore capability exists.
