# Extension Targets

Snapshot source: Steam manifests on the Steam Deck at `deck@192.168.8.102`, collected on 2026-07-29.

## Target Policy

- These installed games are the first pool for post-MVP extension targets.
- When a Vortex extension exists for a game, DMM should duplicate the Vortex extension behavior one-for-one before inventing new behavior.
- Source verification is required before implementing an extension: read the Vortex extension source, identify the installer rules, deployment folders, mod types, metadata parsers, runtime/tool requirements, and any special lifecycle hooks.
- Game-specific logic belongs in the extension package/spec. Core DMM code should expose reusable extension-framework APIs instead of hardcoding behavior for a single game.
- If a Vortex extension needs a capability that DMM does not have yet, add the smallest generic API that supports that capability and keep the game-specific directive in the extension.
- Do not guess Nexus domains in the UI or backend. Domains should come from extension metadata or a verified catalog lookup.

## Priority Queue

| Priority | App ID | Game | Library | Target Notes |
| --- | ---: | --- | --- | --- |
| MVP | 413150 | Stardew Valley | Internal | Current vertical slice. Clone Vortex Stardew behavior for SMAPI installer, SMAPI mod installer, root-folder installer, runtime requirements, and launch-tool integration. |
| P1 | 489830 | The Elder Scrolls V: Skyrim Special Edition | Internal | High-impact Vortex parity target. Requires source review for installers, deployment method, plugins/load order, archives, profiles, script extender/runtime tooling, and conflict management. |
| P1 | 377160 | Fallout 4 | Internal | High-impact Vortex parity target and first Windows/Proton test bed after MVP. Clean reinstall recommended before DMM testing because prior Vortex-managed state is suspected. Requires source review for plugin load order, F4SE/runtime tools, archive invalidation, game data roots, and conflict handling. |
| P1 | 292030 | The Witcher 3: Wild Hunt | Internal | Good Nexus/Vortex candidate. Requires source review for DLC/mod folder layout, script merger needs, and file conflict semantics. |
| P1 | 108600 | Project Zomboid | Internal | Workshop-only first-party extension implemented for coexistence/actions; no Nexus domain or archive installer is declared until Nexus/Vortex support is verified or a DMM-native design is approved. |
| P1 | 294100 | RimWorld | SD card | Verify Vortex behavior and whether Steam Workshop dominates the real user flow. |
| P1 | 233860 | Kenshi | Internal | Verify Vortex extension support, install roots, load order format, and profile handling. |
| P1 | 10150 | Prototype | Internal | First-party extension now registers Nexus browsing, live executable/RCF diagnostics, and a research blocker for archives. No Vortex extension was found in checked sources; specific ASI, TexMod, RCF, and patcher installers still need representative archive validation before support is enabled. |
| P1 | 115320 | Prototype 2 | Internal | First-party extension now registers Nexus browsing, live executable/RCF diagnostics, and a research blocker for archives. No Vortex extension was found in checked sources; specific ASI, TexMod, RCF, and patcher installers still need representative archive validation before support is enabled. |
| P1 | 287700 | Metal Gear Solid V: The Phantom Pain | Internal | First-party extension registers Nexus browsing, source-verified SnakeBite `.MGSV` package detection, extension-owned QAR/FPK generation for `master/0/00.dat` and `master/0/01.dat`, restore-aware deployment, SnakeBite-style package conflict blocking, and metadata/game-version validation. Needs live Nexus SnakeBite package validation and polished conflict-review UX. |
| P1 | 392160 | X4: Foundations | Internal | Verify extension behavior and mod folder layout. |
| P1 | 281990 | Stellaris | Internal | Workshop-only first-party extension implemented for coexistence/actions; no Nexus domain, launcher descriptor handling, archive installer, or profile mapping is declared until Vortex/source behavior is verified. |
| P2 | 975370 | Dwarf Fortress | SD card | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. DFHack pairing and local/archive mod support are deferred until the Dwarf Fortress/DFHack runtime boundary is verified. |
| P2 | 2346660 | DFHack - Dwarf Fortress Modding Engine | SD card | Treat as a tool/runtime target paired with Dwarf Fortress, not a standalone modded game until verified. |
| P2 | 286160 | Tabletop Simulator | Internal | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Save-object/cache semantics, profile behavior, and any Nexus/direct archive support need representative runtime/archive validation before support expands. |
| P2 | 1295660 | Sid Meier's Civilization VII | Internal | Verify Vortex support and expected mod locations. |
| P2 | 214950 | Total War: ROME II - Emperor Edition | Internal | First-party research-blocked extension now declares the Nexus domain `totalwarrome2`, live executable/pack diagnostics, and no Workshop support after Steam appdetails/local Deck checks found no Workshop category/state. No Vortex extension was found in checked sources; archive installs are blocked until representative Nexus pack/launcher/data-folder behavior is classified. |
| P2 | 885970 | Total War: ROME REMASTERED | Internal | Workshop-only first-party extension exists from Steam appdetails and observed Steam Deck Workshop manifest state. Total War pack-file/load-order handling and any Nexus/direct archive support remain blocked until source/runtime behavior is verified. |
| P2 | 2951630 | Total War: PHARAOH DYNASTIES | Internal | Workshop-only first-party extension exists from observed Steam Deck Workshop manifest state. Total War pack-file/load-order handling and any Nexus/direct archive support remain blocked until source/runtime behavior is verified. |
| P2 | 2909400 | Final Fantasy VII Rebirth | Internal | Verify Nexus/Vortex support and whether mods require external loaders. |
| P2 | 359870 | Final Fantasy X/X-2 HD Remaster | Internal | First-party research-blocked extension now declares the Nexus API-verified domain `finalfantasyxx2hdremaster` and live executable/VBF diagnostics. No Vortex extension was found in checked central/bundled sources; archive installs are blocked until representative mod layouts and any VBF mutation requirements are verified. |
| P2 | 1817190 | Marvel's Spider-Man: Miles Morales | Internal | First source-verified extension implemented from the Vortex Spider-Man/Miles package: `.mmpcmod` archives install under `SMPCTool/ModManager/MMPCMods`, multi-mod archives use installer choices, and DMM generates `ModManager.txt`. Still blocked for automatic MMPC tool install/execution and `.mmpcmodpack` submodule expansion until those generic extension-framework capabilities exist. |
| P2 | 996580 | Spyro Reignited Trilogy | Internal | First source-verified extension implemented from Vortex `game-spyroreignitedtrilogy`: `.pak` archives under `falcon/content/paks/~mods`, FOMOD archive exclusion, and Vortex-style pak load-order prefixes through the Unreal load-order hook. Still needs live archive validation. |
| P2 | 976730 | Halo: The Master Chief Collection | Internal | First source-verified extension implemented from Vortex `game-masterchiefcollection`: plug-and-play `modinfo.json`, `modpack_config.cfg`, recognized Halo game folders, Assembly tool metadata, build tag versioning, and Proton `ModManifest.txt` generation. Still needs live archive validation and launcher/no-EAC UX review. |
| P2 | 1237950 | Star Wars Battlefront II | Internal | Verify extension support and Frosty/mod-loader implications. |
| P2 | 1774580 | Star Wars Jedi: Survivor | Internal | Verify extension support and mod loader requirements. |
| P3 | 346010 | Besiege | Internal | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Nexus/direct archive support is deferred until source/runtime behavior is verified. |
| P3 | 2732960 | Command & Conquer Generals Zero Hour | SD card | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Local map/mod archive behavior needs representative archive review before support expands. |
| P3 | 718670 | Cultist Simulator | Internal | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Non-Workshop imports need source or archive review. |
| P3 | 310560 | DiRT Rally | SD card | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Loose-file/external-manager mod behavior is not yet verified. |
| P3 | 246620 | Plague Inc: Evolved | SD card | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Scenario/import formats need source or archive review before non-Workshop support. |
| P3 | 1948280 | Stacklands | Internal | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Thunderstore/local flows, if useful later, need official/source review. |
| P3 | 1066780 | Transport Fever 2 | SD card | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Local mod-folder/load-order behavior needs source/runtime review. |
| P3 | 973230 | We Who Are About To Die | Internal | First-party Workshop-only extension now declares Steam Workshop coexistence/actions from Steam appdetails verification. Non-Workshop archive behavior needs source or representative archive review. |
| P3 | all others below | Remaining installed games | Mixed | Review after MVP/P1 targets, unless the user needs a specific game sooner. |

## Installed Games Snapshot

### Half-Life 2 Source Search Paths

Snapshot source: `gameinfo.txt` files from `/home/deck/.local/share/Steam/steamapps/common/Half-Life 2`, collected over SSH on 2026-08-07.

| App ID | Game | Verified custom search path |
| ---: | --- | --- |
| 220 | Half-Life 2 | `hl2/custom/*` |
| 340 | Half-Life 2: Lost Coast | `lostcoast/custom/*`, then `hl2/custom/*` |
| 380 | Half-Life 2: Episode One | `episodic/custom/*`, then `hl2/custom/*` |
| 420 | Half-Life 2: Episode Two | `ep2/custom/*`, then `episodic/custom/*`, then `hl2/custom/*` |

### Internal Steam Library

| App ID | Game | Install Dir |
| ---: | --- | --- |
| 1473350 | (the) Gnorp Apologue | Gnorp |
| 107100 | Bastion | Bastion |
| 346010 | Besiege | Besiege |
| 774361 | Blasphemous | Blasphemous |
| 26800 | Braid | Braid |
| 291550 | Brawlhalla | Brawlhalla |
| 1578650 | Citizen Sleeper | Citizen Sleeper |
| 718670 | Cultist Simulator | Cultist Simulator |
| 1868140 | DAVE THE DIVER | Dave the Diver |
| 632470 | Disco Elysium | Disco Elysium |
| 377160 | Fallout 4 | Fallout 4 |
| 2909400 | Final Fantasy VII Rebirth | FINAL FANTASY VII REBIRTH |
| 359870 | Final Fantasy X/X-2 HD Remaster | FINAL FANTASY FFX&FFX-2 HD Remaster |
| 2231380 | Ghost Recon Breakpoint | Ghost Recon Breakpoint |
| 220 | Half-Life 2 | Half-Life 2 |
| 380 | Half-Life 2: Episode One | Half-Life 2 |
| 420 | Half-Life 2: Episode Two | Half-Life 2 |
| 340 | Half-Life 2: Lost Coast | Half-Life 2 |
| 976730 | Halo: The Master Chief Collection | Halo The Master Chief Collection |
| 219150 | Hotline Miami | hotline_miami |
| 233860 | Kenshi | Kenshi |
| 214560 | Mark of the Ninja | mark_of_the_ninja |
| 1817190 | Marvel's Spider-Man: Miles Morales | Marvel's Spider-Man Miles Morales |
| 220860 | McPixel | mcpixel |
| 2131630 | Metal Gear Solid - Master Collection Version | MGS1 |
| 2131640 | Metal Gear Solid 2: Sons of Liberty - Master Collection Version | MGS2 |
| 287700 | Metal Gear Solid V: The Phantom Pain | MGS_TPP |
| 1449560 | Metro Exodus Enhanced Edition | Metro Exodus Enhanced Edition |
| 686060 | Mewgenics | Mewgenics |
| 17410 | Mirror's Edge | mirrors edge |
| 761830 | Mr. Prepper | MrPrepper |
| 2168680 | Nuclear Option | Nuclear Option |
| 1687950 | Persona 5 Royal | P5R |
| 218230 | PlanetSide 2 | PlanetSide 2 |
| 108600 | Project Zomboid | ProjectZomboid |
| 10150 | Prototype | Prototype |
| 115320 | Prototype 2 | Prototype 2 |
| 2210 | Quake 4 | Quake 4 |
| 4760 | Rome: Total War | Rome Total War Gold |
| 4770 | Rome: Total War - Alexander | Rome Total War Alexander |
| 1216320 | Shieldwall | Shieldwall |
| 1295660 | Sid Meier's Civilization VII | Sid Meier's Civilization VII |
| 202170 | Sleeping Dogs | SleepingDogs |
| 2943150 | SNO: Ultimate Freeriding | SNO |
| 996580 | Spyro Reignited Trilogy | Spyro Reignited Trilogy |
| 1948280 | Stacklands | Stacklands |
| 1774580 | Star Wars Jedi: Survivor | Jedi Survivor |
| 1237950 | Star Wars Battlefront II | STAR WARS Battlefront II |
| 455910 | Star Wars: Rogue Squadron 3D | RogueSquadron |
| 413150 | Stardew Valley | Stardew Valley |
| 281990 | Stellaris | Stellaris |
| 286160 | Tabletop Simulator | Tabletop Simulator |
| 489830 | The Elder Scrolls V: Skyrim Special Edition | Skyrim Special Edition |
| 2753900 | The King is Watching | The King is Watching |
| 292030 | The Witcher 3: Wild Hunt | The Witcher 3 |
| 220780 | Thomas Was Alone | thomaswasalone |
| 2221490 | Tom Clancy's The Division 2 | Tom Clancy's The Division 2 |
| 2951630 | Total War: PHARAOH DYNASTIES | Total War PHARAOH DYNASTIES |
| 214950 | Total War: ROME II - Emperor Edition | Total War Rome II |
| 885970 | Total War: ROME REMASTERED | Total War ROME REMASTERED |
| 1611600 | WARNO | WARNO |
| 973230 | We Who Are About To Die | We who are about to Die |
| 392160 | X4: Foundations | X4 Foundations |

### SD Card Steam Library

| App ID | Game | Install Dir |
| ---: | --- | --- |
| 2732960 | Command & Conquer Generals Zero Hour | Command & Conquer Generals - Zero Hour |
| 2229870 | Command & Conquer: Generals | Command and Conquer Generals |
| 2346660 | DFHack - Dwarf Fortress Modding Engine | Dwarf Fortress |
| 310560 | DiRT Rally | DiRT Rally |
| 975370 | Dwarf Fortress | Dwarf Fortress |
| 224760 | FEZ | FEZ |
| 917150 | Godhood | Godhood |
| 70 | Half-Life | Half-Life |
| 297120 | Heavy Bullets | Heavy Bullets |
| 367520 | Hollow Knight | Hollow Knight |
| 3405340 | Megabonk | Megabonk |
| 2131680 | Metal Gear & Metal Gear 2: Solid Snake | MG and MG2 |
| 242680 | Nuclear Throne | Nuclear Throne |
| 246620 | Plague Inc: Evolved | PlagueInc |
| 620 | Portal 2 | Portal 2 |
| 1210320 | Potion Craft: Alchemist Simulator | Potion Craft |
| 2290180 | Riders Republic | RidersRepublic |
| 294100 | RimWorld | RimWorld |
| 239350 | Spelunky | Spelunky |
| 412830 | Steins;Gate | STEINS;GATE |
| 250900 | The Binding of Isaac: Rebirth | The Binding of Isaac Rebirth |
| 1066780 | Transport Fever 2 | Transport Fever 2 |

## Tool And Runtime Manifests

These app manifests are installed, but they are not primary game extension targets:

| App ID | Name | Library | Notes |
| ---: | --- | --- | --- |
| 993090 | Lossless Scaling | Internal | Utility app, not a mod target unless a future workflow needs it. |
| 3658110 | Proton 10.0 | Internal | Steam compatibility runtime. |
| 2805730 | Proton 9.0 | Internal | Steam compatibility runtime. |
| 1161040 | Proton BattlEye Runtime | Internal | Steam compatibility runtime. |
| 1826330 | Proton EasyAntiCheat Runtime | Internal | Steam compatibility runtime. |
| 1493710 | Proton Experimental | Internal | Steam compatibility runtime. |
| 1070560 | Steam Linux Runtime 1.0 (scout) | Internal | Steam runtime. |
| 1391110 | Steam Linux Runtime 2.0 (soldier) | Internal | Steam runtime. |
| 1628350 | Steam Linux Runtime 3.0 (sniper) | Internal | Steam runtime. |
| 4183110 | Steam Linux Runtime 4.0 | Internal | Steam runtime. |
| 228980 | Steamworks Common Redistributables | Internal | Shared dependency package. |
