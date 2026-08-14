package blasphemous

import (
	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "774361"
	VortexGameID = "blasphemous"
	Name         = "Blasphemous"
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
			"Blasphemous.exe",
			"Blasphemous_Data/globalgamemanagers",
		},
		NativeExecutableMarkers: []string{
			"Blasphemous.x86_64",
			"Blasphemous_Data/globalgamemanagers",
		},
		RuntimeMarkers: []string{
			"BepInEx/core/BepInEx.dll",
			"BepInEx/core/BepInEx.Core.dll",
			"BepInEx/core/BepInEx.Preloader.dll",
			"BepInEx/core/BepInEx.Preloader.Core.dll",
			"run_bepinex.sh",
			"winhttp.dll",
		},
		NativeLinuxLaunchTool: true,
		RuntimeInstallHint:    "Install the BepInEx Unity runtime for the detected Blasphemous platform, then enable and deploy it from DMM before enabling Blasphemous BepInEx plugin mods. Native Linux installs require run_bepinex.sh to be executable and configured as the Steam launch target.",
		Sources: []sdk.SourceRef{
			{Name: "Vortex shared BepInEx extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-bepinex"},
			{Name: "Blasphemous Nexus BepInEx plugin install instructions", URL: "https://www.nexusmods.com/blasphemous/mods/1"},
			{Name: "BepInEx native Unix Steam launch documentation", URL: "https://docs.bepinex.dev/articles/advanced/steam_interop.html"},
			{Name: "Live Steam Deck native executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		},
	})
}
