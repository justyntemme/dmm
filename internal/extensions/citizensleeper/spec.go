package citizensleeper

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1578650"
	VortexGameID = "citizensleeper"
	Name         = "Citizen Sleeper"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Citizen Sleeper has a verified Vortex extension manifest entry and linked extension repository, but the extension source has not yet been inspected. DMM blocks archive installs until the Unity/BepInEx or root-file layout rules are verified and encoded in this extension.",
		RequiredFiles: []string{
			"Citizen Sleeper.exe",
			"Citizen Sleeper_Data/globalgamemanagers",
			"Citizen Sleeper_Data/Managed",
		},
		RequirementMessage:     "The Citizen Sleeper game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Citizen Sleeper game folder contains the expected Unity executable and data folders.",
		RequirementInstallHint: "Verify Citizen Sleeper files in Steam before testing Citizen Sleeper mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-444-file-1656", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Citizen Sleeper Vortex extension page", URL: "https://www.nexusmods.com/site/mods/444"},
			{Name: "Linked Vortex extension repository from manifest description", URL: "https://github.com/BluesKutya/VortexExtensions"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
