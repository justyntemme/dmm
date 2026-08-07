package halflife2

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
	HalfLife2AppID      = "220"
	EpisodeOneAppID     = "380"
	EpisodeTwoAppID     = "420"
	VortexGameID        = "halflife2"
	Name                = "Half-Life 2"
	researchModType     = "halflife2-research-blocked"
	extensionManifestID = "site-mod-80-file-516"
)

var requiredGameFiles = []string{
	"hl2.sh",
	"hl2_linux",
	"hl2/gameinfo.txt",
	"episodic/gameinfo.txt",
	"ep2/gameinfo.txt",
}

const unsupportedReason = "Half-Life 2 has a verified Vortex extension manifest entry, but the extension source/package has not yet been inspected. Source-engine mods can target base game folders, episode folders, custom folders, VPK archives, sourcemods, or external tools, so DMM blocks archive installs until those layouts are classified into extension-owned installer rules."

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
		SteamAppIDs:  []string{HalfLife2AppID, EpisodeOneAppID, EpisodeTwoAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			DefaultStrategy:       installplan.DeployStrategySymlink,
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: researchModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "research:halflife2:blocked",
		VortexInstallerID: "halflife2-research-blocked",
		Priority:          10000,
		ModType:           researchModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "halflife2-required-files",
		Name:        "Half-Life 2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{researchModType},
		Message:     "The Half-Life 2 game folder is missing files needed for future extension support.",
		OKMessage:   "The Half-Life 2 game folder contains the expected shared Source-engine executable and episode folders.",
		InstallHint: "Verify Half-Life 2, Episode One, and Episode Two files in Steam before testing Half-Life 2 mods.",
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
			Name: "Vortex central extension manifest entry " + extensionManifestID,
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Half-Life 2 Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/80",
		},
		{
			Name: "Checked Vortex bundled game extensions; no Half-Life 2 source found",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
