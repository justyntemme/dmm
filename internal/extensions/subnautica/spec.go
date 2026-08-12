package subnautica

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const verifiedVortexCommit = "2349a17900a37c2120e90733045dc6b303135b89"

const (
	SteamAppID   = "264710"
	VortexGameID = "subnautica"
	Name         = "Subnautica"
	SupportModID = "202"
	qmodsRoot    = "QMods"
	modType      = "subnautica-qmods"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		SupportModID: SupportModID,
		QueryModPath: qmodsRoot,
		MergeMode:    sdk.GameMergeModeNone,
		Deployment:   installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: qmodsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:subnautica:qmods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        qmodsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-subnautica extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/" + verifiedVortexCommit + "/extensions/games/game-subnautica/src"})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex support mod declared by game-subnautica", URL: "https://www.nexusmods.com/site/mods/" + SupportModID})
}
