package commandconquergenerals

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
	SteamAppID   = "2229870"
	VortexGameID = "cncgenerals"
	Name         = "Command & Conquer: Generals"

	bigModType = "cncgenerals-big"
)

var requiredGameFiles = []string{
	"Generals.exe",
	"Game.dat",
	"INI.big",
}

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
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: bigModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:cncgenerals:big",
		VortexInstallerID: "cncgenerals-big",
		Priority:          30,
		ModType:           bigModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBigArchive,
		CustomBuild:       buildBigArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "cncgenerals-required-files",
		Name:        "Command & Conquer: Generals install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{bigModType},
		Message:     "The Command & Conquer: Generals game folder is missing files needed for .big mod support.",
		OKMessage:   "The Command & Conquer: Generals game folder contains the expected executable and .big archives.",
		InstallHint: "Verify the game files in Steam before testing Generals .big mods.",
		Check:       checkRequiredGameFiles,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
			continue
		}
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Command & Conquer: Generals domain", URL: "https://www.nexusmods.com/cncgenerals"},
		{Name: "C&C Labs .big package install guidance", URL: "https://www.cnclabs.com/forums/posts/13491/how-to-install-a-mod-addon-for-zero-hour/"},
		{Name: "Steam community guide showing GenLauncher as a separate external flow", URL: "https://steamcommunity.com/sharedfiles/filedetails/?id=3175443026"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Generals handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}
