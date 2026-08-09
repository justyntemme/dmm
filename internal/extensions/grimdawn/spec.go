package grimdawn

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "219990"
	VortexGameID = "grimdawn"
	Name         = "Grim Dawn"

	executable = "Grim Dawn.exe"
	modRoot    = "mods"
	modType    = "grimdawn-mods"
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
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:grimdawn:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "grimdawn-ensure-mods-folder",
		Name:           "Ensure Grim Dawn mods folder exists",
		GeneratedFiles: []string{modRoot},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "grimdawn-hash-version",
		Name:    "Grim Dawn executable and Game.dll hash version",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex hashes Grim Dawn.exe and Game.dll for game-version metadata. DMM needs a reusable hash-version provider runtime before this can run.",
	})
	r.RegisterGameStore(sdk.GameStoreSpec{
		ID:      "gog",
		Name:    "GOG",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex can discover Grim Dawn through the GOG registry key. DMM's current Steam Deck target uses Steam discovery.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-grimdawn extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-grimdawn/src",
	})
}
