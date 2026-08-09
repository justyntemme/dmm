package kerbalspaceprogram

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversionhash"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "220200"
	VortexGameID = "kerbalspaceprogram"
	Name         = "Kerbal Space Program"

	executableLinux   = "KSP.x86_64"
	executableWindows = "KSP_x64.exe"
	gameDataRoot      = "GameData"
	modType           = "kerbalspaceprogram-gamedata"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executableLinux,
		RequiredFiles:      []string{executableLinux},
		QueryModPath:       gameDataRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterInstallPlatform(sdk.InstallPlatformSpec{ID: "kerbalspaceprogram-linux", Name: "Kerbal Space Program Linux", Markers: []string{executableLinux}})
	r.RegisterInstallPlatform(sdk.InstallPlatformSpec{ID: "kerbalspaceprogram-windows", Name: "Kerbal Space Program Windows", Markers: []string{executableWindows}})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: gameDataRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:kerbalspaceprogram:gamedata",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        gameDataRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameVersionProvider(gameversionhash.Provider(gameversionhash.Options{ID: "kerbalspaceprogram-assembly-version", Name: "Kerbal Space Program Assembly-CSharp hash", VortexGameID: VortexGameID, HashFiles: []string{"KSP_x64_Data/Managed/Assembly-CSharp.dll"}}))
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-kerbalspaceprogram extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-kerbalspaceprogram/src",
	})
}
