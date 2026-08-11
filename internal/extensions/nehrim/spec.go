package nehrim

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID      = "1014940"
	OblivionSteamID = "22330"
	VortexGameID    = "nehrim"
	Name            = "Nehrim: At Fate's Edge"
	executable      = "Oblivion.exe"
	launcher        = "NehrimLauncher.exe"
	dataRoot        = "data"
	targetRootID    = "nehrim-oblivion-root"
	dataModTypeID   = "nehrim-data"
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
		ExecutableRelative: executable,
		RequiredFiles:      []string{launcher, executable},
		QueryModPath:       dataRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": OblivionSteamID},
		Deployment: installplan.DeploymentSpec{
			DefaultStrategy:       installplan.DeployStrategyCopy,
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       targetRootID,
		Name:     "Oblivion Steam install root",
		Resolver: targetroots.SteamAppInstallRoot(OblivionSteamID),
	})
	r.RegisterModType(installplan.ModTypeSpec{
		ID:           dataModTypeID,
		TargetRoot:   dataRoot,
		TargetRootID: targetRootID,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:nehrim:data",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           dataModTypeID,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        dataRoot,
		TargetRootID:      targetRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "nehrim-launcher",
		Name:               "Nehrim Launcher",
		ExecutableRelative: launcher,
		RequiredFiles:      []string{launcher},
		Relative:           true,
		Exclusive:          true,
		Status:             sdk.CapabilityStatusReady,
		Message:            "Vortex exposes Nehrim Launcher as an exclusive supported tool. DMM exposes the same relative executable through the generic extension-tool runtime.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-nehrim extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-nehrim/src",
	})
}
