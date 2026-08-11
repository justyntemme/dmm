package teso

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversionhash"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "306130"
	VortexGameID = "teso"
	NexusDomain  = "elderscrollsonline"
	Name         = "The Elder Scrolls Online"

	executable = "Bethesda.net_Launcher.exe"
	addonsRoot = "teso-documents-addons"
	modType    = "teso-addon"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{NexusDomain},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       addonsRoot,
		Name:     "The Elder Scrolls Online Documents AddOns",
		Resolver: targetroots.ProtonDocuments(SteamAppID, "Elder Scrolls Online", "live", "Addons"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: addonsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:teso:addon",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      addonsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{ID: "teso-ensure-addons-folder", Name: "Ensure The Elder Scrolls Online AddOns folder exists", Actions: sdk.EnsureTargetRootDirectories(addonsRoot, ".")})
	r.RegisterGameVersionProvider(gameversionhash.Provider(gameversionhash.Options{
		ID:           "teso-hash-version",
		Name:         "The Elder Scrolls Online hash version",
		VortexGameID: VortexGameID,
		HashFiles: []string{
			"../The Elder Scrolls Online/game/game_player.version",
			"../The Elder Scrolls Online/depot/depot.version",
			"Bethesda.net_Launcher.version",
		},
	}))
	r.RegisterGameStore(sdk.GameStoreSpec{ID: "registry", Name: "Windows registry", Status: sdk.CapabilityStatusNotApplicable, Message: "Vortex can discover the standalone ESO launcher through the Windows registry. DMM's Steam Deck MVP runtime uses Steam discovery."})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-teso extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-teso/src"})
}
