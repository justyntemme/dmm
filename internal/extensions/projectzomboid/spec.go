package projectzomboid

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
	SteamAppID   = "108600"
	ID           = "projectzomboid"
	Name         = "Project Zomboid"
	localModsID  = "projectzomboid-local-mods"
	modInfoType  = "projectzomboid-mod"
	zomboidMods  = "Zomboid/mods"
	vortexModID  = "692"
	vortexFileID = "5970"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       ID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{ID},
		VortexGameID: ID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       localModsID,
		Name:     "Project Zomboid user mods",
		Resolver: localModsRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modInfoType, TargetRootID: localModsID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:projectzomboid:mod-info",
		VortexInstallerID: "projectzomboid-mod-info",
		Priority:          50,
		ModType:           modInfoType,
		NameSource:        installplan.NameSourceManifestDisplay,
		CustomMatch:       matchModInfoArchive,
		CustomBuild:       buildModInfoArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func localModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return sdk.TargetRootResult{}, errors.New("home directory is required to resolve Project Zomboid user mods path")
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(home, filepath.FromSlash(zomboidMods)),
		Source: "Project Zomboid user data",
	}, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Project Zomboid Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/" + vortexModID,
		},
	}
}
