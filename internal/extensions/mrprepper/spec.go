package mrprepper

import (
	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "761830"
	VortexGameID = "mrprepper"
	Name         = "Mr. Prepper"
)

func Extension() sdk.Extension {
	return bepinexext.UnityExtension(bepinexext.UnityGameSpec{
		ID:           VortexGameID,
		Name:         Name,
		Version:      "1.0.0-dmm.2",
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		WindowsExecutableMarkers: []string{
			"MrPrepper.exe",
			"MrPrepper_Data/globalgamemanagers",
		},
		RuntimeInstallHint: "Install the Windows x64 BepInEx 5 runtime for Mr. Prepper, then enable and deploy it from DMM before enabling Mr. Prepper BepInEx plugin mods.",
		UnclassifiedReason: "Mr. Prepper archive layout is not classified by the verified Unity/BepInEx extension rules. DMM supports BepInEx runtime, BepInEx root/config packages, and BepInEx plugin DLL archives; other layouts stay blocked until source-reviewed.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex shared BepInEx extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
			{Name: "Mr. Prepper Nexus BepInEx plugin instructions", URL: "https://www.nexusmods.com/mrprepper/mods/1?tab=description"},
			{Name: "Live Steam Deck Windows/Proton executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
