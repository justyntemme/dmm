package daggerfallunity

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	VortexGameID = "daggerfallunity"
	Name         = "Daggerfall Unity"

	streamingAssetsRoot = "DaggerfallUnity_Data/StreamingAssets"
	modType             = "daggerfallunity-streaming-assets"
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
		VortexGameID:       VortexGameID,
		AllowNoSteamAppID:  true,
		ExecutableRelative: "DaggerfallUnity.exe",
		RequiredFiles:      []string{"DaggerfallUnity.exe"},
		QueryModPath:       streamingAssetsRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: streamingAssetsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:daggerfallunity:dfmod",
		VortexInstallerID: "dfmodmultiplatform",
		Priority:          15,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchDFModArchive,
		CustomBuild:       buildDFModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-daggerfallunity extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-daggerfallunity/src",
		},
	}
}
