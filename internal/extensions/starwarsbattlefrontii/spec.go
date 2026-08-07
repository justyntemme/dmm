package starwarsbattlefrontii

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
	SteamAppID   = "1237950"
	VortexGameID = "starwarsbattlefront22017"
	Name         = "STAR WARS Battlefront II"

	frostyRoot       = "FrostyModManager"
	frostyExecutable = "FrostyModManager.exe"
	fbmodRoot        = "FrostyModManager/Mods/StarWarsBattlefrontII"
	fbmodModType     = "starwarsbattlefront22017-fbmod"
)

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
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: fbmodModType, TargetRoot: fbmodRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:starwarsbattlefront22017:fbmod",
		VortexInstallerID: "starwarsbattlefront22017-mod",
		Priority:          25,
		ModType:           fbmodModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchFBModArchive,
		CustomBuild:       buildFBModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "starwarsbattlefront22017-frosty-installed",
		Name:        "Frosty Mod Manager",
		Kind:        "mod-launcher",
		Required:    true,
		ModTypes:    []string{fbmodModType},
		Message:     "Frosty Mod Manager is required before enabled Battlefront II .fbmod files can load.",
		OKMessage:   "Frosty Mod Manager is present in the game folder.",
		HelpURL:     "https://frostytoolsuite.com/downloads.html",
		InstallHint: "Install Frosty Mod Manager into the game's FrostyModManager folder, then configure the Frosty launch tool.",
		Check:       checkFrostyFiles,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "starwarsbattlefront22017-frosty-launch",
		Name:               "Launch Modded Game",
		ExecutableRelative: filepath.ToSlash(filepath.Join(frostyRoot, frostyExecutable)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(frostyRoot, frostyExecutable))},
		DefaultPrimary:     true,
		ModTypes:           []string{fbmodModType},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "starwarsbattlefront22017-frosty",
		Name:               "Frosty Mod Manager",
		ExecutableRelative: filepath.ToSlash(filepath.Join(frostyRoot, frostyExecutable)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(frostyRoot, frostyExecutable))},
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkFrostyFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	rel := filepath.Join(frostyRoot, frostyExecutable)
	path := filepath.Join(gamePath, rel)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Star Wars Battlefront II extension source",
			URL:  "https://github.com/alistair3149/game-starwarsbattlefront22017",
		},
		{
			Name: "Current Nexus Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/112",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
