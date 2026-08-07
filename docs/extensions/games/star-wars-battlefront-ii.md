# Star Wars Battlefront II Extension Notes

## Identity

- Steam AppID: `1237950`
- DMM extension ID: `starwarsbattlefront22017`
- Nexus domain: `starwarsbattlefront22017`

## Verified Sources

- Public Vortex Star Wars Battlefront II extension source: `https://github.com/alistair3149/game-starwarsbattlefront22017`
- Current Nexus Vortex extension page: `https://www.nexusmods.com/site/mods/112`
- Live Steam Deck install path verification:
  - Game folder: `/home/deck/.local/share/Steam/steamapps/common/STAR WARS Battlefront II`
  - Executable: `starwarsbattlefrontii.exe`

## Current DMM Capability

- Nexus domain and Steam AppID are registered.
- Copy-only deployment is the default strategy.
- `.fbmod` archive planning is implemented from the verified public Vortex extension behavior.
- Single `.fbmod` archives install the files from the matched `.fbmod` folder into `FrostyModManager/Mods/StarWarsBattlefrontII`.
- Multi-`.fbmod` archives pause as DMM installer-choice actions so the user can choose the variant before staging.
- Frosty Mod Manager is registered as a required runtime dependency for `.fbmod` mods.
- Frosty Mod Manager is registered as the primary launch tool when enabled Battlefront II `.fbmod` mods are present, including the Vortex-verified `-launch default` argument.

## Beta Gaps

- DMM does not auto-install Frosty Mod Manager yet.
- DMM does not run Frosty Mod Manager or clean/rebuild Frosty `ModData` yet.
- Frosty plugins, DatapathFix setup, and Frosty project/import automation require source-backed parity work before promotion.
- Live Nexus archive validation is required.

## Validation Targets

- Single `.fbmod` Nexus archive.
- Multi-`.fbmod` Nexus archive to validate the installer-choice flow.
- Frosty Mod Manager installed under `FrostyModManager/FrostyModManager.exe`.
- Launch option application through the extension-owned Frosty primary tool.
