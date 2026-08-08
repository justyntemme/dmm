# Metal Gear Solid V: The Phantom Pain Extension Notes

## Identity

- Steam AppID: `287700`
- DMM extension ID: `metalgearsolidvtpp`
- Nexus domain: `metalgearsolidvtpp`

## Verified Sources

- SnakeBite README/source: `https://github.com/topher-au/SnakeBite`
- SnakeBite Nexus page: `https://www.nexusmods.com/metalgearsolidvtpp/mods/106`
- MGSV Modding Wiki SnakeBite page: `https://mgsvmoddingwiki.github.io/SnakeBite_Mod_Manager/`
- Nexus game domain: `https://www.nexusmods.com/metalgearsolidvtpp`
- Live Steam Deck path check: `/home/deck/.local/share/Steam/steamapps/common/MGS_TPP`

## Current DMM Capability

- DMM registers the Nexus domain so MGSV can be browsed through the existing browser-first Nexus flow.
- SnakeBite `.MGSV` packages are detected through their `metadata.xml` shape and staged as `metalgearsolidvtpp-snakebite` mods.
- Enabled SnakeBite packages rebuild DMM-generated `master/0/00.dat` and, when needed, `master/0/01.dat` through the extension `will-deploy` hook. The core deploy layer applies those generated archives with `patch-existing` restore metadata so profile switches, disable, purge, repair, and rollback stay DMM-owned.
- The generic GZS package supplies QAR/FPK read/write primitives. MGSV-specific SnakeBite metadata parsing and merge decisions stay inside this extension.
- DMM models SnakeBite's source-verified `MoveDatFiles` behavior without permanently mutating the base game archives: system entries are moved out of the generated `00.dat` into generated `01.dat`, while `foxpatch.dat` and enabled-mod QAR entries remain in generated `00.dat`.
- DMM mirrors SnakeBite's default conflict-checking stance by blocking enabled SnakeBite packages that modify the same QAR entry or the same file inside the same FPK. The deployment error names the conflicting mods and package paths, and the user can resolve it by disabling one mod or moving it to another profile.
- DMM validates SnakeBite package metadata before rebuilding packed archives. Packages created for pre-`0.8.0.0` SnakeBite metadata are blocked, `MGSVersion=0.0.0.0` is treated as a wildcard, and packages requiring a newer MGSV executable version are blocked through the extension-owned `mgsvtpp.exe` game-version provider.
- The SnakeBite deployment hook reports coarse progress phases through the generic extension event-progress callback, so large QAR/FPK work can update the normal DMM job row instead of appearing frozen.
- Required-file diagnostics verify `mgsvtpp.exe`, `master/0/00.dat`, and `master/0/01.dat`.

## Remaining Beta Gaps

- Live Deck validation against a real Nexus SnakeBite package is still required before this can be counted as MVP-complete.
- The conflict path is explicit and safe, but the MVP still needs a polished phone/tablet conflict review surface if users expect to inspect all SnakeBite conflicts outside the failed deploy/action message.
- Progress is currently phase-level, not byte-level. If real Deck testing shows individual QAR/FPK reads or writes still feel frozen, add lower-level progress callbacks to the generic GZS primitives.
