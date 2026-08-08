# MVP Manual Testing

Use this file as the manual validation checklist for DMM `0.1.0`. Automated checks should prove API shape and basic planner behavior; this checklist covers the parts that need a real Steam Deck, Gaming Mode, Nexus browser credentials, Steam APIs, or game launch.

## Required Setup

- Install the newest package on the Steam Deck.
- Reboot if Decky does not reload the plugin cleanly.
- In Gaming Mode, open Decky Mod Manager and confirm the backend starts automatically.
- Phone/tablet URL: `http://192.168.8.102:17942` on the same LAN/hotspot.
- Confirm Settings > Sources shows:
  - `Nexus Mods` ready.
  - `Thunderstore`, `GitHub Releases`, `Modrinth`, `GameBanana`, `Direct Archive URL`, and `Local Archive` ready.
  - `mod.io` and `CurseForge` as ready only when keys are configured.
  - `ModDB` and `itch.io` deferred until a supported automated API/client path is verified.
  - `Steam Workshop` ready for installed-content management only.

## Release Blockers

- No visible installed game should be `unsupported`.
- Every MVP target must have an explicit extension profile with one of these states:
  - Full installer support from verified Vortex/source behavior.
  - Workshop-only management from verified Steam Workshop state/API behavior.
  - Metadata/source-only support with no false install claims when no safe automated handler exists.
- Stardew Valley must remain the reference complete vertical slice.
- Nexus capture must use the controlled browser flow and Nexus' own Mod Manager Download link, not a direct API-download shortcut for non-premium/browser-required files.
- Newly installed mods should be disabled by default unless auto-enable is explicitly enabled.
- Enable/disable should apply the selected profile without exposing staging as the normal workflow.
- FOMOD and other installer-choice flows must persist, resume, and update without manual refresh.

## Smoke Commands

Run these from the repo when the Deck server is online:

```bash
BASE_URL=http://192.168.8.102:17942 ./testing/live_web_asset_check.sh
BASE_URL=http://192.168.8.102:17942 ./testing/live_extension_coverage_check.sh
BASE_URL=http://192.168.8.102:17942 APP_ID=413150 ./testing/live_provider_resolve_check.sh
```

The provider resolve check should verify source tags without downloading archives. `mod.io` and `CurseForge` run only when API keys and test URLs are provided through environment variables.

## Decky Plugin

- Open DMM and verify route tabs are visible and responsive: `Manage`, `Games`, `Settings`, `Debug`.
- Verify the backend status, build fingerprint, and log terminal are visible from Debug.
- In Games, use D-pad only:
  - Search for Stardew Valley.
  - Change sort order.
  - Favorite/unfavorite a game.
  - Select Stardew Valley.
  - Confirm focused rows are visibly highlighted and text does not clip.
- Inside a selected game:
  - Mod rows fit horizontally.
  - Focused rows are visibly highlighted.
  - `A` toggles enable/disable.
  - Profile changes apply automatically.
  - Reinstall/remove controls are visible and not clipped.
- Confirm Decky source tags show on installed mods, Workshop rows, jobs, and Action Center rows.

## Phone And Tablet UI

- Test phone portrait.
- Test iPad/tablet portrait and landscape.
- Confirm no primary page requires horizontal scrolling.
- Confirm the top status strip buttons navigate to Action Center, the game picker, and Sources.
- Open Action Center with zero actions and confirm it is still a useful clickable tab with an empty state.
- Confirm blocked installer candidates show as review-only items without an `Install to` profile selector or installer choices.
- Select a game, switch between Mods, Actions, Profiles, Review, and Paths.
- Confirm source filters and source pills are visible where mod/job/action rows appear.
- Confirm web UI updates after actions without manual refresh.

## Nexus Browser Capture

- In Decky, select Stardew Valley.
- Open Explore Mods > Nexus Mods.
- Search a known term such as `config`, `content`, or `tractor`.
- Change sort/time filters.
- Open a result's real Nexus page in the controlled browser.
- Click Nexus' `Mod Manager Download` / Vortex link.
- Confirm DMM captures the browser-generated `nxm://` credential.
- Confirm a Decky toast appears when capture/download starts.
- Confirm the browser closes after successful capture and returns to the selected game page.
- Confirm the phone/tablet Action Center updates without refresh.
- Trigger one enable/disable apply and confirm Decky shows at most one deploy completion toast, not repeated deploy-running toasts.

## Stardew Valley Reference Slice

- Install SMAPI through Nexus capture.
- Install at least one SMAPI mod through Nexus capture.
- Confirm SMAPI diagnostics are green.
- Confirm Steam launch options point at SMAPI through the Decky/Steam API path.
- Launch Stardew from Steam.
- Confirm SMAPI starts and the enabled mod loads.
- Disable the mod in DMM, relaunch, and confirm it no longer loads.
- Create a second profile, enable a different mod set, switch profiles, and confirm deployed files change correctly.

## FOMOD / Installer Choices

- Stardew Valley negative fixture: `Neural Harvest` (`https://www.nexusmods.com/stardewvalley/mods/32817`, file `128820`).
  - Do not use it as a success fixture: the downloaded archive does not contain the files referenced by its own `fomod/ModuleConfig.xml` (`Common/NeuralHarvest.dll`, etc.).
  - The archive contains `build.sh` and `build_vortex.sh`, but Vortex FOMOD support does not auto-run arbitrary build scripts. The uploaded file is source/package-prep input, not the built FOMOD output.
  - Current expected behavior: DMM should block it before showing installer choices and keep it as a review-only Action Center item with a clear malformed-installer reason.
- Preferred real success fixture: Fallout 4 `FOMOD for Achievements Mods Enabler` (`https://www.nexusmods.com/fallout4/mods/90384`), found through Decky Nexus search query `fomod`.
- Secondary success fixtures: Fallout 4 search query `fomod` or Skyrim SE search query `fomod`; prefer small patch/option packs instead of large all-in-one compendiums.
- Capture through the controlled Nexus browser, not a direct HTTPS file URL.
- Confirm Action Center shows installer choices on phone/tablet.
- If auto-display FOMOD is enabled, confirm Decky opens a modal or gives a clear toast telling the user to open DMM.
- Change a choice and confirm conditional choices update without refresh.
- Finish install and confirm the mod is installed disabled unless auto-enable is on.
- Reinstall the same exact file and confirm saved exact-file choices are reused or can be forgotten.

## Game Extension Manual Matrix

These targets need real archive/deploy validation before they can be called complete, even when planner tests pass:

- Stardew Valley: already live-verified; repeat after major pipeline changes.
- Fallout 4: Nexus capture, FOMOD, F4SE archive, plugin activation, archive invalidation, load-order preview.
- Skyrim SE: Nexus capture, FOMOD, SKSE archive, plugin activation, archive invalidation, load-order preview.
- The Witcher 3: mod/DLC archive, generated `mods.settings`, Script Merger notice, menu-fragment notice.
- Final Fantasy VII Rebirth: pak/ucas/utoc, UE4SS, FF7RML, copy deployment, sortable pak prefixes.
- RimWorld: one-`About.xml` archive, Workshop coexistence.
- Kenshi: `.mod` folder archive, Workshop order/disable/enable.
- X4: `content.xml` archive, Proton Documents extension root.
- Halo MCC: plug-and-play `modinfo.json`, generated Proton `ModManifest.txt`.
- Spyro Reignited Trilogy: `.pak` archive and sortable pak prefixes.
- Spider-Man Miles Morales: `.mmpcmod` archive and generated load-order file.
- Bastion: native Linux `Content/Game` config replacement archive to `Linux/Content/Game`; executable patch archive remains blocked.
- Mirror's Edge: `TdGame/CookedPC` Unreal package replacement archive; user-Documents `Published/CookedPC` mod-menu flow remains blocked.
- Portal 2: VPK/archive deployment to `portal2_dlc3`.
- Half-Life 2 and episodes: VPK deployment to verified `custom` roots.
- Half-Life: `valve/` content replacement archive and loose `.bsp` map archive to `valve/maps`; standalone GoldSrc mod folders stay blocked.
- Quake 4: q4base `.pk4`/DLL/config replacement archive, conflict warning if replacing base files, blocked `fs_game` folder archive until dynamic launch action exists.
- Rome: Total War: vanilla `data/` replacement archive and Alexander `alexander/data/` replacement archive; full conversion launchers stay blocked.
- Spelunky: `Data/` replacement archive, loose `Localization`/`Music`/`Textures` archive, and blocked Patchlunky/Spelunktool/raw texture-source archive.
- STEINS;GATE: `USRDIR/` replacement archive and blocked executable/tool archive.
- Civilization VII: `.modinfo` package into Proton LocalAppData Mods.
- The Binding of Isaac: archive-root deployment into `mods`.
- Potion Craft: BepInEx runtime package, BepInEx plugin DLL archive, runtime diagnostic, enable/disable/deploy.
- Dave the Diver: BepInEx IL2CPP runtime package, BepInEx plugin DLL archive, runtime diagnostic, enable/disable/deploy.
- Mr. Prepper: BepInEx runtime package, BepInEx plugin DLL archive, runtime diagnostic, enable/disable/deploy.
- Blasphemous: native Linux BepInEx runtime package, executable `run_bepinex.sh`, Steam launch-tool action, plugin DLL archive, runtime diagnostic, enable/disable/deploy.

Targets that currently must not claim automated install support until source or representative archive behavior is verified:

- Brawlhalla
- Command & Conquer: Generals
- Final Fantasy X/X-2 HD Remaster
- Metal Gear Solid V: The Phantom Pain
- Persona 5 Royal
- Prototype
- Prototype 2
- Riders Republic
- The Division 2
- The King is Watching
- Total War: ROME II

## Steam Workshop

- Open a game with Workshop items, such as Kenshi, RimWorld, Project Zomboid, or Stellaris.
- Confirm Workshop items are labeled Steam-managed/source-tagged.
- Confirm DMM-managed mods can coexist where the extension says it is safe.
- Queue a safe order change and confirm it executes through Decky/Steam APIs.
- Test enable/disable only on a safe item.
- Test unsubscribe only when intentionally okay with changing Steam subscription state.

## Rollback And Recovery

- Open Advanced Profile Tools.
- Confirm deployment history is visible.
- Run Restore Last Applied State after noting the current mod state.
- Confirm a rollback job appears and only DMM-owned files are touched.
- Confirm failed apply jobs show a clear failure toast and recovery path.

## Notes To Report

For every failed manual test, capture:

- Game/app ID.
- Provider/source.
- Mod name and URL or ID.
- UI surface: Decky, phone, or tablet.
- Expected result.
- Actual result.
- Whether the UI updated without refresh.
- Any toast shown or missing.
