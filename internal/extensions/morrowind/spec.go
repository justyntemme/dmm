package morrowind

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "22320"
	VortexGameID = "morrowind"
	Name         = "Morrowind"

	dataRoot          = "Data Files"
	dataFolderModType = "morrowind-data-folder"
	dataRootModType   = "morrowind-data-root"
	morrowindINI      = "Morrowind.ini"
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
		ExecutableRelative: "morrowind.exe",
		RequiredFiles:      []string{"morrowind.exe"},
		QueryModPath:       dataRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, modType := range gamebryo.DataRootModTypes(dataRootInstallerOptions()) {
		r.RegisterModType(modType)
	}
	for _, installer := range gamebryo.DataRootInstallers(dataRootInstallerOptions()) {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:morrowind:fomod",
		Name:        "Morrowind FOMOD",
		Kind:        "fomod",
		ModType:     dataRootModType,
		TargetRoot:  dataRoot,
		StopFolders: gamebryo.StopFolders(dataRoot),
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "tes3edit",
		Name:               "TES3Edit",
		ExecutableRelative: "TES3Edit.exe",
		Status:             sdk.CapabilityStatusMetadata,
		Message:            "Vortex exposes TES3Edit as a supported external tool.",
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "mw-construction-set",
		Name:               "Construction Set",
		ExecutableRelative: "TES Construction Set.exe",
		RequiredFiles:      []string{"TES Construction Set.exe"},
		Relative:           true,
		Exclusive:          true,
		Status:             sdk.CapabilityStatusMetadata,
		Message:            "Vortex exposes the Morrowind Construction Set as a supported external tool.",
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "morrowind-ini-load-order", Name: "Morrowind.ini Game Files"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Generate Morrowind.ini plugin list",
		Handler: willDeploy,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "Apply Morrowind plugin timestamps",
		Handler: didDeploy,
	})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{
		ID:      "morrowind-plugins-page",
		Name:    "Morrowind Plugins",
		Scope:   "game",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex exposes a Morrowind plugin page. DMM writes Morrowind.ini from profile priority today; generic plugin drag/drop UI remains to be implemented.",
	})
	r.RegisterCollectionFeature(sdk.CollectionFeatureSpec{
		ID:      "morrowind-collection-data",
		Name:    "Morrowind collection load order",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex serializes Morrowind load-order collection data; DMM needs collection import/export runtime before this feature can run.",
	})
	r.RegisterAttributeExtractor(sdk.AttributeExtractorSpec{
		ID:      "morrowind-plugin-list",
		Name:    "Installed ESP/ESM plugin list",
		Target:  "mods",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex stores discovered ESP/ESM file names as mod attributes after install. DMM derives managed plugin names from deployment mappings for now.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func dataRootInstallerOptions() gamebryo.DataRootInstallerOptions {
	return gamebryo.DataRootInstallerOptions{
		GameID:            VortexGameID,
		DataFolderModType: dataFolderModType,
		DataRootModType:   dataRootModType,
		DataRoot:          dataRoot,
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-morrowind extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-morrowind/src",
		},
		{
			Name: "Vortex Morrowind plugin-management extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/morrowind-plugin-management/src",
		},
	}
}
