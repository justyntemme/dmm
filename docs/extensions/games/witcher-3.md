# The Witcher 3 Extension Notes

## Identity

- Steam AppID: `292030`
- Steam AppID DX entry: `499450`
- DMM extension ID: `witcher3`
- Nexus domain: `witcher3`

## Verified Sources

- Vortex Witcher 3 game registration: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/index.ts`
- Vortex Witcher 3 installers: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/installers.ts`
- Vortex Witcher 3 common merge/load-order constants: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/common.ts`
- Vortex Witcher 3 XML merge: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/mergers.ts`
- Vortex Witcher 3 lifecycle and load-order hooks: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/eventHandlers.ts`
- Vortex Witcher 3 Script Merger setup: `https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/scriptmerger.ts`

## Current DMM Capability

- Nexus domain and Steam AppIDs are registered.
- Menu mod root, mixed mod/DLC, top-level mod, content-only, and DLC installer shapes are represented.
- Script Merger is registered as a managed launch tool with source-backed GitHub acquisition from `IDCs/WitcherScriptMerger`. DMM recognizes Script Merger archives as `tool-only` installs, verifies the downloaded archive and extracted executable against the Vortex-pinned MD5 values from `MD5Cache.json`, stages the whole payload under DMM-owned storage, records tool metadata, and rewrites `WitcherScriptMerger.exe.config` after install with the game root, vanilla scripts root, and `Mods` root.
- Basic managed `mods.settings` generation exists through an extension deploy hook.
- Menu `.part.txt` fragments are merged by the Witcher 3 extension during `will-deploy`: DMM scans enabled `witcher3menumodroot` staging folders, ignores `input.xml` fragments like Vortex, merges fragment INI keys over the current or `.vortex_backup` Documents settings file, and returns restore-aware `patch-existing` mappings for files such as `input.settings`, `user.settings`, and `dx12user.settings`.
- Config-matrix XML files are merged by the Witcher 3 extension during `will-deploy`: DMM removes the raw config XML deployment mapping, reads the native or `.vortex_backup` game file, merges `UserConfig.Group` nodes by `id`, replaces matching `VisibleVars.Var` nodes by `id`, appends missing vars/groups, and returns one restore-aware `patch-existing` mapping.

## Beta Gaps

- Vortex hidden menu-mod cache/adoption behavior is incomplete.
- Script Merger execution still needs live validation on Deck after a managed tool install.
- Manual load-order preservation and full load-order UI semantics are incomplete.
- Collection/profile data behavior needs parity review.
- Live archive validation is required.

## Validation Targets

- Plain `mods/modName` archive.
- `dlc` archive.
- Mixed `mods` plus `dlc` archive.
- Menu XML mod that requires merge behavior.
