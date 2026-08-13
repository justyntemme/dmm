package elex

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "411300"
	VortexGameID = "elex"
	Name         = "Elex"

	packedModType = "elex-packed"
	packedRoot    = "data/packed"
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
		ExecutableRelative: "system/ELEX.exe",
		RequiredFiles:      []string{"system/ELEX.exe"},
		QueryModPath:       packedRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "elex-ensure-packed-folder",
		Name:    "Ensure Elex packed mod folder exists",
		Actions: sdk.EnsureGameDirectories(packedRoot),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: packedModType, TargetRoot: packedRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:elex:pak",
		VortexInstallerID: "elex-mod",
		Priority:          25,
		ModType:           packedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-elex extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-elex/src",
		},
	}
}
