package factorio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "427520"
	VortexGameID = "factorio"
	Name         = "Factorio"

	modsRootID = "factorio-user-mods"
	modType    = "factorio-mod"
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
		ExecutableRelative:  "bin/x64/factorio",
		RequiredFiles:       []string{"data/core/graphics/factorio.ico"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       modsRootID,
		Name:     "Factorio user mods",
		Resolver: linuxModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: modsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:factorio:default",
		VortexInstallerID: "factorio-default",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      modsRootID,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "factorio-ensure-mods-folder",
		Name: "Ensure Factorio user mods folder exists",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func linuxModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return sdk.TargetRootResult{}, errors.New("home directory is required to resolve Factorio mods path")
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(home, ".factorio", "mods"),
		Source: "Vortex Linux Factorio user mods path",
	}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-factorio extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-factorio/src",
	}}
}
