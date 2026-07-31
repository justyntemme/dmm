# Decky Mod Manager Plane Test Checklist

Use this checklist for manual Steam Deck, phone, and iPad testing. Capture exact error text when convenient, but logs live on the Deck so short notes are enough.

## Setup

- Install the latest package from `/home/deck/.testing/decky-mod-manager.tar.gz`.
- Reboot if Decky does not reload the plugin cleanly after install.
- Open Decky Mod Manager in Gaming Mode and confirm the backend starts automatically.
- Phone/tablet URL: `http://192.168.8.102:17942` when the Deck is on the same LAN/hotspot.
- Confirm Nexus API key is configured in the phone/tablet Settings page.
- Open Settings > Sources on phone/iPad and confirm Nexus, Thunderstore, GitHub Releases, Modrinth, GameBanana, Direct Archive URL, planned credential-gated sources, and Steam Workshop status are listed correctly.
- If you have mod.io or CurseForge API keys, paste them in Settings > Sources and confirm those sources switch from `Needs Key` to `Ready`.

## Decky Plugin

- Open DMM and verify the Quick Access panel shows the compact status launcher.
- Open full DMM and verify route tabs are visible: `Manage`, `Games`, `Settings`, `Debug`.
- In `Games`, use D-pad only:
  - Search for Stardew Valley.
  - Change sort order.
  - Favorite/unfavorite a game.
  - Select Stardew Valley.
  - Confirm the selected row is visibly highlighted and text does not clip.
- Inside Stardew Valley:
  - Verify mod rows fit horizontally.
  - Verify focused rows are visibly highlighted.
  - Press `A` on a mod to enable/disable it.
  - Confirm DMM applies the profile automatically and says the game may need restart.
  - Confirm `Y` reinstall and menu/remove actions are visible and not clipped.
- Press `B` inside the selected game view and note whether it returns to the game list or exits DMM.

## Nexus Capture Flow

- From Gaming Mode Firefox, open a Stardew Valley Nexus mod page.
- Click `Mod Manager Download`.
- Confirm a Decky toast appears when DMM captures/downloads the archive.
- Open phone/iPad Action Center:
  - The action should appear without manual refresh.
  - If auto-install is enabled, the action may complete automatically.
  - If installer choices or manual install are needed, controls should disable immediately after tapping.
- After install, open Stardew Valley in DMM:
  - The new mod should appear disabled by default unless auto-enable is on.
  - Enable it and confirm profile changes apply automatically.

## Nexus Browse Inside DMM

- In Decky `Games`, select Stardew Valley and open `Browse Nexus`.
- Search a term such as `config`, `content`, or `tractor`.
- Test sort/time controls.
- Open file rows.
- Use `Add` for a file and confirm it enters the same captured-install pipeline.
- If Nexus requires a browser-generated link, use `Open File Page` and then `Mod Manager Download`.
- Repeat on iPad web UI from the selected Stardew game page.

## Stardew Runtime Verification

- Ensure SMAPI diagnostics show green/ok.
- Enable at least one SMAPI mod.
- Launch Stardew Valley from Steam.
- Confirm the SMAPI console appears and the enabled mod is listed/loaded.
- Disable the same mod in DMM, relaunch Stardew, and confirm it no longer loads.

## Profiles

- Create or select a second profile.
- Enable a different mod set.
- Switch back to the default profile.
- Confirm DMM reconciles enabled mods without requiring staging knowledge.
- Check that profile actions do not show stale pending changes after apply.

## FOMOD / Installer Choices

- Test with a known Vortex-compatible FOMOD from a supported game such as Fallout 4 or Skyrim.
- Capture through browser `Mod Manager Download`, not direct HTTPS file add, unless DMM explicitly says direct install is supported.
- Confirm Action Center shows installer choices on phone/iPad.
- In Gaming Mode, confirm the Decky modal appears automatically when auto-install and auto-display FOMOD are enabled.
- Change a choice and confirm dependent/conditional choices update without refreshing.
- Finish install and verify the mod appears disabled by default.

## Updates

- Use `Check Updates` on Stardew mods from phone/iPad and Decky.
- If an update exists, install it.
- Confirm the old installed row is replaced rather than duplicated.
- Confirm enabled/disabled state is preserved through the update.
- If the provider blocks direct update download, confirm DMM offers `Open Provider File Page` or opens the provider file page from Decky.

## Rollback / Recovery

- Open the advanced recovery/deployment panel.
- Confirm deployment history is visible.
- Use `Restore last applied state` only after noting the current mod state.
- Confirm it creates a rollback job and only touches DMM-owned files.

## Steam Workshop

- Open a game with installed Steam Workshop content.
- Confirm DMM labels Workshop mods as Steam-managed/source-tagged.
- Move Workshop entries up/down only on a safe test game and confirm DMM queues a `Steam Workshop` order action instead of treating Workshop content like a downloaded archive.
- Test enable/disable/unsubscribe only if the game is safe to modify during travel.
- Confirm DMM still allows Nexus/DMM-managed mods for the same game when the extension declares coexistence safe.

## Source Tags

- Confirm every mod row shows a small source pill such as `Nexus`, `Steam Workshop`, or future provider name.
- Confirm source tags appear on both phone/iPad and Decky surfaces where mod rows are visible.
- In the phone/iPad Mods view, filter by source and sort by source.
- In the Decky Mods view, cycle mod sorting until `Source` appears and confirm the list remains navigable.
- In a selected game, paste a direct HTTPS archive URL only when it is safe to test that archive.
- Confirm direct/provider imports are tied to the selected game and show a non-Nexus source tag.
- In a selected game on phone/iPad, upload a safe `.zip`, `.7z`, or `.rar` through `Local Archive`.
- Confirm local archive uploads install through the same Action Center/installer-choice path and show a `Local` source tag.
- In a selected game, paste a safe `https://modrinth.com/mod/{slug}` or `https://modrinth.com/mod/{slug}/version/{version}` URL and confirm it enters the captured-install flow and is tagged `Modrinth`.
- In a selected game, paste a safe `https://gamebanana.com/mods/{id}` URL and confirm it enters the captured-install flow and is tagged `GameBanana`.
- Bare `https://gamebanana.com/dl/{file}` links should import as `Direct` only when added from a selected game, because GameBanana's file API does not expose reliable parent mod/game metadata.
- With a mod.io key configured, select a safe game and paste a `https://mod.io/g/{game}/m/{mod}` URL. Confirm it enters the captured-install flow and is tagged `mod.io`.
- With a CurseForge key configured, select a safe game and paste a `https://www.curseforge.com/{game}/{section}/{mod}` URL. Confirm it enters the captured-install flow and is tagged `CurseForge`.
- Confirm Steam Workshop entries remain tagged as Steam-managed and are not confused with DMM-downloaded catalog mods.

## Notes To Send Back

- Game/app ID tested.
- Mod name and Nexus/Workshop ID if visible.
- Which UI was used: Decky, phone, or iPad.
- Expected result.
- Actual result.
- Whether the UI updated without refresh.
- Any toast shown or missing.
