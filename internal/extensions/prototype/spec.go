package prototype

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
	SteamAppID   = "10150"
	VortexGameID = "prototype"
	Name         = "Prototype"

	researchModType = "prototype-research-blocked"
)

var requiredGameFiles = []string{
	"prototypef.exe",
	"art.rcf",
	"scripts.rcf",
}

const unsupportedReason = "Prototype has no verified Vortex extension in the checked Nexus/Vortex sources, and current Nexus mods use multiple manual patterns including root ASI drops, TexMod packages, and extracted RCF folders. DMM needs pattern-specific extension rules before installing Prototype archives safely."

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
		ID:                "research:prototype:blocked",
		VortexInstallerID: "prototype-research-blocked",
		Priority:          10000,
		ModType:           researchModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "prototype-required-files",
		Name:        "Prototype install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{researchModType},
		Message:     "The Prototype game folder is missing files needed for future extension support.",
		OKMessage:   "The Prototype game folder contains the expected executable and RCF archives.",
		InstallHint: "Verify the game files in Steam before testing Prototype mods.",
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
			URL:  "https://www.nexusmods.com/prototype",
		},
		{
			Name: "Checked Vortex central extension manifest; no Prototype entry found",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Representative Nexus root ASI fix",
			URL:  "https://www.nexusmods.com/prototype/mods/52",
		},
		{
			Name: "Representative Nexus RCF/extractor discussion",
			URL:  "https://www.nexusmods.com/prototype/mods/81",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
