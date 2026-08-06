# DMM Virtual File System Model

Decky Mod Manager treats the game directory as a deployment target, not as the canonical mod library. The canonical state lives in DMM-owned storage and SQLite. The game folder receives only the files or links needed for the currently applied profile.

## Storage Ownership

DMM-owned storage is split by purpose:

- `downloads`: cached source archives and provider metadata.
- `staging`: normalized, extracted mod contents arranged by game and installed mod.
- `tmp`: extraction and installer workspaces that can be deleted after a failed or completed operation.
- `backups`: restore material for DMM-managed copied files or generated patches.
- `logs`: backend and Decky diagnostics.
- `db`: SQLite state for games, profiles, installed mods, deployment manifests, jobs, and source/provider identity.

Archives and staged files are shared between profiles. A profile does not own a second copy of the mod files. A profile owns membership and state: enabled/disabled, priority, conflict winners, load-order data when a game needs it, and the deployment manifest produced when that profile is applied.

## VFS Projection

DMM's MVP "virtual file system" is a manifest-driven projection rather than a kernel-mounted overlay. The deployment planner builds a desired file view from:

- The selected game.
- The selected profile.
- Enabled profile mods.
- Extension-declared mod types and deployment roots.
- Extension-generated files such as plugin/load-order files or config patches.
- Profile priority and explicit per-file conflict winners.
- The last active DMM deployment manifest.

The planner compares the desired view with the currently active DMM-owned deployment and produces operations:

- `add`: place a new DMM-owned artifact into the game or app-data target.
- `keep`: leave an existing artifact in place because it already matches the desired source.
- `replace`: update a DMM-owned artifact whose source changed.
- `remove`: remove a DMM-owned artifact that is no longer part of the selected profile.
- `conflict`: block before touching an unmanaged target.

The game directory is never the source of truth. If a staged mod is deleted or moved unexpectedly, DMM should report the installed record as needing recovery rather than inventing a new target from the live game folder.

## Symlink Deployment

The normal deployment artifact is a link from the game-facing path to the staged source file:

```text
game/Mods/ExampleMod/manifest.json -> dmm/staging/413150/<installed-mod-id>/manifest.json
```

The deployment strategy is chosen by the extension and filesystem constraints:

- Prefer hardlinks when staging and target are on the same filesystem and the game supports them.
- Use symlinks when hardlinks are not viable and the game/runtime supports them.
- Use copies only when the extension declares that the target cannot safely be linked, such as launch/runtime files that must resolve relative files from the game root.

Every deployed artifact is recorded in the deployment manifest with its target path, source path, strategy, checksum where applicable, owning profile, owning installed mod, and restore information if DMM had to patch or copy over a managed target.

## Profile Switching

Switching profiles means reconciling deployment artifacts, not rewriting staged mods:

1. Load the current active deployment manifest for the game.
2. Build the desired manifest for the target profile.
3. Remove artifacts owned by the old manifest and absent from the new manifest.
4. Keep artifacts already matching the new manifest.
5. Add or replace artifacts required by the new manifest.
6. Persist the new active manifest only after filesystem verification succeeds.

This lets two profiles share one installed mod while keeping different enabled states and priorities. Moving a mod between profiles changes `profile_mods` membership. It does not copy staged files unless a later storage-management feature explicitly relocates DMM storage.

## Conflict Safety

DMM may remove or repair only files it can prove it owns through its manifest. Existing files, folders, or links in the target path block deployment unless they match the active DMM manifest or an extension-declared safe generated-file policy.

Unmanaged states are treated conservatively:

- Manual mods in the game folder block target paths that DMM would overwrite.
- Existing Vortex/NMM deployments are detected as external state and not automatically cleaned in MVP.
- Steam Workshop content is Steam-owned platform content. DMM can display and manage it through verified Steam APIs, but it is not staged or purged as DMM-owned storage.

## Rollback And Repair

Deployment is transactional at the product level. If an apply operation fails after touching files, DMM uses the previous manifest and per-operation restore records to revert DMM-owned changes made during that attempt.

Repair compares the active manifest to the live filesystem and can recreate missing links, remove broken DMM-owned artifacts, or report stale staging paths. Purge removes only active-manifest-owned artifacts and clears the active deployment record. It never deletes unmanaged parent game folders.

## Extension Boundary

Game extensions describe how archive contents become deployable mappings:

- Supported source domains and providers.
- Installer matchers and metadata extractors.
- Mod types and target roots.
- Launch/runtime requirements.
- Generated files and merge hooks.
- Preferred deployment strategies.
- Load-order or plugin activation rules.

The core validates every mapping returned by extensions before writing files. Extension output is input to the VFS projection; it is not permission to bypass path validation, conflict checks, manifest ownership, or rollback.

## Future Direction

A real mounted overlay or FUSE-style VFS could be evaluated later, but MVP uses the safer Vortex-inspired staging-plus-manifest model. It is easier to inspect, easier to repair over SSH, works across Steam library paths, and keeps profile transitions explicit and auditable.
