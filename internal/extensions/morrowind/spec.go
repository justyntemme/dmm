package morrowind

import (
	"path/filepath"
	"strings"

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
		Status:             sdk.CapabilityStatusReady,
		Message:            "Vortex exposes TES3Edit as a supported external tool. DMM exposes the same relative executable through the generic extension-tool runtime.",
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "mw-construction-set",
		Name:               "Construction Set",
		ExecutableRelative: "TES Construction Set.exe",
		RequiredFiles:      []string{"TES Construction Set.exe"},
		Relative:           true,
		Exclusive:          true,
		Status:             sdk.CapabilityStatusReady,
		Message:            "DMM discovers Vortex's game-root Construction Set executable when present and can queue it through the Decky extension-tool launch path.",
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "morrowind-ini-load-order",
		Name:           "Morrowind.ini Game Files",
		TargetRelative: morrowindINI,
		TargetRoot:     dataRoot,
		ModTypes:       []string{dataRootModType, dataFolderModType},
		FileExtensions: []string{".esm", ".esp"},
		EntryNameMode:  sdk.LoadOrderEntryNameFileName,
		Message:        "Mirrors Vortex's Morrowind load-order surface by writing the active profile plugin order into Morrowind.ini and applying timestamp ordering during deployment.",
	})
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
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidInstallMod,
		Name:    "Refresh Morrowind plugin metadata after install",
		Handler: didInstallRefreshPluginMetadata,
	})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{
		ID:      "morrowind-plugins-page",
		Name:    "Morrowind Plugins",
		Scope:   "game",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes managed Morrowind.ini plugin order through generic extension load-order profile controls. Collection import/export uses the generic DMM profile collection runtime.",
	})
	r.RegisterCollectionFeature(sdk.CollectionFeatureSpec{
		ID:      "morrowind-collection-data",
		Name:    "Morrowind collection load order",
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex serializes Morrowind load-order collection data; DMM exports and imports installed profile mod identity, enabled state, and order through the generic profile collection runtime.",
	})
	r.RegisterAttributeExtractor(sdk.AttributeExtractorSpec{
		ID:      "morrowind-plugin-list",
		Name:    "Installed ESP/ESM plugin list",
		Target:  "mods",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM extracts ESP/ESM plugin attributes during extension-owned install planning and uses them with deployment mappings for Morrowind.ini load order.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "morrowind-1.0.3-plugin-attribute-migration",
		Name:        "Morrowind plugin attribute migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.3",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:             "scan-plugins",
			Name:           "Scan staged ESP/ESM plugins",
			Command:        sdk.StateMigrationCommandScanStagedFiles,
			MetadataKind:   "morrowind-plugin",
			FileExtensions: []string{".esm", ".esp"},
		}},
		Message: "Mirrors Vortex 1.0.3 migration by scanning historical staged mods for ESP/ESM plugin files and storing equivalent DMM metadata for load-order generation.",
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
		MetadataExtractors: []installplan.MetadataExtractorSpec{
			{
				Kind:           "morrowind-plugin",
				FileExtensions: []string{".esm", ".esp"},
				Parse:          pluginMetadataFromFile,
			},
		},
	}
}

func pluginMetadataFromFile(path string) installplan.ModMetadata {
	name := strings.TrimSpace(filepath.Base(path))
	if !isPluginFile(name) {
		return installplan.ModMetadata{}
	}
	return installplan.ModMetadata{
		Kind:     "morrowind-plugin",
		Name:     name,
		UniqueID: strings.ToLower(name),
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
