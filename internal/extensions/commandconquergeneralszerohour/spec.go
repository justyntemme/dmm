package commandconquergeneralszerohour

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
	SteamAppID = "2732960"
	ID         = "commandconquergeneralszerohour"
	Name       = "Command & Conquer Generals Zero Hour"

	bigModType = "cncgeneralszerohour-big"
)

var requiredGameFiles = []string{
	"Generals.exe",
	"Game.dat",
	"INIZH.big",
}

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
		SteamAppIDs: []string{SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: bigModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:cncgeneralszerohour:big",
		VortexInstallerID: "cncgeneralszerohour-big",
		Priority:          30,
		ModType:           bigModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBigArchive,
		CustomBuild:       buildBigArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "cncgeneralszerohour-required-files",
		Name:        "Command & Conquer Generals Zero Hour install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{bigModType},
		Message:     "The Command & Conquer Generals Zero Hour game folder is missing files needed for .big mod support.",
		OKMessage:   "The Command & Conquer Generals Zero Hour game folder contains the expected executable and .big archives.",
		InstallHint: "Verify the game files in Steam before testing Zero Hour .big mods.",
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
		{
			Name: "Steam Store appdetails category verification",
			URL:  "https://store.steampowered.com/api/appdetails?appids=2732960&filters=categories",
		},
		{
			Name: "Steam Deck installed app manifest snapshot",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
		{
			Name: "C&C Labs .big package install guidance",
			URL:  "https://www.cnclabs.com/forums/posts/13491/how-to-install-a-mod-addon-for-zero-hour/",
		},
	}
}
