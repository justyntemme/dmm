# Decky Add Source Options

Decision target: how the selected-game page exposes mod browsing/import sources in the Decky plugin.

There is no `Add` segment in the approved Decky redesign. `Explore Mods` and `Import Archive` are direct selected-game page actions.

Current known browse source: Nexus. Other providers may support URL import today, but Nexus is the only in-DMM browse/search source.

## Option A: Direct Nexus Action

```
Add

[Explore Nexus]
Browse Nexus mods for this game.

Mod Link
[Paste Nexus/mod URL]

Import Archive
[Open Downloads]
```

Behavior:

- `Explore Nexus` opens the Nexus browser modal directly.
- Other source support appears only in paste/import flows until those sources gain browse support.

Pros:

- Fastest path for the current real workflow.
- Least visual noise.
- Best controller flow because one button maps to one obvious action.

Cons:

- Future browse providers require renaming/reworking the first action.
- It under-communicates that DMM is multi-source.

Best fit:

- MVP if Nexus remains the only browsable source.

## Option B: Generic Explore Mods Source Picker

```
Add

[Explore Mods]
Choose a source to browse.

Mod Link
[Paste supported URL]

Import Archive
[Open Downloads]
```

Then:

```
Explore Mods

[Nexus]
Ready · Browse supported mods

[Thunderstore]
URL import only

[GameBanana]
URL import only
```

Behavior:

- `Explore Mods` opens a source picker.
- Sources that cannot browse are shown as disabled/info rows.
- Selecting Nexus opens the Nexus browser modal.

Pros:

- Future-proof source model.
- Clearly explains provider capability differences.
- Keeps the Add segment stable as sources expand.

Cons:

- Adds one extra click for the only current browse source.
- Can feel like ceremony while Nexus is the only real browse option.

Best fit:

- Later MVP or post-MVP once at least two sources support browsing.

## Option C: Inline Source List

```
Add

Browse
[Nexus]        Ready
[Thunderstore] URL only
[GameBanana]   URL only

Mod Link
[Paste supported URL]

Import Archive
[Open Downloads]
```

Behavior:

- The Add segment always shows source rows.
- Ready rows are actionable.
- URL-only rows either explain paste support or are non-actionable.

Pros:

- Most transparent about provider capability.
- Avoids a separate source-picker modal.
- Scales to a handful of sources.

Cons:

- Uses precious Decky vertical space.
- Disabled provider rows can make the UI feel cluttered.
- More D-pad stops before reaching paste/import.

Best fit:

- Not ideal for Decky until multiple browse sources are ready.

## Option D: Adaptive Primary Action

```
Add

[Explore Nexus]
Only browse source available.

More Sources
Nexus ready · 6 URL import sources

Mod Link
[Paste supported URL]

Import Archive
[Open Downloads]
```

Behavior:

- When only one browse source exists, show that source directly.
- A secondary `More Sources` row opens provider capability details.
- When multiple browse sources exist, the primary action becomes `Explore Mods`.

Pros:

- Best current UX without painting us into a corner.
- Avoids extra click today.
- Still gives users a place to understand provider support.

Cons:

- Slightly more conditional behavior in UI code.
- The primary action label changes when more browse providers are added.

Best fit:

- Recommended for MVP.

## Option E: Source Tabs Inside Add

```
Add

[Nexus] [Link] [Archive]

Nexus:
[Search Nexus]
Recent / Popular / Updated

Link:
[Paste supported URL]

Archive:
[Open Downloads]
```

Behavior:

- Add has its own internal segmented control.
- Each install path gets a clean dedicated subview.

Pros:

- Very organized.
- Keeps each workflow simple once selected.
- Maps well to future sources if source count is small.

Cons:

- Nested segmented controls inside a selected-game segmented control may feel heavy.
- More controller state to get right.

Best fit:

- Consider after MVP if Add becomes too dense.

## Option F: Unified Results With Source Filters

```
Add

[Explore Mods]
Browse supported sources for this game.

Mod Link
[Paste supported URL]

Import Archive
[Open Downloads]
```

Then:

```
Explore Mods

Search mods
Sort: Popular

Sources
[toggle] Nexus
[toggle] Thunderstore
[toggle] GameBanana

Results
Nexus      Mod Menu
GameBanana Future result
Thunderstore Future result
```

Behavior:

- `Explore Mods` opens one combined browse/search surface.
- Results include every enabled source that supports browsing for the selected game.
- Source filters are multi-select toggles, so users can disable or enable source feeds without leaving the results view.
- Sources that only support URL import do not appear as result feeds until they have a verified browse/search capability.
- Every result row shows a source pill.

Pros:

- Best long-term mental model: users browse mods, then filter by source only when needed.
- Avoids forcing the user to choose a store before searching.
- Keeps future multi-provider browsing in one workflow.

Cons:

- Requires normalized search/sort behavior across providers.
- Needs clear disabled/unavailable states when a provider cannot browse a game.

Best fit:

- Approved target for the Decky redesign.

## Recommendation

Use Option F for the Decky redesign:

- Primary action is `Explore Mods`.
- The browse modal shows all browse-capable sources by default.
- A source filter lets users enable/disable specific sources.
- In the current MVP state, Nexus will likely be the only enabled browse source, but the UX does not have to change when more browse providers are added.
