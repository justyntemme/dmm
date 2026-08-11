package darkestdungeon

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "262060"
	VortexGameID = "darkestdungeon"
	Name         = "Darkest Dungeon"

	gogID   = "1450711444"
	epicID  = "36cbf259e631478eaac6ea244e55a709"
	modsDir = "mods"

	gogExecutable   = "_windowsnosteam/win64/Darkest.exe"
	steamExecutable = "_windows/win64/Darkest.exe"

	modType = "darkestdungeon-mod"
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
		ExecutableRelative: gogExecutable,
		RequiredFiles:      []string{gogExecutable},
		QueryModPath:       modsDir,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "darkestdungeon-steam-executable",
		Name:               "Darkest Dungeon Steam executable",
		ExecutableRelative: steamExecutable,
		RequiredFiles:      []string{steamExecutable},
		Relative:           true,
		Status:             sdk.CapabilityStatusReady,
		Message:            "DMM discovers Vortex's selected Steam executable when present and can queue it through the Decky extension-tool launch path.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "darkestdungeon-epic-launcher",
		Name:     "Epic Darkest Dungeon launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    epicID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex declares an Epic launcher requirement for the Epic store variant; DMM's MVP Steam Deck path uses the Steam app registration.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "darkestdungeon-gog-discovery",
		Name:     "GOG Darkest Dungeon discovery",
		Launcher: "gog",
		Store:    "gog",
		AppID:    gogID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex includes GOG discovery for Darkest Dungeon; DMM does not execute GOG discovery in the Steam Deck MVP path.",
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modsDir})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:darkestdungeon:project",
		VortexInstallerID: "dd-project-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchProjectArchive,
		CustomBuild:       buildProjectArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:darkestdungeon:no-project",
		VortexInstallerID: "dd-noproject-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNoProjectArchive,
		CustomBuild:       buildNoProjectArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "darkestdungeon-prepare-for-modding",
		Name:    "Prepare Darkest Dungeon mods folder",
		Actions: sdk.EnsureGameDirectories(modsDir, "dlc"),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-darkestdungeon extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-darkestdungeon/src",
	}}
}
