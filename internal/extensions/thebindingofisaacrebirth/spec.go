package thebindingofisaacrebirth

import (
	"context"
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "250900"
	VortexGameID = "thebindingofisaacrebirth"
	Name         = "The Binding of Isaac: Rebirth"

	modType = "thebindingofisaacrebirth-mods"
	modRoot = "mods"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "1.0.0-dmm.1",
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
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:thebindingofisaacrebirth:mods",
		VortexInstallerID: "thebindingofisaacrebirth-mods",
		Priority:          50,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "thebindingofisaacrebirth-executable",
		Name:     "The Binding of Isaac executable marker",
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
	for _, rel := range []string{"isaac-ng.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-516-file-4127", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "The Binding of Isaac: Rebirth Vortex extension package v1.0.0", URL: "https://www.nexusmods.com/site/mods/516?tab=files"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
