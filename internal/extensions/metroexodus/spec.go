package metroexodus

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1449560"
	VortexGameID = "metroexodus"
	Name         = "Metro Exodus"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Metro Exodus has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. DMM blocks archive installs until representative Metro archive layouts and Vortex installer rules are verified for the Steam Deck Enhanced Edition install.",
		RequiredFiles: []string{
			"MetroExodus.exe",
			"content.vfx",
			"content_03.vfs0",
		},
		RequirementMessage:     "The Metro Exodus game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Metro Exodus game folder contains the expected executable and VFS archive layout.",
		RequirementInstallHint: "Verify Metro Exodus files in Steam before testing Metro Exodus mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-907-file-8800", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Metro Exodus Vortex extension page", URL: "https://www.nexusmods.com/site/mods/907"},
			{Name: "Checked Vortex bundled game extensions; no Metro Exodus source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
