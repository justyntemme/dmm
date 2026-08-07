package finalfantasyxx2hdremaster

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
	SteamAppID   = "359870"
	VortexGameID = "finalfantasyxx2hdremaster"
	Name         = "Final Fantasy X/X-2 HD Remaster"

	researchModType = "finalfantasyxx2hdremaster-research-blocked"
)

var requiredGameFiles = []string{
	"FFX&X-2_LAUNCHER.exe",
	"FFX.exe",
	"FFX-2.exe",
	"data/FFX_Data.vbf",
	"data/FFX2_Data.vbf",
}

const unsupportedReason = "Final Fantasy X/X-2 HD Remaster has a verified Nexus domain but no verified Vortex extension in the checked central manifest or Vortex source. Current support is blocked until representative archives are reviewed and extension-owned rules define whether files patch VBF archives, replace loose binaries, or target a loader-specific folder."

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
		ID:                "research:finalfantasyxx2hdremaster:blocked",
		VortexInstallerID: "finalfantasyxx2hdremaster-research-blocked",
		Priority:          10000,
		ModType:           researchModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "finalfantasyxx2hdremaster-required-files",
		Name:        "Final Fantasy X/X-2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{researchModType},
		Message:     "The Final Fantasy X/X-2 game folder is missing files needed for future extension support.",
		OKMessage:   "The Final Fantasy X/X-2 game folder contains the expected executables and VBF archives.",
		InstallHint: "Verify the game files in Steam before testing Final Fantasy X/X-2 mods.",
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
			Name: "Nexus API game-domain verification",
			URL:  "https://api.nexusmods.com/v1/games.json",
		},
		{
			Name: "Checked Vortex central extension manifest; no Final Fantasy X/X-2 entry found",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Checked Vortex bundled game extensions; no Final Fantasy X/X-2 extension found",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
