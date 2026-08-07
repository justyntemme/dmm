# Extension Research Notes

DMM game support is extension-owned. The core mod manager must not guess Nexus domains, installer roots, launch tools, load-order files, Workshop behavior, or runtime requirements for a game.

Each game note in `docs/extensions/games/` tracks:

- Verified upstream sources.
- Current DMM extension capability.
- Beta gaps before the extension can be treated as broadly supported.
- Suggested validation mods or live checks.

The runtime source of truth is still the compiled Go extension registered under `internal/extensions/`. These docs explain the research state behind those extensions and the next beta validation work.

## Support Badges

- `DMM`: a first-party DMM extension is registered for the Steam app.
- `Nexus`: the extension declares at least one verified Nexus game domain.
- `Workshop`: the extension declares Steam Workshop coexistence/actions.
- `Installers`: the extension declares archive planning and/or installer-choice support.
- `Load Order`: the extension declares plugin activation, merge, or load-order behavior.
- `Launch`: the extension declares launch tools or primary-tool behavior.

## Current First-Party Notes

- [Stardew Valley](games/stardew-valley.md)
- [Bastion](games/bastion.md)
- [Blasphemous](games/blasphemous.md)
- [Braid](games/braid.md)
- [Brawlhalla](games/brawlhalla.md)
- [Fallout 4](games/fallout-4.md)
- [FEZ](games/fez.md)
- [Skyrim Special Edition](games/skyrim-special-edition.md)
- [The Witcher 3](games/witcher-3.md)
- [Besiege](games/besiege.md)
- [Citizen Sleeper](games/citizen-sleeper.md)
- [Command & Conquer: Generals](games/command-conquer-generals.md)
- [Disco Elysium](games/disco-elysium.md)
- [Command & Conquer Generals Zero Hour](games/command-conquer-generals-zero-hour.md)
- [Cultist Simulator](games/cultist-simulator.md)
- [Dave the Diver](games/dave-the-diver.md)
- [DiRT Rally](games/dirt-rally.md)
- [Final Fantasy VII Rebirth](games/final-fantasy-vii-rebirth.md)
- [Final Fantasy X/X-2 HD Remaster](games/final-fantasy-x-x2-hd-remaster.md)
- [Ghost Recon Breakpoint](games/ghost-recon-breakpoint.md)
- [Half-Life](games/half-life.md)
- [Half-Life 2](games/half-life-2.md)
- [Hollow Knight](games/hollow-knight.md)
- [Hotline Miami](games/hotline-miami.md)
- [Kenshi](games/kenshi.md)
- [Megabonk](games/megabonk.md)
- [Metro Exodus](games/metro-exodus.md)
- [Mewgenics](games/mewgenics.md)
- [Mirror's Edge](games/mirrors-edge.md)
- [Mr. Prepper](games/mr-prepper.md)
- [Persona 5 Royal](games/persona-5-royal.md)
- [Potion Craft: Alchemist Simulator](games/potion-craft.md)
- [Quake 4](games/quake-4.md)
- [Riders Republic](games/riders-republic.md)
- [RimWorld](games/rimworld.md)
- [Rome: Total War](games/rome-total-war.md)
- [Halo: The Master Chief Collection](games/halo-master-chief-collection.md)
- [Metal Gear & Metal Gear 2 Master Collection](games/metal-gear-and-metal-gear-2-master-collection.md)
- [Metal Gear Solid V: The Phantom Pain](games/metal-gear-solid-v-the-phantom-pain.md)
- [Metal Gear Solid Master Collection](games/metal-gear-solid-master-collection.md)
- [Metal Gear Solid 2 Master Collection](games/metal-gear-solid-2-master-collection.md)
- [Metal Gear Solid 3 Master Collection](games/metal-gear-solid-3-master-collection.md)
- [Spyro Reignited Trilogy](games/spyro-reignited-trilogy.md)
- [Marvel's Spider-Man: Miles Morales](games/spider-man-miles-morales.md)
- [Spelunky](games/spelunky.md)
- [Steins;Gate](games/steins-gate.md)
- [Tom Clancy's The Division 2](games/the-division-2.md)
- [The King Is Watching](games/the-king-is-watching.md)
- [X4: Foundations](games/x4-foundations.md)
- [Civilization VII](games/civilization-vii.md)
- [Dwarf Fortress](games/dwarf-fortress.md)
- [Portal 2](games/portal-2.md)
- [Prototype](games/prototype.md)
- [Prototype 2](games/prototype-2.md)
- [Project Zomboid](games/project-zomboid.md)
- [Plague Inc: Evolved](games/plague-inc-evolved.md)
- [Nuclear Option](games/nuclear-option.md)
- [Nuclear Throne](games/nuclear-throne.md)
- [Star Wars Battlefront II](games/star-wars-battlefront-ii.md)
- [Star Wars Jedi: Survivor](games/star-wars-jedi-survivor.md)
- [STAR WARS: Rogue Squadron 3D](games/star-wars-rogue-squadron.md)
- [Stellaris](games/stellaris.md)
- [Stacklands](games/stacklands.md)
- [Tabletop Simulator](games/tabletop-simulator.md)
- [The Binding of Isaac: Rebirth](games/the-binding-of-isaac-rebirth.md)
- [Total War: PHARAOH DYNASTIES](games/total-war-pharaoh-dynasties.md)
- [Total War: ROME II](games/total-war-rome-ii.md)
- [Total War: ROME REMASTERED](games/total-war-rome-remastered.md)
- [Transport Fever 2](games/transport-fever-2.md)
- [WARNO](games/warno.md)
- [We Who Are About To Die](games/we-who-are-about-to-die.md)

## Research Rule

Before adding or promoting support for a game, inspect the relevant Vortex extension source, Nexus domain behavior, and at least one representative archive. If a source cannot be verified, keep the extension capability narrow and mark the gap here.
