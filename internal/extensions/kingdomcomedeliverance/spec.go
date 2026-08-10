package kingdomcomedeliverance

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "379430"
	VortexGameID = "kingdomcomedeliverance"
	Name         = "Kingdom Come: Deliverance"

	steamExecutable = "Bin/Win64/KingdomCome.exe"
	modsRoot        = "Mods"
	modOrderFile    = "Mods/mod_order.txt"
	modType         = "kingdomcomedeliverance-mod"
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
		ExecutableRelative: steamExecutable,
		RequiredFiles:      []string{"Data/Levels/rataje/level.pak"},
		QueryModPath:       modsRoot,
		MergeMode:          sdk.GameMergeModeDynamic,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:kingdomcomedeliverance:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "kingdomcomedeliverance-mod-order",
		Name:           "Kingdom Come mod_order.txt",
		TargetRelative: modOrderFile,
		TargetRoot:     modsRoot,
		ModTypes:       []string{modType},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Generate Kingdom Come mod_order.txt",
		Handler: willDeploy,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidPurge,
		Name:    "Preserve manual Kingdom Come mod_order.txt entries",
		Handler: didPurge,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "kingdomcomedeliverance-ensure-mods-folder",
		Name:    "Ensure Kingdom Come Mods folder exists",
		Actions: sdk.EnsureGameDirectories(modsRoot),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "kingdomcomedeliverance-xbox-launcher",
		Name:     "Xbox launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    "DeepSilver.KingdomComeDeliverance",
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex uses Xbox launcher metadata for the Microsoft Store version. DMM's current Steam Deck target uses the Steam executable path.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "kingdomcomedeliverance-epic-launcher",
		Name:     "Epic launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    "Eel",
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex uses Epic launcher metadata for the Epic version. DMM's current Steam Deck target uses the Steam executable path.",
	})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{
		ID:      "kingdomcomedeliverance-load-order-page",
		Name:    "Kingdom Come Load Order",
		Scope:   "game",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes mod_order.txt order through generic extension load-order profile controls. Vortex collection import/export remains tracked as a separate blocked collection feature.",
	})
	r.RegisterCollectionFeature(sdk.CollectionFeatureSpec{
		ID:      "kingdomcomedeliverance-collection-data",
		Name:    "Kingdom Come collection data",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex serializes Kingdom Come load-order collection data; DMM needs collection import/export runtime before this feature can run.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-kingdomcome-deliverance extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-kingdomcome-deliverance/src",
	}}
}
