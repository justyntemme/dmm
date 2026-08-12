package rimworld

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversiontext"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "294100"
	VortexGameID = "rimworld"
	Name         = "RimWorld"

	modRoot = "Mods"
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
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "rimworld-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Message:  "Mirrors Vortex's RimWorld requiresLauncher check: installs containing steam_api64.dll should launch through Steam.",
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "rimworld-steam-mod", TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:rimworld:steam-mod",
		VortexInstallerID: "rimworld-steam-mod",
		Priority:          25,
		ModType:           "rimworld-steam-mod",
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSteamMod,
		CustomBuild:       buildSteamMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(gameversiontext.Provider(gameversiontext.Options{
		ID:              "rimworld-version-file",
		Name:            "version.txt",
		Paths:           []string{"version.txt"},
		CaseInsensitive: true,
	}))
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex RimWorld game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-rimworld/src/index.js",
		},
	}
}
