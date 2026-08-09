package galacticcivilizations3

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
	SteamAppID   = "976210"
	VortexGameID = "galacticcivilizations3"
	Name         = "Galactic Civilizations III"

	activeDocumentsRootID  = "galciv3-active-documents"
	crusadeDocumentsRootID = "galciv3-crusade-documents"
	modType                = "galciv3-mod"
	crusadeModType         = "galciv3crusade"
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
		ExecutableRelative:  "GalCiv3.exe",
		RequiredFiles:       []string{"GalCiv3.exe"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       activeDocumentsRootID,
		Name:     "GalCiv3 active documents mods",
		Resolver: activeDocumentsRoot,
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       crusadeDocumentsRootID,
		Name:     "GalCiv3 Crusade documents mods",
		Resolver: crusadeDocumentsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: activeDocumentsRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: crusadeModType, TargetRootID: crusadeDocumentsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:galciv3:archive",
		VortexInstallerID: "galciv3installer",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		CustomBuild:       buildArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "GalCiv3 in-game enable reminder",
		Handler: didDeployReminder,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func activeDocumentsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	crusade, err := galCivDocumentsPath("GC3Crusade")
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	if info, err := os.Stat(crusade); err == nil && info.IsDir() {
		return sdk.TargetRootResult{Path: crusade, Source: "Vortex documents path: GC3Crusade exists"}, nil
	}
	base, err := galCivDocumentsPath("GalCiv3")
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: base, Source: "Vortex documents path"}, nil
}

func crusadeDocumentsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	path, err := galCivDocumentsPath("GC3Crusade")
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: path, Source: "Vortex documents path: GC3Crusade"}, nil
}

func galCivDocumentsPath(folder string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", errors.New("home directory is required to resolve Galactic Civilizations III documents path")
	}
	return filepath.Join(home, "Documents", "My Games", folder), nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-galciv3 extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-galciv3/src",
		},
	}
}
