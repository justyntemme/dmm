package survivingmars

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "464920"
	VortexGameID = "survivingmars"
	Name         = "Surviving Mars"

	appDataModsRootID = "survivingmars-appdata-mods"
	modType           = "survivingmars-mod"
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
		ExecutableRelative:  "MarsSteam.exe",
		RequiredFiles:       []string{"Packs/Cubemaps.hpk"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       appDataModsRootID,
		Name:     "Surviving Mars AppData mods",
		Resolver: appDataModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: appDataModsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:survivingmars:mod",
		VortexInstallerID: "survivingmars-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModContentArchive,
		CustomBuild:       buildModContentArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func appDataModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return sdk.TargetRootResult{}, errors.New("Steam library path is required to resolve Surviving Mars Proton AppData mods path")
	}
	return sdk.TargetRootResult{
		Path: filepath.Join(
			libraryPath,
			"steamapps",
			"compatdata",
			SteamAppID,
			"pfx",
			"drive_c",
			"users",
			"steamuser",
			"AppData",
			"Roaming",
			"Surviving Mars",
			"mods",
		),
		Source: "Vortex AppData path via Steam Proton prefix",
	}, nil
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-survivingmars extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-survivingmars/src",
		},
	}
}
