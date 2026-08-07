# Final Fantasy VII Rebirth Extension Notes

## Identity

- Steam AppID: `2909400`
- DMM extension ID: `finalfantasy7rebirth`
- Nexus domain: `finalfantasy7rebirth`

## Verified Sources

- Cached Vortex extension package on the Steam Deck: `/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Final Fantasy VII Rebirth Vortex Extension v0.4.0/index.js`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/1150`
- Proton AppID/executable layout evidence: `https://github.com/ValveSoftware/Proton/issues/8408`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Copy-only deployment is the default strategy because the verified Vortex extension disables symlinks for FF7 Rebirth's IO Store layout.
- Pak/ucas/utoc, FF7RML loader, FF7RML `.uplugin` mods, UE4SS root, UE4SS scripts, UE4SS DLL mods, UE4SS LogicMods, UE4SS script/LogicMods combo packages, config, saves, binary-adjacent, and root-folder archive shapes are represented.
- Unreal pak prefix load-order hook is implemented.
- Config archives deploy through an extension-owned Proton Documents target root:
  `Documents/My Games/FINAL FANTASY VII REBIRTH/Saved/Config/WindowsNoEditor`
- Save archives deploy through an extension-owned Proton Documents target root:
  `Documents/My Games/FINAL FANTASY VII REBIRTH/Steam/{numericUserId}` when that directory exists, otherwise the Steam save folder itself.
- Multi-option pak-family archives produce an installer-choice request, mirroring Vortex's prompt when the archive contains more pak/ucas/utoc files than a normal IO Store set.

## Beta Gaps

- Live FF7 Rebirth archive validation is still needed for pak, FF7RML, UE4SS, config, save, and root/binaries variants.
- Vortex's optional UE4SS auto-download action is not implemented as an automatic DMM runtime helper yet.
- Package-specific load-order semantics need validation against real FF7 Rebirth pak archives.

## Validation Targets

- Simple pak mod.
- FF7RML mod.
- UE4SS mod.
- Config file mod.
- Save file import.
- Root/binary-adjacent mod.
