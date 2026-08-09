package darksouls

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
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
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex requests Steam launcher behavior for the Steam copy of Dark Souls.",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "darksouls-dsfix-requirement",
		Name:    "Dark Souls DSfix setup prompt",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex prompts the user to install DSfix from Nexus mod 19 when DATA/dsfix/tex_override is missing. DMM needs a reusable tool/mod requirement action before automating that prompt.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-darksouls extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-darksouls/src",
	})
}
