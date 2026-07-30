package rimworld

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "294100"
	VortexGameID = "rimworld"
	Name         = "RimWorld"

	modRoot = "Mods"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "rimworld-steam-mod", TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:rimworld:steam-mod",
		VortexInstallerID: "rimworld-steam-mod",
		Priority:          25,
		ModType:           "rimworld-steam-mod",
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSteamMod,
		CustomBuild:       buildSteamMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "rimworld-version-file",
		Name:     "version.txt",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, "version.txt"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	return sdk.GameVersionResult{Version: strings.TrimSpace(string(data)), Source: "version.txt"}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex RimWorld game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-rimworld/src/index.js",
		},
	}
}
