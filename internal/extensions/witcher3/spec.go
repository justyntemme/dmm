package witcher3

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "292030"
	SteamAppIDDX = "499450"
	VortexGameID = "witcher3"
	Name         = "The Witcher 3"
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
		SteamAppIDs:  []string{SteamAppID, SteamAppIDDX},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "W3ScriptMerger",
		Name:               "W3 Script Merger",
		ExecutableRelative: "WitcherScriptMerger.exe",
		RequiredFiles:      []string{"WitcherScriptMerger.exe"},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "witcher3-xml-menu-merge", Name: "Witcher 3 XML/menu merge"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "witcher3-mods-settings", Name: "Witcher 3 mods.settings load order"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Generate Witcher 3 mods.settings",
		Handler: willDeploy,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "witcher3menumodroot", TargetRoot: ""},
		{ID: "witcher3tl", TargetRoot: ""},
		{ID: "witcher3dlc", TargetRoot: ""},
		{ID: "witcher3-mod-root", TargetRoot: "Mods"},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:witcher3:scriptmergerdummy",
			VortexInstallerID: "scriptmergerdummy",
			Priority:          15,
			Match: installplan.MatchSpec{
				FileBasenames: []string{"WitcherScriptMerger.exe"},
			},
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "The Witcher 3 Script Merger is a tool, not a mod. DMM must install/configure it through the Witcher 3 extension tool flow before script-merge support is complete.",
		},
		{
			ID:                "vortex:witcher3:witcher3tl",
			VortexInstallerID: "witcher3tl",
			Priority:          30,
			ModType:           "witcher3tl",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchTopLevelMod,
			CustomBuild:       buildTopLevelMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:witcher3:witcher3menumodroot",
			VortexInstallerID: "witcher3menumodroot",
			Priority:          20,
			ModType:           "witcher3menumodroot",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchMenuModRoot,
			CustomBuild:       buildMenuModRoot,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:witcher3:witcher3mixed",
			VortexInstallerID: "witcher3mixed",
			Priority:          25,
			ModType:           "witcher3menumodroot",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchMixedModAndDLC,
			CustomBuild:       buildMixedModAndDLC,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:witcher3:witcher3content",
			VortexInstallerID: "witcher3content",
			Priority:          50,
			ModType:           "witcher3tl",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchContentOnly,
			CustomBuild:       buildContentOnly,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:witcher3:witcher3dlcmod",
			VortexInstallerID: "witcher3dlcmod",
			Priority:          60,
			ModType:           "witcher3dlc",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchDLCMod,
			CustomBuild:       buildDLCMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:witcher3:mods-root",
			VortexInstallerID: "game-query-mod-path",
			Priority:          100,
			ModType:           "witcher3-mod-root",
			NameSource:        installplan.NameSourceArchive,
			StripCommonRoot:   true,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Witcher 3 game registration",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/index.ts",
		},
		{
			Name: "Vortex Witcher 3 installers",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/installers.ts",
		},
		{
			Name: "Vortex Witcher 3 common merge/load-order constants",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/common.ts",
		},
		{
			Name: "Vortex Witcher 3 lifecycle and load-order hooks",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/eventHandlers.ts",
		},
	}
}
