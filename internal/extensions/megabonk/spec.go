package megabonk

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "3405340"
	VortexGameID = "megabonk"
	Name         = "Megabonk"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Megabonk has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. The manifest page describes a BepInEx-or-MelonLoader choice, loader-specific plugin roots, Unity asset placement, and root fallback behavior, so DMM blocks archive installs until those rules are implemented through extension-owned choices.",
		RequiredFiles: []string{
			"Megabonk.x86_64",
			"GameAssembly.so",
			"Megabonk_Data/globalgamemanagers",
		},
		RequirementMessage:     "The Megabonk game folder is missing files needed for future extension support.",
		RequirementOKMessage:   "The Megabonk game folder contains the expected native Linux executable and Unity data folder.",
		RequirementInstallHint: "Verify Megabonk files in Steam before testing Megabonk mods.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex central extension manifest entry site-mod-1495-file-7663", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
			{Name: "Megabonk Vortex extension page", URL: "https://www.nexusmods.com/site/mods/1495"},
			{Name: "Checked Vortex bundled game extensions; no Megabonk source found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
			{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
