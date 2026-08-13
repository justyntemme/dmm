package sekiro

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
	SteamAppID   = "814380"
	VortexGameID = "sekiro"
	Name         = "Sekiro"

	modRoot = "mods"
	modType = "sekiro-mod-engine"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "sekiro.exe",
		RequiredFiles:      []string{"sekiro.exe"},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "sekiro-ensure-mod-engine-parts-folder",
		Name:    "Ensure Sekiro Mod Engine parts folder exists",
		Actions: sdk.EnsureGameDirectories("mods/parts"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:sekiro:root-mod",
		VortexInstallerID: "sek-root-mod",
		Priority:          20,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchRootArchive,
		CustomBuild:       buildRootArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:sekiro:loose-parts",
		VortexInstallerID: "sek-loose-files",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchLoosePartsArchive,
		CustomBuild:       buildLoosePartsArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "sekiro-mod-engine-present",
		Name:        "Sekiro Mod Engine",
		Kind:        "game-file",
		Required:    true,
		ModTypes:    []string{modType},
		Message:     "Sekiro Mod Engine was not detected. Mods installed into the mods folder generally require dinput8.dll from Sekiro Mod Engine to be enabled and deployed.",
		OKMessage:   "Sekiro Mod Engine is present.",
		InstallHint: "Install Sekiro Mod Engine from Nexus Mods and keep it enabled in this profile.",
		Check:       modEngineCheck,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modEngineCheck(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	path := filepath.Join(gamePath, "dinput8.dll")
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-sekiro extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sekiro/src",
		},
	}
}
