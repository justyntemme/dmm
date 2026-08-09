package wolcen

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/xmlmerge"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "424370"
	VortexGameID = "wolcenlordsofmayhem"
	Name         = "Wolcen: Lords of Mayhem"

	modType      = "wolcen-game"
	gameModRoot  = "Game"
	executable   = "win_x64/Wolcen.exe"
	mergeHandler = "wolcen-xml-mtl-merge"
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
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executable,
		RequiredFiles:      []string{executable},
		QueryModPath:       gameModRoot,
		MergeMode:          sdk.GameMergeModeAll,
		StopPatterns: []string{
			`(^|/)levels(/|$)`,
			`(^|/)Lib(/|$)`,
			`(^|/)Loot(/|$)`,
			`(^|/)Objects(/|$)`,
			`(^|/)Umbra(/|$)`,
		},
		Environment: map[string]string{"SteamAPPId": SteamAppID},
		Deployment:  installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: gameModRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:wolcen:game-root",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		Match:             installplan.MatchSpec{UseGameStopPatterns: true},
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterMerge(sdk.MergeSpec{ID: mergeHandler, Name: "Wolcen XML/MTL merge"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: sdk.EventWillDeploy,
		Name:  "Wolcen XML/MTL merge generation",
		Handler: xmlmerge.WillDeploy(xmlmerge.Options{
			Extensions: []string{".xml", ".mtl"},
			Message:    "Wolcen XML/MTL merge generated deployment files.",
		}),
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "wolcen-prepare-mods",
		Name:           "Prepare Wolcen Mods folder",
		GeneratedFiles: []string{"Mods"},
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-wolcen extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-wolcen/src",
	})
}
