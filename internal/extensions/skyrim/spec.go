package skyrim

import (
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/fnis"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "72850"
	VortexGameID = "skyrim"
	Name         = "Skyrim"

	dataFolderModType = "skyrim-data-folder"
	dataRootModType   = "skyrim-data-root"
	scriptExtModType  = "skyrim-script-extender"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "TESV.exe",
		RequiredFiles:      []string{"TESV.exe"},
		QueryModPath:       "Data",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
		Workshop:           sdk.SteamWorkshopSpec{AllowCoexistence: true, Actions: sdk.StandardSteamWorkshopActions()},
	})
	for _, modType := range gamebryo.DataRootModTypes(dataRootInstallerOptions()) {
		r.RegisterModType(modType)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: scriptExtModType, TargetRoot: ""})
	for _, installer := range gamebryo.DataRootInstallers(dataRootInstallerOptions()) {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstaller(gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{ID: "vortex:skyrim:script-extender", VortexInstallerID: "script-extender-installer", Name: "Skyrim Script Extender", ModType: scriptExtModType, LoaderExecutable: "skse_loader.exe", ToolID: "skse"}))
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "skyrim-skse-installed",
		Name:          "Skyrim Script Extender",
		ModType:       scriptExtModType,
		Message:       "Skyrim Script Extender files are not deployed to the Skyrim install folder. SKSE mods will not load until SKSE is installed and deployed.",
		OKMessage:     "Skyrim Script Extender files are present in the Skyrim install folder.",
		HelpURL:       "http://skse.silverlock.org/",
		InstallHint:   "Install SKSE through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"skse_loader.exe", "TESV.exe"},
	}))
	r.RegisterExtensionTest(gamebryo.ScriptExtenderErrorTest(gamebryo.ScriptExtenderErrorTestOptions{
		ID:   "skyrim-skse-plugin-errors",
		Name: "Skyrim Script Extender plugin errors",
		Logs: []gamebryo.ScriptExtenderLogSpec{{
			Base:     gamebryo.ScriptExtenderLogBaseProtonDocuments,
			MyGames:  "Skyrim",
			Relative: "SKSE/skse.log",
		}},
		Plugins: []string{"SKSE/Plugins"},
	}))
	r.RegisterExtensionTest(gamebryo.ArchiveBackdateTest(gamebryo.ArchiveBackdateOptions{
		ID:        "skyrim-archive-backdate",
		Name:      "Skyrim archive timestamp backdate",
		Extension: ".bsa",
		Prefixes:  []string{"skyrim - ", "hearthfires", "dragonborn", "highrestexturepack"},
		TargetAge: time.Date(2008, 11, 1, 0, 0, 0, 0, time.Local),
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{ID: "vortex:skyrim:fomod", Name: "FOMOD installer", Kind: "fomod", ModType: dataRootModType, TargetRoot: "Data", StopFolders: gamebryo.StopFolders("skse", "SkyProc Patchers")})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{ID: "skse", Name: "Skyrim Script Extender", ExecutableRelative: "skse_loader.exe", RequiredFiles: []string{"skse_loader.exe", "TESV.exe"}, DefaultPrimary: true, ModTypes: []string{scriptExtModType}, ProviderModTypes: []string{scriptExtModType}})
	registerSupportedTools(r)
	fnis.RegisterSupport(r, fnis.SupportOptions{GameID: VortexGameID, NexusSection: "skyrim", NexusModID: "11811", PatchListName: "PatchList.txt"})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{ID: "skyrim-gamebryo-plugins", Name: "Skyrim plugins.txt activation", GameID: VortexGameID, AppDataPath: "Skyrim", Format: gamebryo.FormatOriginal, LOOTGameID: VortexGameID, LOOTPrelude: true, NativePlugins: []string{"skyrim.esm", "update.esm"}, ArchiveCheckType: "BSA", ArchiveCheckVersions: []int{104, 103}})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "skyrim",
		SaveININame: "Skyrim.ini",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Skyrim.ini"},
			{Name: "SkyrimPrefs.ini"},
		},
		FilePatches: gamebryo.ArchiveInvalidationProfilePatches(gamebryo.ArchiveInvalidationOptions{ININame: "Skyrim.ini"}),
	})
	gamebryo.RegisterSkyrimFontSettingsTest(r, gamebryo.SkyrimFontSettingsOptions{GameID: VortexGameID})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Skyrim archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "skyrim-archive-invalidation",
			Name:        "Skyrim archive invalidation",
			MyGamesPath: "skyrim",
			ININame:     "Skyrim.ini",
			DataRoot:    "Data",
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func dataRootInstallerOptions() gamebryo.DataRootInstallerOptions {
	return gamebryo.DataRootInstallerOptions{GameID: VortexGameID, DataFolderModType: dataFolderModType, DataRootModType: dataRootModType}
}

func registerSupportedTools(r sdk.Registrar) {
	for _, tool := range []sdk.SupportedToolSpec{
		{ID: "TES5Edit", Name: "TES5Edit", ExecutableRelative: "TES5Edit.exe", RequiredFiles: []string{"TES5Edit.exe"}},
		{ID: "WryeBash", Name: "Wrye Bash", ExecutableRelative: "Wrye Bash.exe", RequiredFiles: []string{"Wrye Bash.exe"}},
		{ID: "FNIS", Name: "Fores New Idles in Skyrim", ShortName: "FNIS", ExecutableRelative: "GenerateFNISForUsers.exe", RequiredFiles: []string{"GenerateFNISForUsers.exe"}, Relative: true},
		{ID: "skse", Name: "Skyrim Script Extender", ShortName: "SKSE", ExecutableRelative: "skse_loader.exe", RequiredFiles: []string{"skse_loader.exe", "TESV.exe"}, Relative: true, Exclusive: true, DefaultPrimary: true},
		{ID: "bodyslide", Name: "BodySlide", ExecutableRelative: "Data/CalienteTools/BodySlide/BodySlide.exe", RequiredFiles: []string{"Data/CalienteTools/BodySlide/BodySlide.exe"}, Relative: true, Variants: []sdk.SupportedToolVariantSpec{{
			ID:                 "bodyslide-x64",
			Name:               "BodySlide x64",
			ExecutableRelative: "Data/CalienteTools/BodySlide/BodySlide x64.exe",
			RequiredFiles:      []string{"Data/CalienteTools/BodySlide/BodySlide x64.exe"},
		}}},
	} {
		r.RegisterSupportedTool(tool)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex Skyrim game extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-skyrim/src"},
		{Name: "Vortex script-extender installer game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/script-extender-installer/src/gameSupport.ts"},
		{Name: "Vortex Gamebryo plugin management game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-plugin-management/src/util/gameSupport.ts"},
		{Name: "Vortex Gamebryo archive invalidation game support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts"},
		{Name: "Vortex local game settings support", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src"},
		{Name: "Vortex Gamebryo settings tests", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-test-settings/src"},
	}
}
