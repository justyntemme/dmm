package fallout4

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "377160"
	VortexGameID = "fallout4"
	Name         = "Fallout 4"
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
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
		},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
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
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "FO4Edit",
		Name:               "FO4Edit",
		ExecutableRelative: "FO4Edit.exe",
		RequiredFiles:      []string{"FO4Edit.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "WryeBash",
		Name:               "Wrye Bash",
		ExecutableRelative: "Wrye Bash.exe",
		RequiredFiles:      []string{"Wrye Bash.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "bodyslide",
		Name:               "BodySlide",
		ExecutableRelative: "Data/Tools/BodySlide/BodySlide.exe",
		RequiredFiles:      []string{"Data/Tools/BodySlide/BodySlide.exe"},
	})
	r.RegisterPluginActivation(gamebryo.PluginActivation(gamebryo.PluginActivationOptions{
		ID:            "fallout4-gamebryo-plugins",
		Name:          "Fallout 4 plugins.txt activation",
		AppDataPath:   "Fallout4",
		Format:        gamebryo.FormatFallout4,
		NativePlugins: nativePlugins(),
		NativePluginManifests: []string{
			"Fallout4.ccc",
		},
		SupportsLightPlugins: true,
	}))
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "fallout4-persistent-subgraph-offsets",
		Name:     "Fallout 4 persistent subgraph offsets",
		Patterns: []string{"**/PersistantSubgraphInfoAndOffsetData.txt"},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "bethesda-merge-mods", Name: "Bethesda plugin/mod merge support"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bethesda-plugin-load-order", Name: "Bethesda plugin load order"})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
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
	}
}
