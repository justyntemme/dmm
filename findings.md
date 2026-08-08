# Findings

## Resolved: Direct Nexus File Installs From DMM Need Browser-Generated Credentials

### Observation

End-to-end Stardew Valley testing proved that direct file installs from DMM's Nexus file-list path cannot be the primary non-premium Nexus workflow. The backend correctly reports that some Nexus files require a browser-generated credential.

### Why This Matters

The Nexus API key is not sufficient for every file download path. For at least some Nexus downloads, Nexus still expects the browser-generated `nxm://` flow or a short-lived download credential produced by clicking the Mod Manager Download button on the Nexus page.

### Impact

This means DMM search can discover mods, but the install handoff must open the real Nexus mod page and let the user click Nexus' own Mod Manager Download / Vortex link so DMM receives the short-lived `nxm://` credential.

### Required Investigation

- Keep the DMM browse/search UI on the validated flow: search results open the Nexus mod page in the controlled DMM browser.
- Do not reintroduce a primary `Show Files` / direct API download path for non-premium Nexus downloads.
- Direct API download attempts may remain as an advanced/update fallback only when they report browser-required cleanly.

### Current Priority

Resolved as architecture direction. Remaining work is UI polish and live validation of the browser-generated `nxm://` path for more games/FOMODs.

## Resolved: Use DMM's Controlled BrowserView For Nexus Credential Capture

### Observation

Opening Nexus pages in the Steam in-built browser failed to dispatch `nxm://` links. The working flow is DMM's controlled BrowserView opened from the Decky plugin, which captures the Nexus credential path successfully.

### Why This Matters

This keeps discovery and credential capture inside the DMM flow instead of relying on Firefox copy/paste or the Steam browser's unsupported scheme handling.

### Required Investigation

- Keep the primary flow as `Explore Mods` -> Nexus search result -> controlled Nexus page -> Nexus Mod Manager Download.
- After a successful captured download starts, close the BrowserView and return the user to the selected game's DMM page.
- Keep Firefox as a fallback/debug path, not the primary UX.

### Current Priority

Resolved as architecture direction. Remaining work is more live testing and clearer troubleshooting logs when BrowserView capture fails.

## Obsolete: Nexus File List Ordering

### Observation

DMM no longer uses a primary `Show Files` list in the Decky Nexus workflow.

### Expected Behavior

If a file-list UI returns later as an advanced view, newest files should appear first by default.

### Required Fix

- Sort descending by upload timestamp when available.
- If upload timestamp is unavailable, sort by file ID descending.
- Keep this out of the primary install path unless we have a premium/API-supported flow.

## Current Direction

The primary credential strategy is DMM-controlled Nexus page browsing from Decky. Any future direct file API work must fit that strategy and must fail with a clean browser-required message for non-premium Nexus accounts.

## Active: Phone/Tablet Nexus Page Paste Is A Deck Browser Handoff

### Observation

Pasting a plain Nexus HTTP/HTTPS mod page into the phone/tablet UI does not produce the browser-generated `nxm://` key on the phone. The backend can resolve the Nexus page identity, but it cannot download non-premium Nexus files from that page URL alone.

### Required UX

- Phone/tablet `Add URL(s)` may directly capture real `nxm://` links and non-Nexus provider URLs that expose supported download URLs.
- Phone/tablet Nexus page URLs must be presented as `Open on Deck` handoffs, not as completed installs.
- The Decky plugin executes the controlled BrowserView handoff; the user clicks Nexus Mod Manager Download there, and DMM captures the generated `nxm://` link.
- Keep a retryable phone/tablet handoff prompt so a missed Deck event does not leave the user stuck.

### Current State

The backend browser-open event now has a bounded 10-minute lifetime instead of a 45-second window. A single pasted Nexus page in the phone/tablet Add Mod card goes straight to the Deck browser handoff and does not create a placeholder captured-install job. The Add Mod card keeps a retryable `Finish on Steam Deck` prompt whenever a Nexus page needs the Deck browser handoff.

## Resolved: Neural Harvest Is A Source Package, Not A Vortex Build Step

### Observation

Nexus reports `stardewvalley/mods/32817/files/128820` as the only main file for Neural Harvest, with filename `Neural Harvest-32817-1-0-0-1743448848.zip` and size `1350 KB`. DMM downloaded that exact file. The cached archive is a valid zip and passes `unzip -t`.

### Verified Contents

- The archive contains `fomod/ModuleConfig.xml`.
- The archive contains `build.sh` and `build_vortex.sh`.
- The archive does not contain `script.cs`, so it is not a scripted C# FOMOD.
- The archive does not contain any `.dll` files.
- The archive does not contain the `Common/` folder required by its own FOMOD XML.

### Vortex Behavior

Vortex supports normal XML FOMOD installers and C# scripted FOMOD installers through its native/IPC FOMOD engines. Those engines consume the extracted file list and return copy/generate/attribute instructions. Vortex game extensions can also register custom archive installers. The Vortex source does not indicate that a normal FOMOD archive executes arbitrary `build.sh` or `build_vortex.sh` scripts during install.

### Conclusion

Neural Harvest's uploaded Nexus file appears to be the source tree that should have been processed by `build_vortex.sh` before upload. DMM should not auto-run build scripts from arbitrary mod archives. The correct DMM behavior is to block before showing installer choices and explain that the installer metadata references missing files.
