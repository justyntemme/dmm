package discoelysium

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "632470"
	VortexGameID = "discoelysium"
	Name         = "Disco Elysium"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Disco Elysium has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. The manifest page describes automatic BepInEx setup, data-folder normalization, GameAssembly replacements, and root fallback behavior, so DMM blocks archive installs until those rules are implemented as extension-owned installers.",
		RequiredFiles: []string{
			"disco.exe",
			"GameAssembly.dll",
			"disco_Data/globalgamemanagers",
		},
		RequirementMessage:     "The Disco Elysium game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Disco Elysium game folder contains the expected executable, Unity player data, and IL2CPP runtime marker.",
		RequirementInstallHint: "Verify Disco Elysium files in Steam before testing Disco Elysium mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-1643-file-7265", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Disco Elysium Vortex extension page", URL: "https://www.nexusmods.com/site/mods/1643"},
			{Name: "Checked Vortex bundled game extensions; no Disco Elysium source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
