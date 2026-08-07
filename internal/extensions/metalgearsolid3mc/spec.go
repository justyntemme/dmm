package metalgearsolid3mc

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
	SteamAppID   = "2131650"
	VortexGameID = "metalgearsolid3mc"
	Name         = "Metal Gear Solid 3: Snake Eater - Master Collection"

	rootModType = "metalgearsolid3mc-root"
)

var requiredGameFiles = []string{
	"launcher.exe",
	"METAL GEAR SOLID3.exe",
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
		ID:                "vortex:metalgearsolid3mc:root",
		VortexInstallerID: "metalgearsolid3mc-root",
		Priority:          50,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "metalgearsolid3mc-required-files",
		Name:        "Metal Gear Solid 3 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{rootModType},
		Message:     "The Metal Gear Solid 3 game folder is missing files required by the Vortex extension.",
		OKMessage:   "The Metal Gear Solid 3 game folder contains the files required by the Vortex extension.",
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
			Name: "Vortex extension source copied from Steam Deck Vortex plugin cache",
			URL:  "deck:/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Metal Gear Solid 3 - Snake Eater - Master Collection Vortex Extension v1.0.0/index.js",
		},
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Metal Gear Solid 3 Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/728",
		},
		{
			Name: "Live Steam Deck path check found only a stale folder without Steam manifest",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
