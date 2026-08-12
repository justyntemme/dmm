package fallout4vr

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SteamAppID   = "611660"
	VortexGameID = "fallout4vr"
	NexusDomain  = "fallout4"
	Name         = "Fallout 4 VR"

	dataFolderModType = "fallout4vr-data-folder"
	dataRootModType   = "fallout4vr-data-root"
	scriptExtModType  = "fallout4vr-script-extender"
	dinputModType     = "dinput"
	eslEnablerLib     = "Daytripper4.dll"
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
		SteamAppIDs:         []string{SteamAppID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  "Fallout4VR.exe",
		RequiredFiles:       []string{"Fallout4VR.exe"},
		QueryModPath:        "Data",
		MergeMode:           sdk.GameMergeModeAll,
		CompatibleDownloads: []string{NexusDomain},
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, modType := range gamebryo.DataRootModTypes(dataRootInstallerOptions()) {
		r.RegisterModType(modType)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: scriptExtModType, TargetRoot: ""})
	r.RegisterModType(sharedmodtypes.DInputModTypeSpec())
	for _, installer := range gamebryo.DataRootInstallers(dataRootInstallerOptions()) {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstaller(gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{
		ID:                "vortex:fallout4vr:script-extender",
		VortexInstallerID: "script-extender-installer",
		Name:              "F4SE VR",
		ModType:           scriptExtModType,
		LoaderExecutable:  "f4sevr_loader.exe",
		ToolID:            "F4SEVR",
	}))
	r.RegisterInstaller(gamebryo.ESLEnablerInstaller(gamebryo.ESLEnablerInstallerOptions{
		ID:                "vortex:fallout4vr:esl-enabler",
		VortexInstallerID: "fallout4vr-esl-enabler",
		GameName:          Name,
		ModType:           dinputModType,
		LibraryFile:       eslEnablerLib,
	}))
	r.RegisterInstaller(sharedmodtypes.DInputInstaller("vortex:fallout4vr:dinput", 50))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "fallout4vr-f4sevr-installed",
		Name:          "F4SE VR",
		ModType:       scriptExtModType,
		Message:       "F4SE VR files are not deployed to the Fallout 4 VR install folder. F4SE VR mods will not load until F4SE VR is installed and deployed.",
		OKMessage:     "F4SE VR files are present in the Fallout 4 VR install folder.",
		HelpURL:       "https://f4se.silverlock.org/",
		InstallHint:   "Install F4SE VR through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"f4sevr_loader.exe", "Fallout4VR.exe"},
	}))
	r.RegisterExtensionTest(gamebryo.ArchiveBackdateTest(gamebryo.ArchiveBackdateOptions{
		ID:        "fallout4vr-archive-backdate",
		Name:      "Fallout 4 VR archive timestamp backdate",
		Extension: ".ba2",
		Prefixes:  []string{"fallout4 - ", "fallout4_vr - "},
		TargetAge: time.Date(2008, 11, 1, 0, 0, 0, 0, time.Local),
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:fallout4vr:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     dataRootModType,
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("f4se"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "F4SEVR",
		Name:               "F4SE VR",
		ExecutableRelative: "f4sevr_loader.exe",
		RequiredFiles:      []string{"f4sevr_loader.exe", "Fallout4VR.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{scriptExtModType},
		ProviderModTypes:   []string{scriptExtModType},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "FO4VREdit",
		Name:               "FO4VREdit",
		ExecutableRelative: "FO4VREdit.exe",
		RequiredFiles:      []string{"FO4VREdit.exe"},
	})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{
		ID:                   "fallout4vr-gamebryo-plugins",
		Name:                 "Fallout 4 VR plugins.txt activation",
		GameID:               VortexGameID,
		AppDataPath:          "Fallout4VR",
		Format:               gamebryo.FormatAsterisked,
		LOOTGameID:           VortexGameID,
		LOOTMasterlistGameID: "fallout4",
		LOOTPrelude:          true,
		NativePlugins:        []string{"fallout4.esm", "fallout4_vr.esm"},
		LightPluginsCondition: &sdk.PluginActivationMetadataConditionSpec{
			MetadataKind:     "vortex-attribute",
			MetadataName:     "eslEnabler",
			MetadataUniqueID: "true",
		},
		ArchiveCheckType:     "BA2",
		ArchiveCheckVersions: []int{1},
	})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Fallout4VR",
		SaveININame: "Fallout4Custom.ini",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Fallout4Custom.ini"},
			{Name: "Fallout4Prefs.ini"},
		},
		FilePatches: gamebryo.ArchiveInvalidationProfilePatches(gamebryo.ArchiveInvalidationOptions{ININame: "Fallout4Custom.ini"}),
	})
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "fallout4vr-persistent-subgraph-offsets",
		Name:     "Fallout 4 VR persistent subgraph offsets",
		Patterns: []string{"**/PersistantSubgraphInfoAndOffsetData.txt"},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "fallout4vr-exe-version",
		Name:     "Fallout4VR.exe file version",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Fallout 4 VR archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "fallout4vr-archive-invalidation",
			Name:        "Fallout 4 VR archive invalidation",
			MyGamesPath: "Fallout4VR",
			ININame:     "Fallout4Custom.ini",
			DataRoot:    "Data",
		}),
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "Refresh Fallout 4 VR plugin metadata",
		Handler: didDeployRefreshPluginMetadata,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func didDeployRefreshPluginMetadata(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"Fallout 4 VR plugin metadata refreshed from extension-managed deployment state."}}, nil
}

func dataRootInstallerOptions() gamebryo.DataRootInstallerOptions {
	return gamebryo.DataRootInstallerOptions{
		GameID:            VortexGameID,
		DataFolderModType: dataFolderModType,
		DataRootModType:   dataRootModType,
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	version, err := peversion.FileVersion(filepath.Join(gamePath, "Fallout4VR.exe"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	if version != "" {
		version += "-VR"
	}
	return sdk.GameVersionResult{Version: version, Source: "Fallout4VR.exe"}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex Fallout 4 VR game extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-fallout4vr/src"},
		{Name: "Vortex script-extender installer game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts"},
		{Name: "Vortex Gamebryo plugin management game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts"},
		{Name: "Vortex Gamebryo archive invalidation game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts"},
		{Name: "Vortex local game settings support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src"},
	}
}
