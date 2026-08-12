package fallout3

import (
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversionhash"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppIDGOTY = "22300"
	SteamAppID     = "22370"
	VortexGameID   = "fallout3"
	Name           = "Fallout 3"

	dataFolderModType = "fallout3-data-folder"
	dataRootModType   = "fallout3-data-root"
	scriptExtModType  = "fallout3-script-extender"
	gogAppID          = "1454315831"
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
		SteamAppIDs:        []string{SteamAppIDGOTY, SteamAppID},
		StoreAppIDs:        map[string][]string{"gog": {gogAppID}},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "fallout3.exe",
		ExecutableVariants: []sdk.GameExecutableVariantSpec{{
			ID:                 "fallout3ng",
			Name:               "Fallout 3 New Executable",
			ExecutableRelative: "fallout3ng.exe",
			RequiredFiles:      []string{"fallout3ng.exe", "Data/fallout3.esm"},
		}},
		RequiredFiles: []string{"Data/fallout3.esm"},
		QueryModPath:  "Data",
		MergeMode:     sdk.GameMergeModeAll,
		Environment:   map[string]string{"SteamAPPId": SteamAppIDGOTY},
		Deployment:    installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	for _, modType := range gamebryo.DataRootModTypes(dataRootInstallerOptions()) {
		r.RegisterModType(modType)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: scriptExtModType, TargetRoot: ""})
	for _, installer := range gamebryo.DataRootInstallers(dataRootInstallerOptions()) {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstaller(gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{
		ID:                "vortex:fallout3:script-extender",
		VortexInstallerID: "script-extender-installer",
		Name:              "Fallout Script Extender",
		ModType:           scriptExtModType,
		LoaderExecutable:  "fose_loader.exe",
		ToolID:            "fose",
	}))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "fallout3-fose-installed",
		Name:          "Fallout Script Extender",
		ModType:       scriptExtModType,
		Message:       "Fallout Script Extender files are not deployed to the Fallout 3 install folder. FOSE mods will not load until FOSE is installed and deployed.",
		OKMessage:     "Fallout Script Extender files are present in the Fallout 3 install folder.",
		HelpURL:       "http://fose.silverlock.org/",
		InstallHint:   "Install FOSE through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"fose_loader.exe", "Data/fallout3.esm"},
	}))
	r.RegisterExtensionTest(gamebryo.ScriptExtenderErrorTest(gamebryo.ScriptExtenderErrorTestOptions{
		ID:   "fallout3-fose-plugin-errors",
		Name: "Fallout Script Extender plugin errors",
		Logs: []gamebryo.ScriptExtenderLogSpec{
			{Base: gamebryo.ScriptExtenderLogBaseGame, Relative: "fose.log"},
			{Base: gamebryo.ScriptExtenderLogBaseGame, Relative: "fose_editor.log"},
		},
		Plugins: []string{"FOSE/Plugins"},
	}))
	r.RegisterExtensionTest(gamebryo.ArchiveBackdateTest(gamebryo.ArchiveBackdateOptions{
		ID:        "fallout3-archive-backdate",
		Name:      "Fallout 3 archive timestamp backdate",
		Extension: ".bsa",
		Prefixes:  []string{"fallout - ", "brokensteel - ", "anchorage - ", "pointlookout - ", "zeta - ", "thepitt - "},
		TargetAge: time.Date(2006, 2, 1, 0, 0, 0, 0, time.Local),
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:fallout3:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     dataRootModType,
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("fose"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "fose",
		Name:               "Fallout Script Extender",
		ExecutableRelative: "fose_loader.exe",
		RequiredFiles:      []string{"fose_loader.exe", "Data/fallout3.esm"},
		DefaultPrimary:     true,
		ModTypes:           []string{scriptExtModType},
		ProviderModTypes:   []string{scriptExtModType},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "FO3Edit", Name: "FO3Edit", ExecutableRelative: "FO3Edit.exe", RequiredFiles: []string{"FO3Edit.exe"}})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "WryeBash", Name: "Wrye Bash", ExecutableRelative: "Wrye Bash.exe", RequiredFiles: []string{"Wrye Bash.exe"}})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "fose", Name: "Fallout Script Extender", ShortName: "FOSE", ExecutableRelative: "fose_loader.exe", RequiredFiles: []string{"fose_loader.exe", "Data/fallout3.esm"}, Relative: true, Exclusive: true, DefaultPrimary: true})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{
		ID:                   "fallout3-gamebryo-plugins",
		Name:                 "Fallout 3 plugins.txt activation",
		GameID:               VortexGameID,
		AppDataPath:          "Fallout3",
		Format:               gamebryo.FormatOriginal,
		LOOTGameID:           VortexGameID,
		LOOTPrelude:          true,
		NativePlugins:        []string{"fallout3.esm"},
		ArchiveCheckType:     "BSA",
		ArchiveCheckVersions: []int{104},
	})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Fallout3",
		SaveININame: "Fallout.ini",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Fallout.ini"},
			{Name: "FalloutPrefs.ini"},
			{Name: "FalloutCustom.ini", Optional: true},
		},
		FilePatches: gamebryo.ArchiveInvalidationProfilePatches(gamebryo.ArchiveInvalidationOptions{ININame: "Fallout.ini"}),
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Fallout 3 archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "fallout3-archive-invalidation",
			Name:        "Fallout 3 archive invalidation",
			MyGamesPath: "Fallout3",
			ININame:     "Fallout.ini",
			DataRoot:    "Data",
			RequiredKeys: map[string]string{
				"bInvalidateOlderFiles":  "1",
				"SArchiveList":           "Fallout - Textures.bsa, Fallout - Meshes.bsa, Fallout - Voices.bsa, Fallout - Sound.bsa, Fallout - MenuVoices.bsa, Fallout - Misc.bsa",
				"sResourceDataDirsFinal": "",
			},
		}),
	})
	registerStoreMetadata(r)
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

func registerStoreMetadata(r sdk.Registrar) {
	for _, store := range []sdk.GameStoreSpec{
		{ID: "gog", Name: "GOG", Message: "Vortex can discover Fallout 3 through GOG app 1454315831. DMM indexes the same extension-declared GOG identity for supported SteamOS store-provider discovery."},
		{ID: "epic", Name: "Epic Games", Message: "Vortex can discover Fallout 3 through Epic app adeae8bbfc94427db57c7dfecce3f1d4 and defaults to the English language folder. DMM indexes the same extension-declared Epic identity and language-folder behavior through the Fallout 3 extension."},
		{ID: "xbox", Name: "Xbox", Message: "Vortex can discover Fallout 3 through Xbox app BethesdaSoftworks.Fallout3 and defaults to the English language folder. DMM indexes the same extension-declared Xbox identity and language-folder behavior through the Fallout 3 extension."},
	} {
		r.RegisterGameStore(store)
	}
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{ID: "fallout3-xbox-launcher", Name: "Xbox app launcher", Launcher: "xbox", Store: "xbox", AppID: "BethesdaSoftworks.Fallout3", Parameters: []sdk.LauncherParameterSpec{{Name: "appExecName", Value: "Game"}}, Message: "DMM indexes Vortex's Xbox launcher identity for Fallout 3 from extension metadata so store-backed registrations satisfy the same app identity."})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{ID: "fallout3-epic-launcher", Name: "Epic launcher", Launcher: "epic", Store: "epic", AppID: "adeae8bbfc94427db57c7dfecce3f1d4", Message: "DMM indexes Vortex's Epic launcher identity for Fallout 3 from extension metadata and matches supported Epic manifests through the generic store-provider discovery path."})
	r.RegisterGameVersionProvider(gameversionhash.Provider(gameversionhash.Options{ID: "fallout3-hash-version", Name: "Fallout3.esm hash version", VortexGameID: VortexGameID, HashFiles: []string{"Data/Fallout3.esm"}}))
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex Fallout 3 game extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-fallout3/src"},
		{Name: "Vortex script-extender installer game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts"},
		{Name: "Vortex Gamebryo plugin management game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts"},
		{Name: "Vortex Gamebryo archive invalidation game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts"},
		{Name: "Vortex local game settings support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src"},
	}
}
