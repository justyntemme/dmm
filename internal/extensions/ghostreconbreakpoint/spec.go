package ghostreconbreakpoint

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2231380"
	VortexGameID = "ghostreconbreakpoint"
	Name         = "Ghost Recon Breakpoint"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Ghost Recon Breakpoint has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. The manifest page describes AnvilToolkit, forge-file repacking, sounddata placement, and manual rename prompts, so DMM blocks archive installs until those workflows are modeled as extension-owned capabilities.",
		RequiredFiles: []string{
			"GRB.exe",
			"GRB_vulkan.exe",
			"DataPC.forge",
			"sounddata/pc",
		},
		RequirementMessage:     "The Ghost Recon Breakpoint game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Ghost Recon Breakpoint game folder contains the expected executables, forge archive, and sounddata layout.",
		RequirementInstallHint: "Verify Ghost Recon Breakpoint files in Steam before testing Breakpoint mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-972-file-7463", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Ghost Recon Breakpoint Vortex extension page", URL: "https://www.nexusmods.com/site/mods/972"},
			{Name: "Checked Vortex bundled game extensions; no Ghost Recon Breakpoint source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
