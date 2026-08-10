package dawnofman

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/umm"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "858810"
	VortexGameID = "dawnofman"
	Name         = "Dawn of Man"

	gameModsRoot    = "Mods"
	scenarioRootID  = "dawnofman-scenarios"
	ummModType      = "dawnofman-umm-mod"
	scenarioModType = "dom-scene-modtype"
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
		ExecutableRelative: "DawnOfMan.exe",
		RequiredFiles:      []string{"DawnOfMan.exe"},
		QueryModPath:       gameModsRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       scenarioRootID,
		Name:     "Dawn of Man scenarios",
		Resolver: scenarioRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: ummModType, TargetRoot: gameModsRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: scenarioModType, TargetRootID: scenarioRootID})
	umm.RegisterToolRuntimeSupport(r, umm.GameOptions{GameID: VortexGameID, GameName: Name, ModType: ummModType})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dawnofman:scene",
		VortexInstallerID: "dom-scene-installer",
		Priority:          25,
		ModType:           scenarioModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSceneArchive,
		CustomBuild:       buildSceneArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dawnofman:umm",
		VortexInstallerID: "dom-mod",
		Priority:          25,
		ModType:           ummModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchUMMArchive,
		CustomBuild:       buildUMMArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "dawnofman-prepare-for-modding",
		Name: "Prepare Dawn of Man mod folders",
		Actions: append(
			sdk.EnsureGameDirectories(gameModsRoot),
			sdk.EnsureTargetRootDirectories(scenarioRootID, ".")...,
		),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func scenarioRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return sdk.TargetRootResult{}, errors.New("home directory is required to resolve Dawn of Man scenario path")
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(home, "Documents", "DawnOfMan", "Scenarios"),
		Source: "Vortex documents path",
	}, nil
}

func sources() []sdk.SourceRef {
	return append([]sdk.SourceRef{
		{
			Name: "Vortex game-dawnofman extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-dawnofman/src",
		},
	}, umm.Sources()...)
}
