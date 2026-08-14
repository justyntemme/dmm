package kenshi

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversiontext"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "233860"
	VortexGameID = "kenshi"
	Name         = "Kenshi"

	modRoot = "mods"
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
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "kenshi-ensure-mods-folder",
		Name:    "Ensure Kenshi mods folder exists",
		Actions: sdk.EnsureGameDirectories(modRoot),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "kenshi-mod", TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:kenshi:mod",
		VortexInstallerID: "kenshi-mod",
		Priority:          25,
		ModType:           "kenshi-mod",
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModArchive,
		CustomBuild:       buildModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "nvidiaProfileInspector",
		Name:               "Darkmod inspector",
		ExecutableRelative: "nvidiaProfileInspector.exe",
		RequiredFiles: []string{
			"nvidiaProfileInspector.exe",
			"nvidiaProfileInspector.pdb",
			"CustomColors.xml",
			"Reference.xml",
		},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "WOCS.Patcher.Scar.PathFinding fix",
		Name:               "OCS.Patcher.Scar.PathFinding fix",
		ExecutableRelative: "OCS.Patcher.Scar.PathFinding.Steam.exe",
		RequiredFiles: []string{
			"OCS.Patcher.Scar.PathFinding.exe",
			"OpenConstructionSet.dll",
		},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "kenshi-steam-launcher",
		Name:     "Kenshi Steam launcher requirement",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM evaluates Vortex's Steam launcher requirement against the discovered Steam app and reports it through launcher diagnostics.",
	})
	r.RegisterGameVersionProvider(gameversiontext.Provider(gameversiontext.Options{
		ID:        "kenshi-current-version",
		Name:      "currentVersion.txt",
		Paths:     []string{"currentVersion.txt"},
		Extractor: gameversiontext.WhitespaceField(1, true),
	}))
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Kenshi game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-kenshi/src/index.js",
		},
	}
}
