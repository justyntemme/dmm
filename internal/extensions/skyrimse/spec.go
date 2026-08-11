package skyrimse

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/fnis"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SteamAppID   = "489830"
	VortexGameID = "skyrimse"
	NexusDomain  = "skyrimspecialedition"
	Name         = "Skyrim Special Edition"
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
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{NexusDomain},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "SkyrimSE.exe",
		RequiredFiles:      []string{"SkyrimSE.exe"},
		QueryModPath:       "Data",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
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
		ID:            "skyrimse-skse64-installed",
		Name:          "Skyrim Script Extender 64",
		ModType:       "skyrimse-script-extender",
		Message:       "Skyrim Script Extender files are not deployed to the Skyrim Special Edition install folder. SKSE mods will not load until SKSE64 is installed and deployed.",
		OKMessage:     "Skyrim Script Extender 64 files are present in the Skyrim Special Edition install folder.",
		HelpURL:       "https://skse.silverlock.org/",
		InstallHint:   "Install SKSE64 through the Nexus Mod Manager Download flow, then enable it in the selected profile.",
		RequiredFiles: []string{"skse64_loader.exe", "SkyrimSE.exe"},
	}))
	r.RegisterExtensionTest(gamebryo.ScriptExtenderErrorTest(gamebryo.ScriptExtenderErrorTestOptions{
		ID:   "skyrimse-skse64-plugin-errors",
		Name: "Skyrim Script Extender 64 plugin errors",
		Logs: []gamebryo.ScriptExtenderLogSpec{{
			Base:     gamebryo.ScriptExtenderLogBaseProtonDocuments,
			MyGames:  "Skyrim Special Edition",
			Relative: "SKSE/skse64.log",
		}},
		Plugins: []string{"SKSE/Plugins"},
	}))
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:          "vortex:skyrimse:fomod",
		Name:        "FOMOD installer",
		Kind:        "fomod",
		ModType:     "skyrimse-data-root",
		TargetRoot:  "Data",
		StopFolders: gamebryo.StopFolders("skse", "SkyProc Patchers"),
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "skse64",
		Name:               "Skyrim Script Extender 64",
		ExecutableRelative: "skse64_loader.exe",
		RequiredFiles:      []string{"skse64_loader.exe", "SkyrimSE.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{"skyrimse-script-extender"},
		ProviderModTypes:   []string{"skyrimse-script-extender"},
	})
	r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
		ID:       "skyrimse-skse64-loader",
		Name:     "Existing Skyrim Script Extender 64 loader",
		Patterns: []string{"skse64_loader.exe"},
	})
	r.RegisterUnmanagedMarker(sdk.UnmanagedMarkerSpec{
		ID:       "skyrimse-plugin-list",
		Name:     "Existing Skyrim Special Edition plugin list",
		Patterns: []string{"plugins.txt", "loadorder.txt"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "SSEEdit",
		Name:               "SSEEdit",
		ExecutableRelative: "SSEEdit.exe",
		RequiredFiles:      []string{"SSEEdit.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "WryeBash",
		Name:               "Wrye Bash",
		ExecutableRelative: "Wrye Bash.exe",
		RequiredFiles:      []string{"Wrye Bash.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "FNIS",
		Name:               "Fores New Idles in Skyrim",
		ExecutableRelative: "GenerateFNISForUsers.exe",
		RequiredFiles:      []string{"GenerateFNISForUsers.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "bodyslide",
		Name:               "BodySlide",
		ExecutableRelative: "Data/CalienteTools/BodySlide/BodySlide.exe",
		RequiredFiles:      []string{"Data/CalienteTools/BodySlide/BodySlide.exe"},
		Variants: []sdk.SupportedToolVariantSpec{{
			ID:                 "bodyslide-x64",
			Name:               "BodySlide x64",
			ExecutableRelative: "Data/CalienteTools/BodySlide/BodySlide x64.exe",
			RequiredFiles:      []string{"Data/CalienteTools/BodySlide/BodySlide x64.exe"},
		}},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "creation-kit-64",
		Name:               "Creation Kit",
		ExecutableRelative: "CreationKit.exe",
		RequiredFiles:      []string{"CreationKit.exe"},
	})
	fnis.RegisterSupport(r, fnis.SupportOptions{GameID: VortexGameID, NexusSection: "skyrimspecialedition", NexusModID: "3038", PatchListName: "PatchListSE.txt"})
	gamebryo.RegisterPluginActivation(r, gamebryo.PluginActivationOptions{
		ID:            "skyrimse-gamebryo-plugins",
		Name:          "Skyrim Special Edition plugins.txt activation",
		GameID:        VortexGameID,
		AppDataPath:   "Skyrim Special Edition",
		Format:        gamebryo.FormatAsterisked,
		LOOTGameID:    VortexGameID,
		LOOTPrelude:   true,
		NativePlugins: nativePlugins(),
		NativePluginManifests: []string{
			"Skyrim.ccc",
		},
		SupportsLightPlugins: true,
		ArchiveCheckType:     "BSA",
		ArchiveCheckVersions: []int{105},
	})
	r.RegisterProfileFeature(gamebryo.LocalLOOTRulesProfileFeature())
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:      VortexGameID,
		MyGamesPath: "Skyrim Special Edition",
		SaveININame: "Skyrim.ini",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Skyrim.ini"},
			{Name: "SkyrimPrefs.ini"},
			{Name: "SkyrimCustom.ini", Optional: true},
		},
		FilePatches: gamebryo.ArchiveInvalidationProfilePatches(gamebryo.ArchiveInvalidationOptions{ININame: "Skyrim.ini"}),
	})
	gamebryo.RegisterSkyrimFontSettingsTest(r, gamebryo.SkyrimFontSettingsOptions{GameID: VortexGameID})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "skyrimse-exe-version",
		Name:     "SkyrimSE.exe file version",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Skyrim Special Edition archive invalidation settings",
		Handler: gamebryo.ArchiveInvalidationHandler(gamebryo.ArchiveInvalidationOptions{
			ID:          "skyrimse-archive-invalidation",
			Name:        "Skyrim Special Edition archive invalidation",
			MyGamesPath: "Skyrim Special Edition",
			ININame:     "Skyrim.ini",
			DataRoot:    "Data",
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "skyrimse-data-folder", TargetRoot: ""},
		{ID: "skyrimse-data-root", TargetRoot: "Data"},
		{ID: "skyrimse-script-extender", TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		gamebryo.ScriptExtenderInstaller(gamebryo.ScriptExtenderInstallerOptions{
			ID:                "vortex:skyrimse:script-extender",
			VortexInstallerID: "script-extender-installer",
			Name:              "Skyrim Script Extender 64 (SKSE64)",
			ModType:           "skyrimse-script-extender",
			LoaderExecutable:  "skse64_loader.exe",
			ToolID:            "skse64",
		}),
		{
			ID:                "vortex:skyrimse:data-folder",
			VortexInstallerID: "game-query-mod-path:data-folder",
			Priority:          50,
			ModType:           "skyrimse-data-folder",
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RequireTopLevelDirs: []string{"Data"},
			},
			InstructionMode: installplan.InstructionRootFolder,
		},
		{
			ID:                "vortex:skyrimse:data-root",
			VortexInstallerID: "game-query-mod-path",
			Priority:          100,
			ModType:           "skyrimse-data-root",
			NameSource:        installplan.NameSourceArchive,
			StripCommonRoot:   true,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
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
	version, err := peversion.FileVersion(filepath.Join(gamePath, "SkyrimSE.exe"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: version, Source: "SkyrimSE.exe"}, nil
}

func nativePlugins() []string {
	return []string{
		"skyrim.esm",
		"update.esm",
		"dawnguard.esm",
		"hearthfires.esm",
		"dragonborn.esm",
		"ccbgssse002-exoticarrows.esl",
		"ccbgssse003-zombies.esl",
		"ccbgssse004-ruinsedge.esl",
		"ccbgssse006-stendarshammer.esl",
		"ccbgssse007-chrysamere.esl",
		"ccbgssse010-petdwarvenarmoredmudcrab.esl",
		"ccbgssse014-spellpack01.esl",
		"ccbgssse019-staffofsheogor.esl",
		"ccbgssse021-lordsmail.esl",
		"ccmtysse001-knightsofthenine.esl",
		"ccqdrsse001-survivalmode.esl",
		"cctwbsse001-puzzledungeon.esm",
		"cceejsse001-hstead.esl",
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Skyrim Special Edition game registration",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-skyrimse/src/index.js",
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
			Name: "Vortex Gamebryo settings tests",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-test-settings/src",
		},
		{
			Name: "Vortex script extender installer",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/script-extender-installer/src/installer.ts",
		},
	}
}
