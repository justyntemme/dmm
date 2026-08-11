package starbound

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "211820"
	VortexGameID = "starbound"
	Name         = "Starbound"

	executable = "win64/starbound.exe"
	modRoot    = "mods"
	modType    = "starbound-mods"
	xboxAppID  = "Chucklefish.StarboundWindows10Edition"
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
		ExecutableVariants: []sdk.GameExecutableVariantSpec{{
			ID:                 "xbox-modifiable-windows-apps",
			Name:               "Xbox ModifiableWindowsApps executable",
			ExecutableRelative: "win/starbound.exe",
			RequiredFiles:      []string{"win/starbound.exe", "assets/packed.pak", "assets/user/songs/12 Days Of Christmas.abc"},
			GamePathContains:   []string{"modifiablewindowsapps"},
		}},
		RequiredFiles: []string{"assets/packed.pak", "assets/user/songs/12 Days Of Christmas.abc"},
		QueryModPath:  modRoot,
		MergeMode:     sdk.GameMergeModeAll,
		Environment:   map[string]string{"SteamAPPId": SteamAppID},
		Deployment:    installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:starbound:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "starbound-ensure-mods-folder",
		Name:    "Ensure Starbound mods folder exists",
		Actions: sdk.EnsureGameDirectories(modRoot),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "starbound-xbox-launcher",
		Name:     "Xbox app launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    xboxAppID,
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "StarboundClient",
		}},
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: "Vortex uses Xbox launcher metadata for the Microsoft Store version. DMM's Steam Deck MVP runtime manages the Steam executable path; Xbox launcher integration is a future non-Steam store boundary.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-starbound extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-starbound/src",
	})
}
