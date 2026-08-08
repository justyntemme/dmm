# View Templates v0.1

These are schematics, not final code. Each template should be accepted or revised before implementation.

## Phone: Action Center Landing

Purpose: first screen after pairing. Shows what needs the user's attention.

```
┌────────────────────────────────────┐
│ DMM                Games     Gear  │
│ Paired · Deck online · LAN only    │
├────────────────────────────────────┤
│ Action Center                      │
│ ┌ Needs Choices ────────────────┐  │
│ │ Mod name        Nexus   Game  │  │
│ │ Short reason                  │  │
│ │ [Continue] [Later]            │  │
│ └───────────────────────────────┘  │
│ ┌ Conflicts / Failed / Notices ─┐  │
│ │ ...                           │  │
│ └───────────────────────────────┘  │
├────────────────────────────────────┤
│ Ready Games                        │
│ Stardew Valley        5 on / 8     │
│ Fallout 4             Review       │
└────────────────────────────────────┘
```

Rules:

- No empty blank state. If no actions exist, show `All clear` plus recently managed games.
- Action cards show source pill, game, profile if relevant, and one primary next action.
- Completed history is secondary and collapsible.

## Phone: Game Drawer

Purpose: fast game selection from any page.

```
┌ Game Drawer ───────────────────────┐
│ Search games                       │
│ Sort: Recent ▼     Manage Ready ✓  │
├────────────────────────────────────┤
│ ★ Stardew Valley       Ready       │
│ ★ Fallout 4            Review      │
│   Final Fantasy VII    Ready       │
│   Witcher 3            Ready       │
└────────────────────────────────────┘
```

Rules:

- Favorites pin above sort order.
- Manage Ready filter is prominent, because metadata-only games are not normal install targets.
- Game rows show capability badges: Nexus, Workshop, Installer, Load Order, Launch.

## Phone: Game Workspace Shell

Purpose: stable frame for all game work.

```
┌────────────────────────────────────┐
│ [Game icon] Stardew Valley    Gear │
│ Default Profile ▼   5 enabled      │
├ Mods │ Add │ Activity │ Review │ Advanced ┤
│                                    │
│ Selected tab content               │
└────────────────────────────────────┘
```

Rules:

- Profile selector is always visible inside game context.
- Game tabs do not mix global settings.
- `Mods` is the default tab.

## Phone: Mods Tab

Purpose: ordinary profile-first management.

```
┌ Mods · Default Profile ────────────┐
│ Search mods        Sort: Profile ▼ │
│ [ ] Show advanced states           │
├────────────────────────────────────┤
│ Enabled                            │
│ ┌ Mod Menu              Nexus  ⏻ │ │
│ │ Current · Update available       │
│ │ [Update] [More]                 │
│ └─────────────────────────────────┘│
│ Disabled                           │
│ ┌ CJB Cheats           Nexus  ○  │ │
│ │ Installed · disabled             │
│ │ [Enable] [More]                 │
│ └─────────────────────────────────┘│
└────────────────────────────────────┘
```

Rules:

- Enable/disable is the primary control.
- Newly installed mods default disabled unless the Deck setting enables otherwise.
- Remove is behind `More` with confirmation.
- Reinstall/reconfigure/check updates are secondary actions.

## Phone: Add Tab

Purpose: add mods to the selected game/profile.

```
┌ Add Mods ──────────────────────────┐
│ Install to: Default Profile ▼      │
│                                    │
│ Explore Sources                    │
│ [Nexus Mods]                       │
│ Future sources appear here         │
│                                    │
│ Paste or Open on Deck              │
│ [Nexus page URL / nxm link] [Open] │
│                                    │
│ ▸ Import Mod Archive               │
└────────────────────────────────────┘
```

Rules:

- Nexus HTTPS page URLs open the controlled Deck browser; direct non-premium API download is not presented as primary.
- Local archive browser stays hidden until the accordion opens.
- The selected profile is explicit before starting install.

## Phone: Installer Choices

Purpose: FOMOD and extension-owned multi-choice installers.

```
┌ Installer Choices ─────────────────┐
│ Mod name                 Nexus     │
│ Step 2 of 4 · Default Profile      │
├────────────────────────────────────┤
│ Visual style                       │
│ ○ Option A                         │
│   Description from installer       │
│ ● Option B                         │
│   Description from installer       │
├────────────────────────────────────┤
│ [Back]                 [Continue]  │
└────────────────────────────────────┘
```

Rules:

- Wizard-style steps, not one giant form.
- Option descriptions are shown when available.
- `Later` saves progress and returns item to Action Center.
- Required/unavailable states are visible but backend validation remains authority.

## Phone: Review Tab

Purpose: explain why a game or profile needs attention.

```
┌ Review ────────────────────────────┐
│ Game Status: Ready / Review        │
│ Runtime: SMAPI ok                  │
│ Launch: configured                 │
│ Workshop: coexists                 │
│ Unsupported archive: reason        │
└────────────────────────────────────┘
```

Rules:

- Review gives remediation, not raw diagnostics only.
- Extension capability coverage is visible here.

## Phone: Advanced Tab

Purpose: power-user tools without confusing normal users.

```
┌ Advanced ──────────────────────────┐
│ Managed Files                      │
│ Strategy: Symlink                  │
│ Conflicts                          │
│ Recovery                           │
│ Restore Points                     │
│ Purge / Reset                      │
└────────────────────────────────────┘
```

Rules:

- File operations, deployment strategy, purge, repair, and restore point details live here.
- Destructive actions require confirmation.

## Tablet Layout

Purpose: same concepts with useful side-by-side space.

```
┌ Game Drawer ┐ ┌ Game Workspace ──────────────────────┐
│ Search      │ │ Header + profile                     │
│ Favorites   │ │ Tabs                                 │
│ Recent      │ │ Main content          Side inspector │
└─────────────┘ └──────────────────────────────────────┘
```

Rules:

- Persistent left game drawer on tablet width.
- Main content and side inspector avoid long single-column scrolling.
- Action Center can use a split list/detail layout.

## Decky: Quick Access Panel

Purpose: compact status and launch controls.

```
┌ Decky Mod Manager ────────────────┐
│ Server: Running                   │
│ Phone: paired / QR available      │
│ URL: 192.168.x.x:17942            │
│ [Open DMM]                        │
└───────────────────────────────────┘
```

Rules:

- No dense mod management here.
- Start/stop and pairing state are visible.

## Decky: Full Route

Purpose: controller-first Deck management.

```
Actions | Games | Settings

Actions:
  Running game context when detected
  Attention items
  Active downloads / installs
  Installer choices
  Failed actions / retry

Games:
  Auto-open running game when detected
  Search
  Sort / Favorites
  Game rows
  Selected game -> Explore Mods / Import Archive direct actions
  Selected game -> installed/profile mod list
  Launch Game

Settings:
  Server controls
  QR phone pairing
  Security toggles
  Source credentials
  Show Debug toggle
  Debug tools, visible only after Show Debug is enabled
```

Rules:

- Native Decky tabs, no nested scroll traps.
- Rows are one focus target; A activates primary action; Y/Menu expose secondary actions.
- Every long row wraps before clipping.
- Binary settings use switch/toggle rows, not side-by-side buttons.
- There is no standalone `Home` tab in the current Decky design.
- There is no selected-game segmented control in the Decky redesign.
- Review diagnostics are compact attention rows only when something needs action.
- Restore/recovery is not shown in the Decky redesign until it is reworked and reliable.

## Decky: Phone Pairing QR Modal

Purpose: add a phone securely without typing a token.

```
┌ Pair Phone ───────────────────────┐
│ Scan this QR from your phone      │
│ ┌──────── QR ────────┐            │
│ │ DMM URL + key      │            │
│ └────────────────────┘            │
│ LAN only: On                     │
│ [Reset Pairing] [Close]          │
└───────────────────────────────────┘
```

Rules:

- QR contains the URL and pairing key.
- Reset invalidates existing phone access.
- Manual URL remains visible as fallback.
