package davethediver

import (
	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1868140"
	VortexGameID = "davethediver"
	Name         = "Dave the Diver"
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
			"DaveTheDiver.exe",
			"UnityPlayer.dll",
			"GameAssembly.dll",
			"DaveTheDiver_Data/globalgamemanagers",
		},
		RuntimeMarkers: []string{
			"BepInEx/core/BepInEx.Core.dll",
			"BepInEx/core/BepInEx.Unity.IL2CPP.dll",
			"BepInEx/core/BepInEx.Preloader.Core.dll",
			"winhttp.dll",
		},
		RuntimeInstallHint: "Install the Windows x64 BepInEx IL2CPP runtime for Dave the Diver, then enable and deploy it from DMM before enabling Dave the Diver BepInEx plugin mods.",
		RuntimeHelpURL:     "https://builds.bepinex.dev/projects/bepinex_be",
		Sources: []sdk.SourceRef{
			{Name: "Vortex shared BepInEx extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
			{Name: "Dave the Diver Nexus BepInEx IL2CPP plugin archive path verification", URL: "https://www.nexusmods.com/davethediver"},
			{Name: "Live Steam Deck Windows/Proton executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
