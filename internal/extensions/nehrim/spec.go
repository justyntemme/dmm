package nehrim

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID       = "1014940"
	OblivionSteamID  = "22330"
	VortexGameID     = "nehrim"
	Name             = "Nehrim: At Fate's Edge"
	executable       = "Oblivion.exe"
	launcher         = "NehrimLauncher.exe"
	dataRoot         = "data"
	blockedModTypeID = "nehrim-cross-app-data"
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
	r.RegisterModType(installplan.ModTypeSpec{
		ID:      blockedModTypeID,
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex resolves Nehrim's deployment root to the Oblivion Steam app install path while detecting Nehrim app 1014940. DMM needs a reusable cross-app Steam root resolver before this can deploy safely.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:nehrim:data",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           blockedModTypeID,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        dataRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Nehrim deployment must target the Oblivion app 22330 install path, not the selected Nehrim app 1014940 path. Add a generic extension-declared cross-app root resolver before enabling this installer.",
		Status:            sdk.CapabilityStatusBlocked,
		Message:           "Blocked until DMM supports Vortex's cross-app Nehrim-to-Oblivion game-root resolution.",
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "nehrim-launcher",
		Name:               "Nehrim Launcher",
		ExecutableRelative: launcher,
		RequiredFiles:      []string{launcher},
		Relative:           true,
		Exclusive:          true,
		Status:             sdk.CapabilityStatusMetadata,
		Message:            "Vortex exposes Nehrim Launcher as an exclusive supported tool.",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "nehrim-cross-app-root",
		Name:    "Nehrim cross-app Steam root",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Add a reusable extension capability that maps one detected Steam app to another app's game root, then enable Nehrim's data-root installer with copy deployment.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-nehrim extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-nehrim/src",
	})
}
