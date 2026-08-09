package grimrock

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "207170"
	VortexGameID = "grimrock"
	Name         = "Legend of Grimrock"

	executable     = "Grimrock.bin.x86"
	dungeonsRoot   = "grimrock-documents-dungeons"
	dungeonModType = "grimrock-dungeon"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       dungeonsRoot,
		Name:     "Legend of Grimrock Documents Dungeons",
		Resolver: targetroots.HostDocuments("Almost Human", "Legend of Grimrock", "Dungeons"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: dungeonModType, TargetRootID: dungeonsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:grimrock:dungeon",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           dungeonModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      dungeonsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{ID: "grimrock-ensure-dungeons-folder", Name: "Ensure Legend of Grimrock Dungeons folder exists", GeneratedFiles: []string{"Almost Human/Legend of Grimrock/Dungeons"}})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-grimrock extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-grimrock/src"})
}
