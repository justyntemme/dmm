# Rome: Total War Extension Notes

## Identity

- Steam AppIDs: `4760` (Rome: Total War), `4770` (Rome: Total War - Alexander)
- DMM extension ID: `rometotalwar`
- Nexus domain: `rometotalwar`

## Verified Sources

- Nexus API game list verified the `rometotalwar` domain.
- Steam Deck installed app snapshot: `extensionTargets.md#installed-games-snapshot`
- Checked bundled Vortex game extension source; no reviewed Rome: Total War handler was found.
- Representative Nexus pages verify direct data-folder replacement layouts:
  - Vanilla Rome archives can drop files such as `export_descr_buildings.txt` into `Rome Total War Gold/data`.
  - Alexander archives can extract an entire `data` folder into `Rome Total War Alexander/alexander`.

## Current DMM Capability

- DMM maps both Rome (`4760`) and Alexander (`4770`) to the shared Nexus domain.
- DMM supports narrow data-folder replacement archives.
- For Rome, supported files deploy under `data/`.
- For Alexander, the same archive shape deploys under `alexander/data/`.
- Unclassified launcher, executable, full-conversion, and mod-folder layouts remain blocked until source-reviewed.

## Beta Gaps

- Live-test one vanilla Rome data replacement and one Alexander data replacement.
- Add full-conversion launcher/mod-folder support only after the launch and profile semantics are verified.
- Confirm unmanaged data-file replacement conflicts are presented through DMM's normal conflict/advanced deployment flow.
