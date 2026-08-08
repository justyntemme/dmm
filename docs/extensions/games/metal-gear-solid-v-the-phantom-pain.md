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
- Enabled SnakeBite packages rebuild a DMM-generated `master/0/00.dat` through the extension `will-deploy` hook. The core deploy layer applies that generated archive with `patch-existing` restore metadata so profile switches, disable, purge, repair, and rollback stay DMM-owned.
- The generic GZS package supplies QAR/FPK read/write primitives. MGSV-specific SnakeBite metadata parsing and merge decisions stay inside this extension.
- The SnakeBite deployment hook reports coarse progress phases through the generic extension event-progress callback, so large QAR/FPK work can update the normal DMM job row instead of appearing frozen.
- Required-file diagnostics verify `mgsvtpp.exe`, `master/0/00.dat`, and `master/0/01.dat`.

## Remaining Beta Gaps

- Live Deck validation against a real Nexus SnakeBite package is still required before this can be counted as MVP-complete.
- The current implementation generates `00.dat` only. It source-verifies SnakeBite's normal install path, but `MoveDatFiles`-style first-run reshuffling of system files is not modeled as a separate DMM action yet.
- Conflict UX for multiple enabled SnakeBite packages should become explicit instead of relying only on profile priority.
- Progress is currently phase-level, not byte-level. If real Deck testing shows individual QAR/FPK reads or writes still feel frozen, add lower-level progress callbacks to the generic GZS primitives.
