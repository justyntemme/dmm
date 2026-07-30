package skyrimse

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
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
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{NexusDomain},
		VortexGameID: VortexGameID,
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterInstallerChoice(sdk.InstallerChoiceSpec{
		ID:         "vortex:skyrimse:fomod",
		Name:       "FOMOD installer",
		Kind:       "fomod",
		ModType:    "skyrimse-data-root",
		TargetRoot: "Data",
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
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "FNIS",
		Name:               "Fores New Idles in Skyrim",
		ExecutableRelative: "GenerateFNISForUsers.exe",
		RequiredFiles:      []string{"GenerateFNISForUsers.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "bodyslide",
		Name:               "BodySlide",
		ExecutableRelative: "Data/CalienteTools/BodySlide/BodySlide.exe",
		RequiredFiles:      []string{"Data/CalienteTools/BodySlide/BodySlide.exe"},
	})
	r.RegisterPluginActivation(gamebryo.PluginActivation(gamebryo.PluginActivationOptions{
		ID:                   "skyrimse-gamebryo-plugins",
		Name:                 "Skyrim Special Edition plugins.txt activation",
		AppDataPath:          "Skyrim Special Edition",
		Format:               gamebryo.FormatFallout4,
		NativePlugins:        nativePlugins(),
		SupportsLightPlugins: true,
	}))
	r.RegisterMerge(sdk.MergeSpec{ID: "bethesda-merge-mods", Name: "Bethesda plugin/mod merge support"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bethesda-plugin-load-order", Name: "Bethesda plugin load order"})
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
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
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
	}
}
