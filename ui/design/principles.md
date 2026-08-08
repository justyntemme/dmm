# UI Design Principles

## Design Paradigm

Decky Mod Manager should feel like a calm control center for a single job: get mods into the right profile and make them playable. The UI should hide implementation machinery until the user asks for it.

The Decky plugin rewrite starts from blank schematics. Existing screens may be used to inventory workflows, but they should not define the new layout, spacing, navigation, or component structure.

The product model is:

1. Pick a game.
2. Pick or keep a profile.
3. Add mods.
4. Resolve required choices.
5. Enable or disable mods.
6. Play.

## Apple-Like Product Rules

- One primary action per screen section.
- State is written in user language: `Installed`, `Enabled`, `Disabled`, `Needs choices`, `Conflict`, `Update available`.
- The UI explains what needs attention, not how the filesystem pipeline works.
- Important state is visible before controls. Controls should answer the user's next obvious question.
- Advanced controls are grouped and labeled as advanced, not scattered through normal flows.
- Avoid crowded rows. A row may wrap to two lines before clipping text.
- Tap targets and Decky focus targets should be obvious from spacing and highlight state.
- Empty states should be useful: show the next action, not a debug blank.
- Binary settings use Apple-inspired toggle rows: label and short effect text on the left, switch on the right, clear on/off state, and no crowded multi-button rows.

## Visual Direction

- Dark, high-contrast, Steam Deck friendly.
- Neutral base with restrained accent colors by state/source, not a one-hue palette.
- Use game art as context where it helps orientation, but do not let artwork reduce legibility.
- Cards are for repeated objects: games, mods, actions, jobs. Sections are full-width layouts, not nested cards.
- Source pills are small and consistent.
- Destructive actions use confirmation and visually distinct danger styling.

## Security UX

- Phone pairing is QR-first from Decky.
- The QR should encode a pairing URL containing the Deck address and pairing key.
- Scanning the QR approves that phone for the current backend token/session.
- The phone stores the token locally and removes token material from the visible URL after pairing.
- Reset Phone Pairing invalidates existing phone access and generates a new QR/pairing key.
- LAN-only remains enabled by default and is explained beside the pairing controls.
