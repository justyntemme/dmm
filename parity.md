# Vortex Parity Gaps

This document tracks the practical gap between Decky Mod Manager and Vortex, ordered by the impact each gap has on quality of life and user confidence. Vortex is broad: the current official Vortex page lists support for more than 65 games, deep Nexus integration, collections, FOMOD work, LOOT-based load ordering, and ongoing extension/game-specific fixes. DMM should not copy the Electron app shape, but it should absorb the workflow lessons that matter on Steam Deck.

Sources checked:

- https://www.nexusmods.com/site/mods/1
- https://github.com/Nexus-Mods/Vortex/wiki
- https://github.com/Nexus-Mods/Vortex/wiki/MODDINGWIKI-Users-General-Deployment-Methods
- https://github.com/Nexus-Mods/Vortex/wiki/MODDINGWIKI-Users-FAQ

## 1. Interactive Installer Support

- Vortex capability: FOMOD and other installer flows can present choices, automate selected paths, and participate in collections.
- DMM status: detects FOMOD and fails safely with a clear unsupported-installer message. FOMOD/installer-choice support is now MVP-required immediately after Stardew extension-framework parity.
- User impact: critical. Many Nexus "Mod Manager Download" archives are not plain copy/extract installs.
- Architectural fit: add an installer stage between archive inspection and staging. Jobs must pause, persist installer state, and resume after UI choices.
- Decision needed: implement a native Go FOMOD parser, bind an existing parser/runtime, or make installer parsing part of game-handler modules.

## 2. Install Instructions And Mod Types

- Vortex capability: downloaded archives are installed through installer instructions and mod types before deployment. Extensions can detect archive layouts, assign mod types, emit install instructions, and route different outputs to different deploy roots.
- DMM status: has an internal install planner backed by Vortex-modeled game metadata specs for the Stardew MVP slice. The current spec separates installer matching from mod type/deploy root behavior, detects valid relaxed `manifest.json` archives for `stardew-valley-installer`, maps root-folder `Content/` archives through `sdvrootfolder`, extracts Linux `smapi-installer` embedded payload archives, emits explicit target mappings, carries target policies and display-name policies, and stores planner/evidence metadata in staged manifests. Mod type deploy roots, deployment eligibility, and game/domain routing are now spec-owned instead of server-owned. Metadata extractors are declared by installer specs: SMAPI manifests ingest logical file names, unique IDs, versions, content-pack targets, and dependencies, while a generic JSON manifest extractor is available for future Vortex-style specs. A Deck copied-data rehearsal proves SMAPI root files are deployed as DMM-managed links. DMM does not yet import or execute broad Vortex extension metadata automatically.
- User impact: critical. Without this layer, DMM can stage installer/tool archives, dependency packages, or multi-type packages as if every file belongs under one game mod folder.
- Architectural fit: evolve the current metadata evaluator so provider metadata, Vortex-style installer/mod-type specs, metadata extractors, and game-handler capabilities all feed the same output contract: mod type, install instructions, deploy target mappings, runtime requirements, and blocked/unsupported reasons.
- Decision needed: whether Vortex metadata enters DMM as reviewed Go specs, a declarative schema, translated extension/package data, or a hybrid before adding more game handlers or special installers.

## 3. Game Handler Coverage

- Vortex capability: many game extensions define discovery, mod types, install rules, dependency behavior, tools, and health checks.
- DMM status: Stardew Valley is the only supported deployment target.
- User impact: critical. The manager is not broadly useful until automatic Nexus downloads can be staged/deployed for more games.
- Architectural fit: introduce a small internal handler contract around game discovery, supported mod types, target mapping, validation, and health checks.
- Decision needed: compiled Go handlers only for the next release, scriptable handlers, or a hybrid extension boundary.

## 4. Safe Adoption Of Existing Mod State

- Vortex capability: manages its own staging/deployment state and reports external changes.
- DMM status: detects dirty/non-clean games and blocks non-MVP deployments; no adoption/cleanup wizard yet.
- User impact: high. Steam Deck users may arrive from broken Vortex/manual installs and need a safe path forward.
- Architectural fit: add a read-only scan that classifies files as unmanaged, likely Vortex-owned, DMM-owned, or unknown before any cleanup action.
- Decision needed: report-only import first, controlled adoption, or cleanup tooling that can remove old Vortex artifacts.

## 5. Conflict Resolution

- Vortex capability: exposes file conflicts, lets users define rules, and keeps plugin load order separate from file override behavior.
- DMM status: preview detects unmanaged conflicts and resolves duplicate staged targets by simple profile priority.
- User impact: high. Simple priority is enough for early Stardew testing, but not enough for large mod lists.
- Architectural fit: extend deployment plans with stable conflict groups and per-file winner overrides stored per profile.
- Decision needed: whether overrides are per file, per folder/mod type, or rule-based "load before/after" relationships.

## 6. Mod Management UI Model

- Vortex/MO2 capability: keep downloads, install state, enabled/disabled mods, profiles, ordering, conflicts, and deployment actions visually distinct while still making the next action obvious.
- DMM status: the mobile/tablet UI now has game modules, install requests, staged mods, priorities, preview, deploy, repair, and purge, but the deployment area still needs a stronger product model and final polish.
- User impact: high. A mod manager earns trust by making file-impacting actions understandable before they run, especially on a phone controlling a Steam Deck game folder.
- Architectural fit: keep the backend as the source of truth for install phases and deployment manifests, then expose concise state summaries plus drill-down details in Svelte. Avoid placing raw file-operation controls ahead of install/profile context.
- Decision needed: final information architecture for the game Plugins surface: timeline-first, table-first, or dashboard-first with advanced drill-down.

## 7. Load Order And Plugin Sorting

- Vortex capability: uses LOOT internally for plugin sorting and supports custom dependency/group rules.
- DMM status: no load-order subsystem; current priority only affects deployment target conflicts.
- User impact: high for Bethesda-style games, low for Stardew-only MVP.
- Architectural fit: keep load order as a separate game-handler capability rather than merging it with deployment priority.
- Decision needed: bind LOOT/libloot for relevant games, shell out to LOOT, or defer to game-specific sorters.

## 8. Mod Requirements And Dependencies

- Vortex capability: tracks mod/file requirements and collection dependencies.
- DMM status: dependency tab currently covers external helper tools. New staged manifests persist install-plan metadata such as `mod_type` plus spec-extracted manifest attributes. Game diagnostics/mobile Review expose registry-derived runtime requirements and missing required manifest dependencies from enabled staged metadata, including SMAPI file and Steam launch-option detection for the Stardew slice. Nexus mod/file requirements are not resolved yet.
- User impact: high. Users need to know why a downloaded mod will not run.
- Architectural fit: provider metadata should expose requirements, and game handlers should validate runtime dependencies such as mod loaders, script extenders, launch options, and game tools. This must explain cases where DMM-owned files are deployed but the game is still launched without the required runtime.
- Decision needed: Nexus-only requirement resolver first, or a provider-neutral dependency model before UI work.

## 9. Collections And Bulk Install

- Vortex capability: supports Nexus Collections, optional members, progress tracking, installer choices, and curated install state.
- DMM status: no collections or bulk install workflow.
- User impact: medium-high. Collections are a major convenience feature but can wait until single-mod install is robust.
- Architectural fit: collections should create a batch install session composed of normal install jobs, with profile-targeted output.
- Decision needed: support Nexus Collections directly before adding other providers, or wait for the generic provider abstraction.

## 10. Update Checks

- Vortex capability: surfaces mod updates and download/install history.
- DMM status: no update checks; local records store source game/mod/file IDs and checksums.
- User impact: medium-high. Users need to know when a staged mod is outdated.
- Architectural fit: provider interface should compare installed source file metadata against current upstream metadata.
- Decision needed: implement Nexus-only update checks first, or design the provider update contract before adding UI.

## 11. Deployment Strategy Selection

- Vortex capability: chooses the best deployment method for the game/system and documents hardlink, symlink, and other strategies.
- DMM status: planner supports hardlink, symlink, and copy; Stardew deployment currently uses symlinks.
- User impact: medium. Symlink deployment is appropriate for the current Linux Deck slice, but some games/filesystems may need another method.
- Architectural fit: deployment strategy should be selected per game/profile after probing filesystem support and handler compatibility.
- Decision needed: automatic strategy selection only, or expose advanced per-game override controls.

## 12. Download Queue Controls

- Vortex capability: mature download/install pipeline with retries, cancellation, and clearer phase tracking.
- DMM status: jobs persist and show state. Waiting/running install requests can be canceled, interrupted running imports restore as waiting after restart, and failed retryable install requests keep request metadata so the user can retry from the phone/tablet UI. Throttling, concurrent queue limits, partial download resume, and richer retry policy are incomplete.
- User impact: medium. This becomes painful with large archives or poor network conditions.
- Architectural fit: pass request-scoped contexts through provider resolve, download, inspect, and extract; persist retryable failure state.
- Decision needed: minimal cancel/retry controls first, or full queue manager with concurrency and bandwidth settings.

## 13. Nexus Browsing Inside DMM

- Vortex capability: tight Nexus integration for discovering, installing, and tracking mods from the site.
- DMM status: relies on Deck browser links or pasted URLs.
- User impact: medium. External browsing keeps DMM lean but adds Steam Deck browser/Flatpak friction.
- Architectural fit: provider search/browse endpoints can remain backend-owned; Svelte only renders results.
- Decision needed: Nexus search page inside DMM, lightweight in-Deck approval flow, or keep external browser as the primary flow.

## 14. Extension Ecosystem

- Vortex capability: game extensions and reviewed community support evolve independently of the core app.
- DMM status: no extension API yet; current code should stay conservative and Go-first. Import URL resolution now goes through a registered catalog resolver boundary, and shared catalog DTOs exist for files/download links, but download-link resolution and install planning are still Nexus/Stardew-specific in the MVP.
- User impact: medium over time, low for MVP.
- Architectural fit: start with internal interfaces and only externalize once two or three real game handlers prove the shape.
- Decision needed: when to freeze an extension API, and what security model applies to third-party handlers on Steam Deck.
