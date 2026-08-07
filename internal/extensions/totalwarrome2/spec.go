package totalwarrome2

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
	SteamAppID   = "214950"
	VortexGameID = "totalwarrome2"
	Name         = "Total War: ROME II - Emperor Edition"

	researchModType = "totalwarrome2-research-blocked"
)

var requiredGameFiles = []string{
	"Rome2.exe",
	"data/manifest.txt",
	"data/data_rome2.pack",
}

const unsupportedReason = "Total War: ROME II has a verified Nexus domain but no verified Vortex extension in the checked Vortex source. Steam appdetails do not declare Steam Workshop support, and current support is blocked until representative Nexus archives are reviewed for pack-file, launcher, data-folder, or external-manager install semantics."

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
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: researchModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "research:totalwarrome2:blocked",
		VortexInstallerID: "totalwarrome2-research-blocked",
		Priority:          10000,
		ModType:           researchModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "totalwarrome2-required-files",
		Name:        "Total War: ROME II install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{researchModType},
		Message:     "The Total War: ROME II game folder is missing files needed for future extension support.",
		OKMessage:   "The Total War: ROME II game folder contains the expected executable and pack-file layout.",
		InstallHint: "Verify the game files in Steam before testing Total War: ROME II mods.",
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
			Name: "Nexus game domain",
			URL:  "https://www.nexusmods.com/totalwarrome2",
		},
		{
			Name: "Checked Vortex bundled game extensions; no Total War: ROME II extension found",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
		},
		{
			Name: "Steam appdetails categories; no Steam Workshop category declared",
			URL:  "https://store.steampowered.com/api/appdetails?appids=214950&filters=categories",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
