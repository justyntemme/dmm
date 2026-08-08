# Design Approval Tracker

Status values: `pending`, `approved`, `revise`, `rejected`.

| Area | Status | Notes |
| --- | --- | --- |
| Ground-up Decky rewrite | approved | Start from blank schematics; existing UI is feature/API reference only. |
| Design paradigm | approved | Calm profile-first control center, Apple-like progressive disclosure. |
| Apple-inspired toggles | approved | Binary settings use switch rows with label/effect text, not crowded button groups. |
| Phone Action Center landing | deferred | Phone/tablet rewrite comes later; concept is approved but not active in this pass. |
| Phone game drawer | pending | Hamburger, search, favorites, sort, Manage Ready filter. |
| Phone game workspace shell | pending | Game header, profile selector, tabs. |
| Phone Mods tab | pending | Enable/disable first; source pills; update state. |
| Phone Add tab | pending | Nexus source, controlled Deck browser handoff, local archive accordion. |
| Phone installer choices | pending | Wizard steps for FOMOD and extension-owned choices. |
| Phone Review tab | deferred | Separate from Advanced, but phone/tablet work is not active now. |
| Phone Advanced tab | deferred | Separate from Review, but phone/tablet work is not active now. |
| Tablet layout | pending | Persistent game drawer and split content/detail where useful. |
| Decky Quick Access panel | pending | Status, URL, QR/pairing, Open DMM only. |
| Decky full route | approved | `Actions`, `Games`, `Settings`. No standalone Home. Debug is hidden inside Settings until enabled. |
| Decky game launch action | approved | Game page includes `Launch Game` where Steam/Decky capability is available. |
| Decky Actions running-game context | approved | Actions shows the currently running game at the top when detected. |
| Decky Games auto-open running game | approved | Games opens the currently running game when detected; otherwise it starts at the game list. |
| Decky selected-game segmented control | rejected | Removed for Decky. The selected game page is direct actions plus the mod list. |
| Decky Add segment removal | approved | No `Add` segment. Selected game page has direct `Explore Mods` and `Import Archive` actions. |
| Decky paste-link removal | approved | Paste-link import is excluded from the current Decky redesign. |
| Decky restore/recovery visibility | approved | Hide restore/recovery from Decky redesign until the feature is reworked and reliable. |
| Decky QR pairing modal | pending | QR contains DMM URL plus approval key. |
| Decky Add source entry | approved | Use unified `Explore Mods` results with multi-select source filters. |
| Decky clean selected-game page | approved | Launch on first action row, Explore/Import on second action row, mods directly underneath. B returns to game list. |
| Decky profile selector | approved | Active profile is a dropdown with existing profiles and `Add New Profile` at the bottom. |
| Decky warning diagnostics | approved | Only abnormal diagnostics appear as compact game-page warning rows; healthy states are hidden. |

## Resolved Decisions

1. Active redesign target is Decky plugin UI only. Phone/tablet rewrite comes later.
2. Decky route removes Home. Default tabs are `Actions`, `Games`, and `Settings`.
3. Debug lives in Settings behind a `Show Debug` toggle.
4. Game pages include a launch action.
5. Settings use Apple-inspired toggle rows.
6. Actions shows running-game context when detected.
7. Games auto-opens the running game when detected.
8. Selected game pages do not use internal segmented controls.
9. Selected game pages show `Launch Game` first, then `Explore Mods` and `Import Archive`, then the mod list.
10. B returns from selected game to game list; there is no `Change Game` button.
11. There is no `Add` segment.
12. Paste-link import is excluded from the current Decky redesign.
13. Restore/recovery is left out of the Decky UI until reworked.
14. Active profile is a dropdown with `Add New Profile` at the bottom.
15. Healthy diagnostics are hidden; only warnings appear on the game page.

## Next Review Questions

1. Approve the first detailed Decky screen mockup before implementation starts.
