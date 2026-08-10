package pillarsofeternity2

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "560130"
	VortexGameID = "pillarsofeternity2"
	Name         = "Pillars of Eternity II: Deadfire"

	msStoreAppID = "VersusEvil.PillarsofEternity2-PC"

	overrideRootID = "poe2-override"
	configRootID   = "poe2-locallow-config"

	modType = "poe2-override-mod"
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
		ExecutableRelative:  "PillarsOfEternityII.exe",
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeNone,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       overrideRootID,
		Name:     "Pillars II override folder",
		Resolver: overrideRoot,
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       configRootID,
		Name:     "Pillars II LocalLow config",
		Resolver: localLowConfigRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRootID: overrideRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:pillarsofeternity2:override",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchOverrideArchive,
		CustomBuild:       buildOverrideArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "poe2-modconfig-load-order",
		Name:           "Pillars II modconfig.json",
		TargetRelative: modConfigFile,
		TargetRootID:   configRootID,
		ModTypes:       []string{modType},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Generate Pillars II modconfig.json",
		Handler: willDeploy,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "poe2-prepare-modding",
		Name: "Prepare Pillars II override and modconfig paths",
		Actions: append(
			sdk.EnsureGameDirectories("PillarsOfEternityII_Data/override"),
			sdk.EnsureTargetRootFiles(configRootID, "{\n  \"Entries\": []\n}\n", modConfigFile)...,
		),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "poe2-xbox-launcher",
		Name:     "Xbox launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    msStoreAppID,
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "App",
		}},
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex launches the Microsoft Store version through Xbox app metadata. DMM's Steam Deck MVP targets the Steam executable and Proton config path.",
	})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{
		ID:      "poe2-load-order-page",
		Name:    "Pillars II Load Order",
		Scope:   "game",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes modconfig.json order through generic extension load-order profile controls.",
	})
	r.RegisterAttributeExtractor(sdk.AttributeExtractorSpec{
		ID:     "poe2-manifest-version",
		Name:   "Pillars II manifest game version range",
		Target: "mods",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func overrideRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.TargetRootResult{}, errors.New("game path is required to resolve Pillars II override folder")
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(gamePath, modPathForGamePath(gamePath)),
		Source: "Vortex queryModPath",
	}, nil
}

func localLowConfigRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return sdk.TargetRootResult{}, errors.New("Steam library path is required to resolve Pillars II Proton LocalLow config path")
	}
	return sdk.TargetRootResult{
		Path: filepath.Join(
			libraryPath,
			"steamapps",
			"compatdata",
			SteamAppID,
			"pfx",
			"drive_c",
			"users",
			"steamuser",
			"AppData",
			"LocalLow",
			"Obsidian Entertainment",
			"Pillars of Eternity II",
		),
		Source: "Vortex LocalLow path via Steam Proton prefix",
	}, nil
}

func modPathForGamePath(gamePath string) string {
	if strings.Contains(strings.ToLower(filepath.ToSlash(gamePath)), "modifiablewindowsapps") {
		return filepath.Join("PillarsOfEternity2_Data", "override")
	}
	return filepath.Join("PillarsOfEternityII_Data", "override")
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-pillarsofeternity2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-pillarsofeternity2/src",
	}}
}
