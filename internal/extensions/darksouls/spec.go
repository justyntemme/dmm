package darksouls

import (
	"context"
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "211420"
	VortexGameID = "darksouls"
	Name         = "Dark Souls"

	executable = "DATA/DARKSOULS.exe"
	modRoot    = "DATA/dsfix/tex_override"
	modType    = "darksouls-dsfix-tex-override"
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
		ExecutableRelative: executable,
		RequiredFiles:      []string{executable},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:darksouls:dsfix-tex-override",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "darksouls-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM evaluates Vortex's Steam launcher requirement against the discovered Steam app and reports it through launcher diagnostics.",
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "darksouls-dsfix-installed",
		Name:        "DSfix",
		Kind:        "mod-loader",
		Required:    true,
		ModTypes:    []string{modType},
		Message:     "DSfix was not found in DATA/dsfix. Dark Souls texture override mods require DSfix before they can load.",
		OKMessage:   "DSfix texture override folder is present.",
		HelpURL:     "https://www.nexusmods.com/darksouls/mods/19",
		InstallHint: "Open the DSfix Nexus page, install DSfix with DMM, then enable texture override mods again.",
		Check:       dsfixMarkers,
		Acquisition: &gamehandler.RuntimeAcquisitionSpec{
			ID:           "darksouls-dsfix-nexus",
			Name:         "DSfix Nexus page",
			Catalog:      "nexus",
			Mode:         "nexus-download",
			Instructions: "Open DSfix on Nexus Mods and click Mod Manager Download for the DSfix file.",
			Required:     true,
			SourceGame:   VortexGameID,
			SourceModID:  "19",
			Message:      "Vortex prompts users to install DSfix from Nexus mod 19 when the DSfix texture override folder is missing.",
		},
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-darksouls extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-darksouls/src",
	})
}

func dsfixMarkers(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	path := filepath.Join(gamePath, filepath.FromSlash(modRoot))
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return []string{"Found " + filepath.ToSlash(path)}
	}
	return nil
}
