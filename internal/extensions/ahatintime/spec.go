package ahatintime

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "253230"
	VortexGameID = "ahatintime"
	Name         = "A Hat in Time"

	modRoot = "HatInTimeGame/Mods"
	modType = "ahatintime-mod"
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
		ExecutableRelative: "Binaries/Win64/HatInTimeGame.exe",
		RequiredFiles:      []string{"Binaries/Win64/HatInTimeGame.exe"},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "HatinTimeEditor",
		Name:               "Modding Tools",
		ExecutableRelative: "Binaries/ModManager.exe",
		RequiredFiles:      []string{"Binaries/ModManager.exe"},
		Relative:           true,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:ahatintime:mod",
		VortexInstallerID: "ahatintime-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModInfoArchive,
		CustomBuild:       buildModInfoArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-ahatintime extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-ahatintime/src",
		},
	}
}
