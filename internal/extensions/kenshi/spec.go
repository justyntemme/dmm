package kenshi

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "233860"
	VortexGameID = "kenshi"
	Name         = "Kenshi"

	modRoot = "mods"
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
	r.RegisterModType(installplan.ModTypeSpec{ID: "kenshi-mod", TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:kenshi:mod",
		VortexInstallerID: "kenshi-mod",
		Priority:          25,
		ModType:           "kenshi-mod",
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModArchive,
		CustomBuild:       buildModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "nvidiaProfileInspector",
		Name:               "Darkmod inspector",
		ExecutableRelative: "nvidiaProfileInspector.exe",
		RequiredFiles: []string{
			"nvidiaProfileInspector.exe",
			"nvidiaProfileInspector.pdb",
			"CustomColors.xml",
			"Reference.xml",
		},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "WOCS.Patcher.Scar.PathFinding fix",
		Name:               "OCS.Patcher.Scar.PathFinding fix",
		ExecutableRelative: "OCS.Patcher.Scar.PathFinding.Steam.exe",
		RequiredFiles: []string{
			"OCS.Patcher.Scar.PathFinding.exe",
			"OpenConstructionSet.dll",
		},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "kenshi-current-version",
		Name:     "currentVersion.txt",
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
	data, err := os.ReadFile(filepath.Join(gamePath, "currentVersion.txt"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 2 {
		return sdk.GameVersionResult{Version: fields[1], Source: "currentVersion.txt"}, nil
	}
	return sdk.GameVersionResult{Version: strings.TrimSpace(string(data)), Source: "currentVersion.txt"}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Kenshi game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-kenshi/src/index.js",
		},
	}
}
