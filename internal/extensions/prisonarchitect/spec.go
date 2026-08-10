package prisonarchitect

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "233450"
	VortexGameID = "prisonarchitect"
	Name         = "Prison Architect"

	linuxExecutable   = "PrisonArchitect"
	windowsExecutable = "Prison Architect64.exe"
	modsRootID        = "prisonarchitect-localappdata-mods"
	modType           = "prisonarchitect-mods"
	versionFilePath   = "Launcher/launcher-settings.json"
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
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  linuxExecutable,
		RequiredFiles:       []string{linuxExecutable},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterInstallPlatform(sdk.InstallPlatformSpec{ID: "native-linux", Name: "Native Linux", Markers: []string{linuxExecutable}})
	r.RegisterInstallPlatform(sdk.InstallPlatformSpec{ID: "proton-windows", Name: "Windows/Proton", Markers: []string{windowsExecutable}})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       modsRootID,
		Name:     "Prison Architect LocalAppData mods",
		Resolver: targetroots.ProtonLocalAppData(SteamAppID, "Introversion", "Prison Architect", "mods"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: modsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:prisonarchitect:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      modsRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "prisonarchitect-ensure-localappdata-mods",
		Name:    "Ensure Prison Architect LocalAppData mods folder exists",
		Actions: sdk.EnsureTargetRootDirectories(modsRootID, "."),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "prisonarchitect-launcher-settings-version",
		Name:     "Prison Architect launcher settings version",
		Provider: gameVersion,
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "prisonarchitect-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex source has a commented Steam launcher probe based on steam_api64.dll. DMM records the launcher fact without forcing a launch-option change.",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "prisonarchitect-native-linux-mod-path",
		Name:    "Prison Architect native Linux mod path verification",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex source uses a LocalAppData-derived mods path. DMM maps that to the Steam Deck Proton prefix; native Linux mod storage must be source-verified before enabling a separate native target root.",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-prisonarchitect extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-prisonarchitect/src",
	})
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, filepath.FromSlash(versionFilePath)))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return sdk.GameVersionResult{}, err
	}
	version := strings.TrimSpace(payload.Version)
	if version == "" {
		return sdk.GameVersionResult{}, nil
	}
	return sdk.GameVersionResult{Version: version, Source: versionFilePath}, nil
}
