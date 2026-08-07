package hollowknight

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "367520"
	VortexGameID = "hollowknight"
	Name         = "Hollow Knight"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Hollow Knight has a verified Vortex extension manifest entry, but the current extension package/source has not yet been inspected. The manifest page describes automatic BepInEx installation, managed DLL replacements, Unity asset placement, plugin folders, and root fallback behavior, so DMM blocks archive installs until those installers are source-verified.",
		RequiredFiles: []string{
			"hollow_knight.exe",
			"hollow_knight_Data/globalgamemanagers",
			"hollow_knight_Data/Managed",
		},
		RequirementMessage:     "The Hollow Knight game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Hollow Knight game folder contains the expected executable and Unity data folders.",
		RequirementInstallHint: "Verify Hollow Knight files in Steam before testing Hollow Knight mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-376-file-7365", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Hollow Knight Vortex extension page", URL: "https://www.nexusmods.com/site/mods/376"},
			{Name: "Checked Vortex bundled game extensions; no Hollow Knight source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
