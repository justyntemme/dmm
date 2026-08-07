package mewgenics

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "686060"
	VortexGameID = "mewgenics"
	Name         = "Mewgenics"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Mewgenics has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. The manifest page describes generated launch commands, load-order-driven script creation, Mewtator/root installs, and mods-folder installs, so DMM blocks archive installs until those lifecycle hooks are implemented in the extension.",
		RequiredFiles: []string{
			"Mewgenics.exe",
			"resources.gpak",
		},
		RequirementMessage:     "The Mewgenics game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Mewgenics game folder contains the expected executable and resources package.",
		RequirementInstallHint: "Verify Mewgenics files in Steam before testing Mewgenics mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-1691-file-8709", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Mewgenics Vortex extension page", URL: "https://www.nexusmods.com/site/mods/1691"},
			{Name: "Checked Vortex bundled game extensions; no Mewgenics source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
