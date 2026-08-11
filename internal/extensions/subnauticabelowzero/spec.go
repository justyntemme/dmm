package subnauticabelowzero

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "848450"
	VortexGameID = "subnauticabelowzero"
	Name         = "Subnautica: Below Zero"
	SupportModID = "203"
	qmodsRoot    = "QMods"
	modType      = "subnauticabelowzero-qmods"
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
		ID:                "vortex:subnauticabelowzero:qmods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        qmodsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-subnauticabelowzero extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-subnauticabelowzero/src"})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex support mod declared by game-subnauticabelowzero", URL: "https://www.nexusmods.com/site/mods/" + SupportModID})
}
