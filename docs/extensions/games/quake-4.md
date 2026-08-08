# Quake 4 Extension Notes

## Identity

- Steam AppID: `2210`
- DMM extension ID: `quake4`
- Nexus domain: `quake4`

## Verified Sources

- Nexus API game list verified the `quake4` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Quake 4 handler was found.
- Representative Nexus pages show two source-verified Quake 4 layouts:
  - `q4base` replacements such as `gamex86.dll`, `.pk4`, and config/script files.
  - Full mod folders that must be launched with `+set fs_game <folder>`.
- ModDB's Quake 4 modding tutorial identifies `q4base` as the base folder containing `.pk4` packages and other game content.

## Current DMM Capability

- DMM supports narrow `q4base` replacement/package archives through extension-owned installer rules.
- Explicit `q4base/` wrapper folders are stripped and deployed under the game's `q4base`.
- Top-level replacement `.pk4`, `.dll`, `.cfg`, `.scriptcfg`, `.def`, material, shader, texture, and related loose files are staged under `q4base`.
- DMM supports full `fs_game` folder mods by staging the detected mod folder beside `q4base` and declaring an extension-owned dynamic launch argument for `+set fs_game <folder>`.
- The generic launch-tool framework now expands dynamic launch arguments from enabled profile mod target roots; the Quake 4 extension uses that capability without core Quake-specific branches.

## Beta Gaps

- Live-test a q4base `.pk4` or `gamex86.dll` replacement archive.
- Live-test a full `fs_game` folder archive such as `Impacts_and_Injury`, including the Decky/Steam launch-option update and in-game load behavior.
- Confirm unmanaged q4base replacement conflicts are presented as normal DMM conflict/advanced-deployment choices rather than overwritten silently.
