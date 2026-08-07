package metalgearandmetalgear2mc

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2131680"
	VortexGameID = "metalgearandmetalgear2mc"
	Name         = "Metal Gear & Metal Gear 2: Solid Snake - Master Collection"

	rootModType = "metalgearandmetalgear2mc-root"
)

var requiredGameFiles = []string{
	"launcher.exe",
	"METAL GEAR.exe",
}

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
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:metalgearandmetalgear2mc:root",
		VortexInstallerID: rootModType,
		Priority:          50,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "metalgearandmetalgear2mc-required-files",
		Name:        "Metal Gear & Metal Gear 2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{rootModType},
		Message:     "The Metal Gear & Metal Gear 2 game folder is missing files required by the Vortex extension.",
		OKMessage:   "The Metal Gear & Metal Gear 2 game folder contains the files required by the Vortex extension.",
		InstallHint: "Verify the game files in Steam before deploying mods.",
		Check:       checkRequiredGameFiles,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	if len(details) != len(requiredGameFiles) {
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Metal Gear & Metal Gear 2 Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/725",
		},
		{
			Name: "Verified Vortex extension package file",
			URL:  "https://www.nexusmods.com/site/mods/725?tab=files&file_id=2522",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
