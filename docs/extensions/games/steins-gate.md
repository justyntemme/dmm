# Steins;Gate Extension Notes

## Identity

- Steam AppID: `412830`
- DMM extension ID: `steinsgate`
- Nexus domain: `steinsgate`

## Verified Sources

- Nexus API game list verified the `steinsgate` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Steins;Gate handler was found.
- Representative Nexus instructions verify replacing files under `USRDIR/movie/1920x1080` by dragging the extracted archive into the Steam `common` folder.

## Current DMM Capability

- DMM declares the verified Nexus domain for browsing/capture.
- DMM supports direct `USRDIR` replacement archives.
- Archives wrapped above `USRDIR` are normalized so replacements deploy under the game `USRDIR` folder.
- Executable launchers, patch tools, scripts, DLL payloads, and unclassified game-root archives remain blocked.

## Beta Gaps

- Live-test one Nexus `USRDIR` replacement archive.
- Add patch-tool support only after a generic external-tool/transform capability exists.
