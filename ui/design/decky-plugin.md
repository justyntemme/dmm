# Decky Plugin Redesign

This is the active design target. It is a ground-up rewrite of the Decky plugin UI from blank schematics, not an iteration on the current layout.

## Navigation

Default tabs:

```
Actions | Games | Settings
```

There is no `Home` tab. The previous Home content has clearer ownership elsewhere:

- Attention items belong in `Actions`.
- Game browsing, current game context, and launch belong in `Games`.
- Server, pairing, auth, provider credentials, and debug visibility belong in `Settings`.

## Quick Access Panel

Purpose: compact status plus entry into the full DMM route.

```
Decky Mod Manager
Server: Running
Phone: Paired / Pair available
URL: 192.168.x.x:17942

[Open DMM]
```

Rules:

- No dense mod management here.
- No multi-row debug tools here.
- Server state and pairing state must be visible.

## Actions Tab

Purpose: everything that currently needs attention.

```
Actions
All clear / 2 need attention

Now playing
Stardew Valley
Default Profile · 5 enabled
Open Game

[Installer choice required]
Mod Name
Stardew Valley · Default Profile · Nexus
Continue

[Download running]
Mod Name
Fallout 4 · Nexus · 42%
Cancel
```

Rules:

- This is the default full-route tab.
- Show the currently running game context at the top when detected.
- The running game context opens that game's management page.
- If empty, show a useful all-clear state and recently managed games.
- Cards use user language: `Needs choices`, `Downloading`, `Installing`, `Failed`, `Ready`.
- Cards are one focus target. `A` performs the primary next action.

## Games Tab

Purpose: select a game, manage mods, add mods, and launch.

List state:

```
Games
Search games
Sort: Recent

★ Stardew Valley
Ready · 5 on / 8 total · Nexus

Fallout 4
Review · FOMOD · Load order
```

Selected-game state:

```
Stardew Valley
Profile: Default Profile ▼
5 enabled / 8 installed

[Launch Game]
[Explore Mods] [Import Archive]

Warning: SMAPI launch option missing

Mod Menu            Enabled     Nexus
CJB Cheats          Disabled    Nexus
```

Rules:

- Auto-open the currently running game when detected.
- If no running supported game is detected, start at the game list.
- B returns from the selected game page to the game list.
- There is no `Change Game` button in the selected game page.
- The profile line is a dropdown. It shows the active profile, lists existing profiles, and includes `Add New Profile` at the bottom.
- Selected-game pages show `Launch Game` on its own line.
- Selected-game pages show direct `Explore Mods` and `Import Archive` actions on the next line.
- Paste-link import is intentionally excluded from the Decky redesign for now.
- Installed/profile mods appear directly below the action buttons.
- There is no selected-game segmented control in the Decky redesign.
- Review diagnostics are not a tab. Only abnormal states appear as compact warning rows on the game page.
- Healthy diagnostics are hidden. For example, do not show `SMAPI configured` when it is fine.
- Advanced/power-user details are not shown on the default Decky game page.
- Search and sort are focusable before game rows.
- Game rows and mod rows visibly highlight on focus.
- Rows wrap to two lines before clipping and never exceed the Decky sidebar width.
- Game page includes `Launch Game` when Steam/Decky capability is available.
- Normal mod management is enable/disable first. Reinstall, reconfigure, remove, and order are secondary.
- Restore/recovery is intentionally excluded from the Decky redesign until the backend and UX are reliable enough to ship.
- `Explore Mods` opens a unified browse modal.
- `Import Archive` opens the Downloads-scoped archive browser/import flow.
- Browse results show all browse-capable sources enabled for the selected game.
- Source filters let the user enable/disable specific result feeds.
- Every result row shows a source pill.
- URL-only sources are not shown in this Decky pass until there is a stronger no-keyboard workflow.

## Settings Tab

Purpose: device settings, security, pairing, and debug visibility.

```
Settings
Server
[toggle] Keep server running

Security
[Pair Phone QR]
[Reset Phone Pairing]
[toggle] LAN only

Automation
[toggle] Auto-install captured downloads
[toggle] Auto-enable installed mods
[toggle] Auto-display installer choices

Debug
[toggle] Show Debug

Debug tools shown only when enabled:
Logs
Build fingerprint
Package update
Diagnostics
```

Rules:

- Binary settings use Apple-inspired toggle rows: label and short effect text on the left, switch on the right.
- Debug tools are hidden until `Show Debug` is enabled.
- Debug is not a default navigation tab.

## Modals

Nexus browser:

- Opens from `Games -> selected game -> Explore Mods`.
- Shows only supported sources first. Nexus is the only browse source today.
- Search, sort, compatibility filter, and close stay reachable without long scrolling.
- Result rows open the real Nexus page in the controlled browser so the user can click Nexus' Mod Manager Download link.

Installer choices/FOMOD:

- Opens from `Actions` or automatically when a Deck-side flow requires choices and auto-display is enabled.
- Wizard-style steps.
- `Later` returns the item to `Actions`.
- Apply/Continue controls stay visible at the bottom of the modal.
