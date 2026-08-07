package thebindingofisaacrebirth

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "250900"
	VortexGameID = "thebindingofisaacrebirth"
	Name         = "The Binding of Isaac: Rebirth"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "The Binding of Isaac: Rebirth has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. DMM blocks archive installs until Afterbirth+/Repentance mod-folder and resources handling are verified and represented as extension-owned installers.",
		RequiredFiles: []string{
			"isaac-ng.exe",
			"resources/packed",
			"resources/scripts",
		},
		RequirementMessage:     "The Binding of Isaac game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Binding of Isaac game folder contains the expected executable and resources folders.",
		RequirementInstallHint: "Verify The Binding of Isaac: Rebirth files in Steam before testing Isaac mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-516-file-4127", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "The Binding of Isaac Vortex extension page", URL: "https://www.nexusmods.com/site/mods/516"},
			{Name: "Checked Vortex bundled game extensions; no Binding of Isaac source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
