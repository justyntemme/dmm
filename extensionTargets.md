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
| P1 | 377160 | Fallout 4 | Internal | High-impact Vortex parity target. Requires source review for plugin load order, F4SE/runtime tools, archive invalidation, game data roots, and conflict handling. |
| P1 | 292030 | The Witcher 3: Wild Hunt | Internal | Good Nexus/Vortex candidate. Requires source review for DLC/mod folder layout, script merger needs, and file conflict semantics. |
| P1 | 108600 | Project Zomboid | Internal | Verify Nexus/Vortex support versus Steam Workshop-first workflows before implementation. |
| P1 | 294100 | RimWorld | SD card | Verify Vortex behavior and whether Steam Workshop dominates the real user flow. |
| P1 | 233860 | Kenshi | Internal | Verify Vortex extension support, install roots, load order format, and profile handling. |
| P1 | 10150 | Prototype | Internal | User-mentioned custom target. Requires TexMod/mod-loader investigation and Vortex/source review before implementation. |
| P1 | 115320 | Prototype 2 | Internal | Related custom target. Requires separate source review; do not assume Prototype 1 behavior applies. |
| P1 | 287700 | Metal Gear Solid V: The Phantom Pain | Internal | Verify Vortex extension behavior, mod package formats, and any external mod loader requirements. |
| P1 | 392160 | X4: Foundations | Internal | Verify extension behavior and mod folder layout. |
| P1 | 281990 | Stellaris | Internal | Verify Vortex behavior, launcher/mod descriptor handling, and profile mapping. |
| P2 | 975370 | Dwarf Fortress | SD card | DFHack is also installed; verify whether DMM should target game mods, DFHack mods, or both. |
| P2 | 2346660 | DFHack - Dwarf Fortress Modding Engine | SD card | Treat as a tool/runtime target paired with Dwarf Fortress, not a standalone modded game until verified. |
| P2 | 286160 | Tabletop Simulator | Internal | Verify Nexus/Vortex ecosystem and content roots. |
| P2 | 1295660 | Sid Meier's Civilization VII | Internal | Verify Vortex support and expected mod locations. |
| P2 | 214950 | Total War: ROME II - Emperor Edition | Internal | Verify Vortex support and launcher/mod-pack behavior. |
| P2 | 885970 | Total War: ROME REMASTERED | Internal | Verify extension support separately from Rome II. |
| P2 | 2951630 | Total War: PHARAOH DYNASTIES | Internal | Verify extension support and mod-pack mechanics. |
| P2 | 2909400 | Final Fantasy VII Rebirth | Internal | Verify Nexus/Vortex support and whether mods require external loaders. |
| P2 | 359870 | Final Fantasy X/X-2 HD Remaster | Internal | Verify extension support and platform-specific install roots. |
| P2 | 1817190 | Marvel's Spider-Man: Miles Morales | Internal | Verify Vortex support and asset replacement strategy. |
| P2 | 976730 | Halo: The Master Chief Collection | Internal | Verify per-title mod roots and anti-cheat/launcher implications. |
| P2 | 1237950 | Star Wars Battlefront II | Internal | Verify extension support and Frosty/mod-loader implications. |
| P2 | 1774580 | Star Wars Jedi: Survivor | Internal | Verify extension support and mod loader requirements. |
| P3 | all others below | Remaining installed games | Mixed | Review after MVP/P1 targets, unless the user needs a specific game sooner. |

## Installed Games Snapshot

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
