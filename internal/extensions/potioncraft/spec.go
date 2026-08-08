package potioncraft

import (
	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1210320"
	VortexGameID = "potioncraftalchemistsimulator"
	Name         = "Potion Craft: Alchemist Simulator"
	ID           = "potioncraft"
)

func Extension() sdk.Extension {
	return bepinexext.UnityExtension(bepinexext.UnityGameSpec{
		ID:           ID,
		Name:         Name,
		Version:      "1.0.0-dmm.2",
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		WindowsExecutableMarkers: []string{
			"Potion Craft.exe",
			"UnityPlayer.dll",
			"Potion Craft_Data/globalgamemanagers",
		},
		RuntimeInstallHint: "Install the Windows x64 BepInEx runtime for Potion Craft, then enable and deploy it from DMM before enabling Potion Craft BepInEx plugin mods.",
		UnclassifiedReason: "Potion Craft archive layout is not classified by the verified Unity/BepInEx extension rules. DMM supports BepInEx runtime, BepInEx root/config packages, and BepInEx plugin DLL archives; other layouts stay blocked until source-reviewed.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex shared BepInEx extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
			{Name: "Potion Craft Nexus BepInEx plugin archive path verification", URL: "https://www.nexusmods.com/potioncraftalchemistsimulator"},
			{Name: "Live Steam Deck Windows/Proton executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
