# Final Fantasy VII Rebirth Extension Notes

## Identity

- Steam AppID: `2909400`
- DMM extension ID: `finalfantasy7rebirth`
- Nexus domain: `finalfantasy7rebirth`

## Verified Sources

- Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/1150`
- Proton AppID/executable layout evidence: `https://github.com/ValveSoftware/Proton/issues/8408`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Copy-only deployment is the default strategy.
- Pak/ucas/utoc, FF7RML, UE4SS root, UE4SS mod, binary-adjacent, and root-folder archive shapes are represented.
- Unreal pak prefix load-order hook is implemented.

## Beta Gaps

- Actual extension package/source still needs deeper inspection if source becomes available outside the Nexus page.
- Config/save installers need validation.
- Package-specific load-order semantics need validation.
- No-symlink behavior on Deck storage needs live archive testing.

## Validation Targets

- Simple pak mod.
- FF7RML mod.
- UE4SS mod.
- Root/binary-adjacent mod.
