package falloutnv

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "22380"
	SteamAppID2  = "22490"
	VortexGameID = "falloutnv"
	NexusDomain  = "newvegas"
	Name         = "Fallout: New Vegas"

	dataFolderModType = "falloutnv-data-folder"
	dataRootModType   = "falloutnv-data-root"
	scriptExtModType  = "falloutnv-script-extender"
	dinputModType     = "dinput"
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
		SteamAppIDs:        []string{SteamAppID, SteamAppID2},
		NexusDomains:       []string{NexusDomain},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "FalloutNV.exe",
		RequiredFiles:      []string{"FalloutNV.exe"},
		QueryModPath:       "Data",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
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
		ID:                "vortex:falloutnv:script-extender",
		VortexInstallerID: "script-extender-installer",
		Name:              "New Vegas Script Extender",
		ModType:           scriptExtModType,
		LoaderExecutable:  "nvse_loader.exe",
		ToolID:            "nvse",
	}))
	r.RegisterInstaller(fourGBPatchInstaller())
	r.RegisterInstaller(sharedmodtypes.DInputInstaller("vortex:falloutnv:dinput", 50))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "falloutnv-nvse-installed",
		Name:          "New Vegas Script Extender",
		ModType:       scriptExtModType,
		Message:       "New Vegas Script Extender files are not deployed to the Fallout: New Vegas install folder. NVSE mods will not load until NVSE is installed and deployed.",
		OKMessage:     "New Vegas Script Extender files are present in the Fallout: New Vegas install folder.",
		HelpURL:       "https://www.nvse.silverlock.org/",
		InstallHint:   "Install NVSE through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"nvse_loader.exe", "FalloutNV.exe"},
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:falloutnv:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     dataRootModType,
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("nvse"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "nvse",
		Name:               "New Vegas Script Extender",
		ExecutableRelative: "nvse_loader.exe",
		RequiredFiles:      []string{"nvse_loader.exe", "FalloutNV.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{scriptExtModType},
		ProviderModTypes:   []string{scriptExtModType},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "falloutnv-xbox-launcher",
		Name:     "Xbox app launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    "BethesdaSoftworks.FalloutNewVegas",
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "Game",
		}},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "falloutnv-epic-launcher",
		Name:     "Epic Games launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    "5daeb974a22a435988892319b3a4f476",
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "FNVEdit",
		Name:               "FNVEdit",
		ExecutableRelative: "FNVEdit.exe",
		RequiredFiles:      []string{"FNVEdit.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "WryeBash",
		Name:               "Wrye Bash",
		ExecutableRelative: "Wrye Bash.exe",
		RequiredFiles:      []string{"Wrye Bash.exe"},
	})
	r.RegisterPluginActivation(gamebryo.PluginActivation(gamebryo.PluginActivationOptions{
		ID:            "falloutnv-gamebryo-plugins",
		Name:          "Fallout: New Vegas plugins.txt activation",
		AppDataPath:   "falloutnv",
		Format:        gamebryo.FormatOriginal,
		LOOTGameID:    VortexGameID,
		LOOTPrelude:   true,
		NativePlugins: []string{"falloutnv.esm"},
	}))
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Fallout: New Vegas archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "falloutnv-archive-invalidation",
			Name:        "Fallout: New Vegas archive invalidation",
			MyGamesPath: "FalloutNV",
			ININame:     "Fallout.ini",
			DataRoot:    "Data",
			RequiredKeys: map[string]string{
				"bInvalidateOlderFiles":  "1",
				"sArchiveList":           "Fallout - Textures.bsa, Fallout - Textures2.bsa, Fallout - Meshes.bsa, Fallout - Voices1.bsa, Fallout - Sound.bsa, Fallout - Misc.bsa",
				"sResourceDataDirsFinal": "",
			},
		}),
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

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Fallout: New Vegas game extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-falloutnv/src",
		},
		{
			Name: "Vortex script-extender installer game support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts",
		},
		{
			Name: "Vortex Gamebryo plugin management game support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts",
		},
		{
			Name: "Vortex Gamebryo archive invalidation game support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts",
		},
	}
}
