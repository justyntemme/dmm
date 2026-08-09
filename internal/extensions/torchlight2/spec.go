package torchlight2

import (
	"context"
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "200710"
	VortexGameID = "torchlight2"
	Name         = "Torchlight II"

	documentsModsRootID = "torchlight2-documents-mods"
	modType             = "torchlight2-mod"
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
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  "ModLauncher.bin.x86",
		RequiredFiles:       []string{"Torchlight2.bin.x86", "ModLauncher.bin.x86"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       documentsModsRootID,
		Name:     "Torchlight II Documents Mods",
		Resolver: documentsModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: documentsModsRootID})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "torchlight2-steam-launcher",
		Name:     "Torchlight II Steam launcher requirement",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex requires Torchlight II Steam installs to launch through Steam.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:torchlight2:mod",
		VortexInstallerID: "torchlight2-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModArchive,
		CustomBuild:       buildModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func documentsModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(home, "Documents", "My Games", "runic games", "torchlight 2", "mods"),
		Source: "Vortex documents path",
	}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-torchlight2 extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-torchlight2/src",
		},
	}
}
