package fallout4

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SteamAppID               = "377160"
	VortexGameID             = "fallout4"
	FalloutLondonNexusDomain = "fallout4london"
	Name                     = "Fallout 4"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID, FalloutLondonNexusDomain},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  "Fallout4.exe",
		RequiredFiles:       []string{"Fallout4.exe"},
		QueryModPath:        "Data",
		MergeMode:           sdk.GameMergeModeAll,
		CompatibleDownloads: []string{FalloutLondonNexusDomain},
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirement(gamebryo.ScriptExtenderRuntimeRequirementOptions{
		ID:            "fallout4-f4se-installed",
		Name:          "Fallout 4 Script Extender",
		ModType:       "fallout4-script-extender",
		Message:       "Fallout 4 Script Extender files are not deployed to the Fallout 4 install folder. F4SE mods will not load until F4SE is installed and deployed.",
		OKMessage:     "Fallout 4 Script Extender files are present in the Fallout 4 install folder.",
		HelpURL:       "https://f4se.silverlock.org/",
		InstallHint:   "Install F4SE through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"f4se_loader.exe", "Fallout4.exe"},
	}))
	r.RegisterExtensionTest(gamebryo.ScriptExtenderErrorTest(gamebryo.ScriptExtenderErrorTestOptions{
		ID:   "fallout4-f4se-plugin-errors",
		Name: "Fallout 4 Script Extender plugin errors",
		Logs: []gamebryo.ScriptExtenderLogSpec{{
			Base:     gamebryo.ScriptExtenderLogBaseProtonDocuments,
			MyGames:  "Fallout4",
			Relative: "F4SE/f4se.log",
		}},
		Plugins: []string{"F4SE/Plugins"},
	}))
	r.RegisterExtensionTest(gamebryo.ArchiveBackdateTest(gamebryo.ArchiveBackdateOptions{
		ID:        "fallout4-archive-backdate",
		Name:      "Fallout 4 archive timestamp backdate",
		Extension: ".ba2",
		Prefixes:  []string{"fallout4 - ", "dlccoast - ", "dlcrobot - ", "dlcworkshop", "dlcnukaworld - "},
		TargetAge: time.Date(2008, 11, 1, 0, 0, 0, 0, time.Local),
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:fallout4:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     "fallout4-data-root",
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("f4se"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "f4se",
		Name:               "Fallout 4 Script Extender",
		ExecutableRelative: "f4se_loader.exe",
		RequiredFiles:      []string{"f4se_loader.exe", "Fallout4.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{"fallout4-script-extender"},
		ProviderModTypes:   []string{"fallout4-script-extender"},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "fallout4-xbox-launcher",
		Name:     "Xbox app launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    "BethesdaSoftworks.Fallout4-PC",
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "Game",
		}},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "fallout4-epic-launcher",
		Name:     "Epic Games launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    "61d52ce4d09d41e48800c22784d13ae8",
	})
	r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
		ID:       "fallout4-f4se-loader",
		Name:     "Existing Fallout 4 Script Extender loader",
		Patterns: []string{"f4se_loader.exe"},
	})
	r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
		ID:       "fallout4-plugin-list",
		Name:     "Existing Fallout 4 plugin list",
		Patterns: []string{"plugins.txt", "loadorder.txt"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "FO4Edit",
		Name:               "FO4Edit",
		ExecutableRelative: "FO4Edit.exe",
		RequiredFiles:      []string{"FO4Edit.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "WryeBash",
		Name:               "Wrye Bash",
		ExecutableRelative: "Wrye Bash.exe",
		RequiredFiles:      []string{"Wrye Bash.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "bodyslide",
		Name:               "BodySlide",
		ExecutableRelative: "Data/Tools/BodySlide/BodySlide.exe",
		RequiredFiles:      []string{"Data/Tools/BodySlide/BodySlide.exe"},
		Variants: []sdk.SupportedToolVariantSpec{{
			ID:                 "bodyslide-x64",
			Name:               "BodySlide x64",
			ExecutableRelative: "Data/Tools/BodySlide/BodySlide x64.exe",
			RequiredFiles:      []string{"Data/Tools/BodySlide/BodySlide x64.exe"},
		}},
	})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{
		ID:            "fallout4-gamebryo-plugins",
		Name:          "Fallout 4 plugins.txt activation",
		GameID:        VortexGameID,
		AppDataPath:   "Fallout4",
		Format:        gamebryo.FormatAsterisked,
		LOOTGameID:    VortexGameID,
		LOOTPrelude:   true,
		NativePlugins: nativePlugins(),
		NativePluginManifests: []string{
			"Fallout4.ccc",
		},
		SupportsLightPlugins: true,
		ArchiveCheckType:     "BA2",
		ArchiveCheckVersions: []int{8, 7, 1},
	})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Fallout4",
		SaveININame: "Fallout4Custom.ini",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Fallout4.ini"},
			{Name: "Fallout4Prefs.ini"},
			{Name: "Fallout4Custom.ini", Optional: true},
		},
		FilePatches: gamebryo.ArchiveInvalidationProfilePatches(gamebryo.ArchiveInvalidationOptions{ININame: "Fallout4.ini"}),
	})
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "fallout4-persistent-subgraph-offsets",
		Name:     "Fallout 4 persistent subgraph offsets",
		Patterns: []string{"**/PersistantSubgraphInfoAndOffsetData.txt"},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "fallout4-exe-version",
		Name:     "Fallout4.exe file version",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Fallout 4 archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "fallout4-archive-invalidation",
			Name:        "Fallout 4 archive invalidation",
			MyGamesPath: "Fallout4",
			ININame:     "Fallout4.ini",
			DataRoot:    "Data",
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
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
	version, err := peversion.FileVersion(filepath.Join(gamePath, "Fallout4.exe"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: version, Source: "Fallout4.exe"}, nil
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "fallout4-data-folder", TargetRoot: ""},
		{ID: "fallout4-data-root", TargetRoot: "Data"},
		{ID: "fallout4-script-extender", TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{
			ID:                "vortex:fallout4:script-extender",
			VortexInstallerID: "script-extender-installer",
			Name:              "Fallout 4 Script Extender (F4SE)",
			ModType:           "fallout4-script-extender",
			LoaderExecutable:  "f4se_loader.exe",
			ToolID:            "f4se",
		}),
		{
			ID:                "vortex:fallout4:data-folder",
			VortexInstallerID: "game-query-mod-path:data-folder",
			Priority:          50,
			ModType:           "fallout4-data-folder",
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RequireTopLevelDirs: []string{"Data"},
			},
			InstructionMode: installplan.InstructionRootFolder,
		},
		{
			ID:                "vortex:fallout4:data-root",
			VortexInstallerID: "game-query-mod-path",
			Priority:          100,
			ModType:           "fallout4-data-root",
			NameSource:        installplan.NameSourceArchive,
			StripCommonRoot:   true,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}

func nativePlugins() []string {
	return []string{
		"fallout4.esm",
		"dlcrobot.esm",
		"dlcworkshop01.esm",
		"dlccoast.esm",
		"dlcworkshop02.esm",
		"dlcworkshop03.esm",
		"dlcnukaworld.esm",
		"dlcultrahighresolution.esm",
		"ccbgsfo4001-pipboy(black).esl",
		"ccbgsfo4002-pipboy(blue).esl",
		"ccbgsfo4003-pipboy(camo01).esl",
		"ccbgsfo4004-pipboy(camo02).esl",
		"ccbgsfo4006-pipboy(chrome).esl",
		"ccbgsfo4012-pipboy(red).esl",
		"ccbgsfo4014-pipboy(white).esl",
		"ccbgsfo4016-prey.esl",
		"ccbgsfo4017-mauler.esl",
		"ccbgsfo4018-gaussrifleprototype.esl",
		"ccbgsfo4019-chinesestealtharmor.esl",
		"ccbgsfo4020-powerarmorskin(black).esl",
		"ccbgsfo4022-powerarmorskin(camo01).esl",
		"ccbgsfo4023-powerarmorskin(camo02).esl",
		"ccbgsfo4025-powerarmorskin(chrome).esl",
		"ccbgsfo4038-horsearmor.esl",
		"ccbgsfo4039-tunnelsnakes.esl",
		"ccbgsfo4041-doommarinearmor.esl",
		"ccbgsfo4042-bfg.esl",
		"ccbgsfo4043-doomchainsaw.esl",
		"ccbgsfo4044-hellfirepowerarmor.esl",
		"ccbgsfo4046-tescan.esl",
		"ccbgsfo4096-as_enclave.esl",
		"ccbgsfo4110-ws_enclave.esl",
		"ccbgsfo4115-x02.esl",
		"ccbgsfo4116-heavyflamer.esl",
		"cceejfo4001-decorationpack.esl",
		"ccfrsfo4001-handmadeshotgun.esl",
		"ccfsvfo4001-modularmilitarybackpack.esl",
		"ccfsvfo4002-midcenturymodern.esl",
		"ccfsvfo4007-halloween.esl",
		"ccotmfo4001-remnants.esl",
		"ccsbjfo4003-grenade.esl",
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Fallout 4 game registration",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-fallout4/src/index.js",
		},
		{
			Name: "Vortex Gamebryo plugin activation support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-plugin-management/src/util/gameSupport.ts",
		},
		{
			Name: "Vortex Gamebryo archive invalidation support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-invalidation/src/util/gameSupport.ts",
		},
		{
			Name: "Vortex local game settings support",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/local-gamesettings/src",
		},
		{
			Name: "Vortex script extender installer",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/script-extender-installer/src/installer.ts",
		},
	}
}
