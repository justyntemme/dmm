# Information Architecture

## Surfaces

### Decky Plugin

Purpose: Steam Deck control, no-phone workflows, pairing, settings, and urgent actions.

Primary areas:

- Quick Access launcher/status panel.
- Full DMM route with tabs:
  - `Actions`
  - `Games`
  - `Settings`
- Game-scoped mod management after selecting a game.
- Game-scoped launch action from the game page where Steam/Decky capability is available.
- Modal workflows:
  - Nexus browser
  - Installer choices/FOMOD
  - Phone pairing QR
  - Confirm destructive actions

### Phone/Tablet Web App

Purpose: primary management surface for browsing, organizing, approving choices, updates, rollback, and profile work.

Primary areas:

- Action Center landing.
- Game drawer with favorites/search/sort.
- Game workspace:
  - `Mods`
  - `Add`
  - `Activity`
  - `Review`
  - `Advanced`
- Global settings:
  - Sources/accounts
  - Security/pairing status
  - Downloads/cache

## Navigation Model

Phone/tablet:

- Hamburger opens game drawer from any game workspace.
- Gear opens global settings.
- The selected game appears in the top-left context area.
- The selected profile appears directly under game context.
- Game module tabs sit below the game/profile header.
- Action Center is the default landing page and is always one tap away.

Decky:

- The Quick Access panel stays compact and launches the full route.
- Full route uses Decky's native route/tab behavior instead of nested scroll-heavy layouts.
- `Actions` is the default route. There is no standalone `Home` route in the current design.
- `Actions` shows the currently running game context at the top when detected.
- `Games` auto-opens the currently running game when detected; otherwise it starts at the game list.
- The selected game page is a clean command surface: launch, explore/import, then the mod list.
- B returns from a selected game page to the game list; do not spend space on a `Change Game` button.
- The selected game page uses a profile dropdown with existing profiles and `Add New Profile` at the bottom.
- Game review diagnostics appear only as compact warning rows when something needs action, not as a standing tab.
- Healthy diagnostics are hidden from the default game page.
- `Settings` owns server controls, security, pairing, source credentials, and a `Show Debug` toggle.
- Debug tools live inside Settings and are hidden until the user enables `Show Debug`.
- B backs out one internal level when possible.
- Search/filter rows must be focusable before list rows.

## Ownership Boundaries

- Backend owns state, decisions, and business logic.
- Web/Decky own layout, interaction, and intent submission.
- Decky owns Steam/overlay-only capabilities such as Steam launch options and Steam Workshop actions.
- UI code should not inspect archive internals, provider rules, deployment maps, or extension-specific logic.
