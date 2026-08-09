package skyrimvr

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SteamAppID   = "611670"
	VortexGameID = "skyrimvr"
	NexusDomain  = "skyrimspecialedition"
	Name         = "Skyrim VR"

	dataFolderModType = "skyrimvr-data-folder"
	dataRootModType   = "skyrimvr-data-root"
	scriptExtModType  = "skyrimvr-script-extender"
	dinputModType     = "dinput"
	eslEnablerLib     = "skyrimvresl.dll"
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
		ExecutableRelative:  "SkyrimVR.exe",
		RequiredFiles:       []string{"SkyrimVR.exe"},
		QueryModPath:        "Data",
		MergeMode:           sdk.GameMergeModeAll,
		CompatibleDownloads: []string{NexusDomain, "skyrimse"},
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
		ID:                "vortex:skyrimvr:script-extender",
		VortexInstallerID: "script-extender-installer",
		Name:              "Skyrim Script Extender VR",
		ModType:           scriptExtModType,
		LoaderExecutable:  "sksevr_loader.exe",
		ToolID:            "sksevr",
	}))
	r.RegisterInstaller(gamebryo.ESLEnablerInstaller(gamebryo.ESLEnablerInstallerOptions{
		ID:                "vortex:skyrimvr:esl-enabler",
		VortexInstallerID: "skyvr-esl-enabler",
		GameName:          Name,
		ModType:           dinputModType,
		LibraryFile:       eslEnablerLib,
	}))
	r.RegisterInstaller(sharedmodtypes.DInputInstaller("vortex:skyrimvr:dinput", 50))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "skyrimvr-sksevr-installed",
		Name:          "Skyrim Script Extender VR",
		ModType:       scriptExtModType,
		Message:       "Skyrim Script Extender VR files are not deployed to the Skyrim VR install folder. SKSEVR mods will not load until SKSEVR is installed and deployed.",
		OKMessage:     "Skyrim Script Extender VR files are present in the Skyrim VR install folder.",
		HelpURL:       "https://skse.silverlock.org/",
		InstallHint:   "Install SKSEVR through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"sksevr_loader.exe", "SkyrimVR.exe"},
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:skyrimvr:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     dataRootModType,
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("skse", "SkyProc Patchers"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "sksevr",
		Name:               "Skyrim Script Extender VR",
		ExecutableRelative: "sksevr_loader.exe",
		RequiredFiles:      []string{"sksevr_loader.exe", "SkyrimVR.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{scriptExtModType},
		ProviderModTypes:   []string{scriptExtModType},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "TES5VREdit",
		Name:               "TES5VREdit",
		ExecutableRelative: "TES5VREdit.exe",
		RequiredFiles:      []string{"TES5VREdit.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "FNIS",
		Name:               "Fores New Idles in Skyrim",
		ShortName:          "FNIS",
		ExecutableRelative: "GenerateFNISForUsers.exe",
		RequiredFiles:      []string{"GenerateFNISForUsers.exe"},
		Relative:           true,
	})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{
		ID:                   "skyrimvr-gamebryo-plugins",
		Name:                 "Skyrim VR plugins.txt activation",
		GameID:               VortexGameID,
		AppDataPath:          "Skyrim VR",
		Format:               gamebryo.FormatAsterisked,
		LOOTGameID:           VortexGameID,
		LOOTMasterlistGameID: "skyrimse",
		LOOTPrelude:          true,
		NativePlugins:        []string{"skyrim.esm", "update.esm", "dawnguard.esm", "hearthfires.esm", "dragonborn.esm", "skyrimvr.esm"},
	})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Skyrim VR",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Skyrim.ini"},
			{Name: "SkyrimVR.ini"},
			{Name: "SkyrimPrefs.ini"},
		},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "skyrimvr-exe-version",
		Name:     "SkyrimVR.exe file version",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Skyrim VR archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "skyrimvr-archive-invalidation",
			Name:        "Skyrim VR archive invalidation",
			MyGamesPath: "Skyrim VR",
			ININame:     "SkyrimVR.ini",
			DataRoot:    "Data",
		}),
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "skyrimvr-dynamic-esl-support",
		Name:    "Skyrim VR dynamic ESL support",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex toggles Skyrim VR ESL plugin support when an enabled mod has the eslEnabler attribute. DMM records the attribute but still needs a generic metadata-driven plugin-activation toggle.",
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
	version, err := peversion.FileVersion(filepath.Join(gamePath, "SkyrimVR.exe"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	if version != "" {
		version += "-VR"
	}
	return sdk.GameVersionResult{Version: version, Source: "SkyrimVR.exe"}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex Skyrim VR game extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-skyrimvr/src"},
		{Name: "Vortex script-extender installer game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts"},
		{Name: "Vortex Gamebryo plugin management game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts"},
		{Name: "Vortex Gamebryo archive invalidation game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts"},
		{Name: "Vortex local game settings support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src"},
	}
}
