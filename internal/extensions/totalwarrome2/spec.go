package totalwarrome2

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "214950"
	VortexGameID = "totalwarrome2"
	Name         = "Total War: ROME II - Emperor Edition"

	packModType      = "totalwarrome2-pack"
	dataRoot         = "data"
	userScriptRootID = "totalwarrome2-user-script-root"
	packNoticeEvent  = "did-deploy"
)

var requiredGameFiles = []string{
	"Rome2.exe",
	"data/manifest.txt",
	"data/data_rome2.pack",
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
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       userScriptRootID,
		Name:     "Total War: ROME II Proton user scripts",
		Resolver: targetroots.ProtonRoamingAppData(SteamAppID, "The Creative Assembly", "Rome2", "scripts"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: packModType, TargetRoot: dataRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:totalwarrome2:pack",
		VortexInstallerID: "totalwarrome2-pack",
		Priority:          25,
		ModType:           packModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPackArchive,
		CustomBuild:       buildPackArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "totalwarrome2-required-files",
		Name:        "Total War: ROME II install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{packModType},
		Message:     "The Total War: ROME II game folder is missing files needed for future extension support.",
		OKMessage:   "The Total War: ROME II game folder contains the expected executable and pack-file layout.",
		InstallHint: "Verify the game files in Steam before testing Total War: ROME II mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Total War: ROME II user.script generator",
		Handler: willDeployUserScript,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   packNoticeEvent,
		Name:    "Total War: ROME II pack activation reminder",
		Handler: didDeployPackNotice,
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
			Name: "Vortex Total War: Three Kingdoms pack installer source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games/game-totalwarthreekingdoms/src/index.js",
		},
		{
			Name: "Total War official Workshop pack-location note",
			URL:  "https://wiki.totalwar.com/w/Steam_Workshop_and_How_to_Make_Mods.html",
		},
		{
			Name: "Total War ROME II user.script loading note",
			URL:  "https://www.totalwar.com/news/improving-game-and-mod-interaction-with-desert-kingdoms",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
