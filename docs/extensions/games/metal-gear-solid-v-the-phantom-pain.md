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
- SnakeBite `.MGSV` packages are detected through their `metadata.xml` shape and blocked with a clear unsupported message.
- Required-file diagnostics verify `mgsvtpp.exe`, `master/0/00.dat`, and `master/0/01.dat`.

## Beta Gaps

- Real MGSV support requires a generic extension-framework capability for packed QAR/FPK archive mutation:
  - read SnakeBite `metadata.xml`
  - prepare/restore `master/0/00.dat` and `master/0/01.dat`
  - merge FPK files
  - rebuild QAR archives
  - persist enough DMM-owned state for disable, uninstall, rollback, and profile switching
- Until that capability exists, DMM must not symlink or copy `.MGSV` package contents into the game folder.
