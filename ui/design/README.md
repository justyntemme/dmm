# Decky Mod Manager UI Design Workspace

This folder is the approval workspace for the MVP UI overhaul. UI code should be derived from approved schematics in this folder, not improvised directly in Svelte or Decky JSX.

The current active design pass is the Decky plugin UI only. Phone/tablet remains an MVP redesign target, but it comes later.

The Decky rewrite starts from a blank product surface. The existing UI is only a reference for features and backend API behavior; it is not the navigation model, layout baseline, spacing system, or component hierarchy for the new design.

## Process

1. Define the design paradigm and product rules.
2. Draft schematic templates for each major view.
3. Review with the user and mark each view `approved`, `revise`, or `rejected`.
4. Convert only approved templates into implementation tasks.
5. Re-check phone, tablet, and Steam Deck controller ergonomics after implementation.

## Current Artifacts

- `principles.md`: product and visual rules for the redesign.
- `information-architecture.md`: navigation model and screen ownership.
- `view-templates.md`: first-pass schematics for phone/tablet and Decky views.
- `decky-plugin.md`: focused ground-up schematic for the current Decky plugin rewrite.
- `approval.md`: review checklist and approval status.

## Non-Negotiables

- Normal users manage profiles and enabled mods. Staging, manifests, file maps, and deployment internals stay advanced.
- Phone and iPad/tablet are first-class. Tablet is not a stretched phone layout.
- Decky views must be controller-first. D-pad focus must move predictably one visible item at a time.
- Source identity is visible on mods, jobs, actions, and update flows.
- Pairing a phone is QR-first: the QR contains the DMM URL plus pairing key needed to approve the phone.
