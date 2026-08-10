# Final Fantasy VII Rebirth Extension Notes

## Identity

- Steam AppID: `2909400`
- DMM extension ID: `finalfantasy7rebirth`
- Nexus domain: `finalfantasy7rebirth`

## Verified Sources

- Cached Vortex extension package on the Steam Deck: `/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Final Fantasy VII Rebirth Vortex Extension v0.4.0/index.js`
- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/1150`
- Public Vortex extension manifest entry: `Final Fantasy VII Rebirth Vortex Extension` v0.5.2, Nexus site mod `1150`, file `6893`.
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
- UE4SS-dependent script, DLL, LogicMods, and combo mods declare a runtime requirement for UE4SS. The requirement is satisfied by either a DMM-managed UE4SS provider mod or a live `End/Binaries/Win64/dwmapi.dll` marker in the game folder.
- UE4SS acquisition is extension-owned runtime behavior: DMM mirrors the Vortex `Download UE4SS` action from cached extension v0.4.0 by declaring Nexus mod `267`, file `1351`, and routing it through the generic captured-install/runtime-provider pipeline with auto-acquire enabled.

## Beta Gaps

- Live FF7 Rebirth archive validation is still needed for pak, FF7RML, UE4SS, config, save, and root/binaries variants.
- Live UE4SS acquisition validation is still needed against a real Deck FF7 Rebirth runtime-provider install.
- Package-specific load-order semantics need validation against real FF7 Rebirth pak archives.

## Validation Targets

- Simple pak mod.
- FF7RML mod.
- UE4SS mod.
- Config file mod.
- Save file import.
- Root/binary-adjacent mod.
