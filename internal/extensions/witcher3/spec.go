package witcher3

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/integrity"
)

const (
	SteamAppID   = "292030"
	SteamAppIDDX = "499450"
	VortexGameID = "witcher3"
	Name         = "The Witcher 3"

	scriptMergerToolID      = "W3ScriptMerger"
	scriptMergerToolName    = "W3 Script Merger"
	scriptMergerToolExe     = "WitcherScriptMerger.exe"
	scriptMergerConfigFile  = "WitcherScriptMerger.exe.config"
	scriptMergerToolModType = "witcher3-script-merger-tool"
	scriptMergerGitHubURL   = "https://github.com/IDCs/WitcherScriptMerger/releases/latest"
	scriptMergerArchiveName = "WitcherScriptMerger-0.6.5.7z"
	scriptMergerVersion     = "0.6.5"
	// Vortex game-witcher3/assets/MD5Cache.json pins the downloaded Script Merger archive.
	scriptMergerArchiveMD5 = "77D57B2384172604E8D859E8BE4F7DF9"
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
		SteamAppIDs:        []string{SteamAppID, SteamAppIDDX},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "bin/x64/witcher3.exe",
		RequiredFiles:      []string{"bin/x64/witcher3.exe"},
		QueryModPath:       "Mods",
		MergeMode:          sdk.GameMergeModeAll,
		RequiresCleanup:    true,
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
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 scriptMergerToolID,
		Name:               scriptMergerToolName,
		ExecutableRelative: "WitcherScriptMerger/WitcherScriptMerger.exe",
		RequiredFiles:      []string{"WitcherScriptMerger/WitcherScriptMerger.exe"},
		Acquisition: &sdk.ToolAcquisitionSpec{
			ID:             "witcher3-script-merger-github",
			Name:           scriptMergerToolName,
			Catalog:        "github",
			Mode:           "direct",
			URL:            scriptMergerGitHubURL,
			ArchiveName:    scriptMergerArchiveName,
			Instructions:   "Vortex queries IDCs/WitcherScriptMerger releases, downloads the latest archive at or above 0.6.5, extracts it under WitcherScriptMerger, and rewrites WitcherScriptMerger.exe.config with the game, vanilla scripts, and Mods paths.",
			Required:       true,
			AutoAcquire:    true,
			SourceModID:    "IDCs/WitcherScriptMerger",
			SourceGame:     "github",
			SourceProvider: "vortex-game-witcher3",
			ExpectedArchiveHashes: []integrity.ExpectedHash{{
				Algorithm: integrity.AlgorithmMD5,
				Value:     scriptMergerArchiveMD5,
				Label:     "Witcher 3 Script Merger archive " + scriptMergerVersion,
			}},
			Message: "Witcher 3 script mods may need Script Merger after deployment. DMM acquires the source-verified GitHub release through the shared managed-tool pipeline.",
		},
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:          "witcher3-install-script-merger",
		Name:        "Install Script Merger",
		Scope:       VortexGameID,
		Kind:        sdk.ExtensionActionKindAcquireTool,
		AcquireTool: &sdk.AcquireToolActionSpec{ToolID: scriptMergerToolID},
		Message:     "Install or reinstall Witcher 3 Script Merger through DMM's managed tool acquisition pipeline.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "witcher3-ensure-mods-and-dlc",
		Name:    "Ensure Witcher 3 mod folders exist",
		Actions: sdk.EnsureGameDirectories("Mods", "DLC"),
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "witcher3-xml-menu-merge", Name: "Witcher 3 XML/menu merge"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:                "witcher3-mods-settings",
		Name:              "Witcher 3 mods.settings load order",
		ModTypes:          []string{"witcher3menumodroot", "witcher3tl", "witcher3dlc", "witcher3-mod-root"},
		Message:           "DMM currently derives the managed mods.settings subset from profile priority; advanced Script Merger ordering remains extension work.",
		UsageInstructions: "Move profile mods up or down to change the generated managed Witcher 3 settings order.",
	})
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "witcher3-menu-readme-conflicts",
		Name:     "Witcher 3 menu/readme conflict ignore",
		Patterns: []string{"README.TXT", "**/*.PART.TXT"},
	})
	r.RegisterDeployIgnore(sdk.DeployIgnoreSpec{
		ID:       "witcher3-menu-readme-deploy",
		Name:     "Witcher 3 menu/readme deploy ignore",
		Patterns: []string{"README.TXT", "**/*.PART.TXT"},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Generate Witcher 3 mods.settings",
		Handler: willDeploy,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidInstallMod,
		Name:    "Configure Witcher 3 Script Merger",
		Handler: didInstallScriptMerger,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "did-deploy",
		Name:    "Remind about Witcher 3 Script Merger after mod changes",
		Handler: didDeployScriptMergerReminder,
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
		{ID: scriptMergerToolModType, DeploymentMode: installplan.ModTypeDeploymentToolOnly},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		scriptMergerToolInstaller(),
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
		{
			Name: "Vortex Witcher 3 Script Merger setup",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-witcher3/src/scriptmerger.ts",
		},
	}
}
