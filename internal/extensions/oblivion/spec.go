package oblivion

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversionhash"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID    = "22330"
	SteamAppIDAlt = "900883"
	VortexGameID  = "oblivion"
	Name          = "Oblivion"

	dataFolderModType = "oblivion-data-folder"
	dataRootModType   = "oblivion-data-root"
	scriptExtModType  = "oblivion-script-extender"
)

var defaultFonts = map[string]string{
	"sfontfile_1": "Data\\Fonts\\Kingthings_Regular.fnt",
	"sfontfile_2": "Data\\Fonts\\Kingthings_Shadowed.fnt",
	"sfontfile_3": "Data\\Fonts\\Tahoma_Bold_Small.fnt",
	"sfontfile_4": "Data\\Fonts\\Daedric_Font.fnt",
	"sfontfile_5": "Data\\Fonts\\Handwritten.fnt",
}

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID, SteamAppIDAlt},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "oblivion.exe",
		RequiredFiles:      []string{"oblivion.exe"},
		QueryModPath:       "Data",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	for _, modType := range gamebryo.DataRootModTypes(dataRootInstallerOptions()) {
		r.RegisterModType(modType)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: scriptExtModType, TargetRoot: ""})
	for _, installer := range gamebryo.DataRootInstallers(dataRootInstallerOptions()) {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstaller(gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{
		ID:                "vortex:oblivion:script-extender",
		VortexInstallerID: "script-extender-installer",
		Name:              "Oblivion Script Extender",
		ModType:           scriptExtModType,
		LoaderExecutable:  "obse_loader.exe",
		ToolID:            "obse",
	}))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "oblivion-obse-installed",
		Name:          "Oblivion Script Extender",
		ModType:       scriptExtModType,
		Message:       "Oblivion Script Extender files are not deployed to the Oblivion install folder. OBSE mods will not load until OBSE is installed and deployed.",
		OKMessage:     "Oblivion Script Extender files are present in the Oblivion install folder.",
		HelpURL:       "https://github.com/llde/xOBSE",
		InstallHint:   "Install OBSE through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"obse_loader.exe", "oblivion.exe"},
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{ID: "vortex:oblivion:fomod", Name: "FOMOD installer", Kind: "fomod", ModType: dataRootModType, TargetRoot: "Data", StopFolders: gamebryo.StopFolders("obse")})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{ID: "obse", Name: "Oblivion Script Extender", ExecutableRelative: "obse_loader.exe", RequiredFiles: []string{"obse_loader.exe"}, DefaultPrimary: true, ModTypes: []string{scriptExtModType}, ProviderModTypes: []string{scriptExtModType}})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "TES4Edit", Name: "TES4Edit", ExecutableRelative: "TES4Edit.exe", RequiredFiles: []string{"TES4Edit.exe"}})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "WryeBash", Name: "Wrye Bash", ExecutableRelative: "Wrye Bash.exe", RequiredFiles: []string{"Wrye Bash.exe"}})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{ID: "obse", Name: "Oblivion Script Extender", ShortName: "OBSE", ExecutableRelative: "obse_loader.exe", RequiredFiles: []string{"obse_loader.exe"}, Relative: true, Exclusive: true})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{ID: "oblivion-gamebryo-plugins", Name: "Oblivion plugins.txt activation", GameID: VortexGameID, AppDataPath: "oblivion", Format: gamebryo.FormatOriginal, LOOTGameID: VortexGameID, LOOTPrelude: true, NativePlugins: []string{"oblivion.esm"}, ArchiveCheckType: "BSA", ArchiveCheckVersions: []int{103}})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Oblivion",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Oblivion.ini"},
		},
	})
	gamebryo.RegisterOblivionFontSettingsTest(r, gamebryo.OblivionFontSettingsOptions{
		GameID:       VortexGameID,
		MyGamesPath:  "Oblivion",
		ININame:      "Oblivion.ini",
		DefaultFonts: defaultFonts,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Oblivion archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "oblivion-archive-invalidation",
			Name:        "Oblivion archive invalidation",
			MyGamesPath: "Oblivion",
			ININame:     "Oblivion.ini",
			DataRoot:    "Data",
			RequiredKeys: map[string]string{
				"bInvalidateOlderFiles":  "1",
				"SArchiveList":           "Oblivion - Meshes.bsa, Oblivion - Textures - Compressed.bsa, Oblivion - Sounds.bsa, Oblivion - Voices1.bsa, Oblivion - Voices2.bsa, Oblivion - Misc.bsa",
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
	return gamebryo.DataRootInstallerOptions{GameID: VortexGameID, DataFolderModType: dataFolderModType, DataRootModType: dataRootModType}
}

func registerStoreMetadata(r sdk.Registrar) {
	r.RegisterGameStore(sdk.GameStoreSpec{ID: "gog", Name: "GOG", Status: sdk.CapabilityStatusMetadata, Message: "Vortex can discover Oblivion through GOG. DMM's current Steam Deck target uses Steam discovery."})
	r.RegisterGameStore(sdk.GameStoreSpec{ID: "xbox", Name: "Xbox", Status: sdk.CapabilityStatusMetadata, Message: "Vortex can discover Oblivion through Xbox and defaults to the English language folder. DMM needs generic store/language selection before enabling this path."})
	r.RegisterGameVersionProvider(gameversionhash.Provider(gameversionhash.Options{ID: "oblivion-hash-version", Name: "Oblivion.esm hash version", VortexGameID: VortexGameID, HashFiles: []string{"Data/Oblivion.esm"}}))
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex Oblivion game extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-oblivion/src"},
		{Name: "Vortex script-extender installer game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts"},
		{Name: "Vortex Gamebryo plugin management game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts"},
		{Name: "Vortex Gamebryo archive invalidation game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts"},
		{Name: "Vortex local game settings support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src"},
		{Name: "Vortex Gamebryo settings tests", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-test-settings/src"},
	}
}
