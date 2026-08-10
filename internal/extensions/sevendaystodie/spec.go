package sevendaystodie

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "251570"
	VortexGameID = "7daystodie"
	Name         = "7 Days to Die"

	gameExecutable = "7DaysToDie.exe"
	modsRoot       = "Mods"
	modsRootID     = "7daystodie-mods-root"
	udfSettingID   = "7daystodie-udf"
	modInfoName    = "modinfo.xml"

	modletModType = "7dtd-mod"
	rootModType   = "7dtd-root-mod"
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
		ExecutableRelative: gameExecutable,
		QueryModPath:       modsRoot,
		MergeMode:          sdk.GameMergeModeDynamic,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       modsRootID,
		Name:     "7 Days to Die Mods folder",
		Resolver: modsTargetRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: modletModType, TargetRootID: modsRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:7daystodie:root-mod",
		VortexInstallerID: rootModType,
		Priority:          20,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchRootModArchive,
		CustomBuild:       buildRootModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:7daystodie:modlet",
		VortexInstallerID: modletModType,
		Priority:          25,
		ModType:           modletModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModletArchive,
		CustomBuild:       buildModletArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:                "7daystodie-folder-prefix-order",
		Name:              "7 Days to Die folder prefix load order",
		ModTypes:          []string{modletModType},
		Message:           "DMM applies deterministic folder prefixes from profile priority during deployment.",
		UsageInstructions: "Move profile mods up or down to change generated 7 Days to Die folder prefixes.",
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "7daystodie-folder-prefix-order", Name: "7 Days to Die folder prefix merge"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Apply 7 Days to Die load-order folder prefixes",
		Handler: loadOrderPrefixHandler,
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "7daystodie-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex requests Steam launcher behavior when steamclient64.dll is present in the game folder.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "7daystodie-user-data-folder",
		Name:    "Configure 7 Days to Die user data folder",
		Actions: sdk.EnsureTargetRootDirectories(modsRootID, "."),
	})
	r.RegisterLaunchOptionRequirement(sdk.LaunchOptionRequirementSpec{
		ID:       "7daystodie-user-data-folder-argument",
		Name:     "7 Days to Die User Data Folder launch argument",
		Mode:     sdk.LaunchOptionModeDefaultArguments,
		Provider: udfLaunchOptionRequirement,
		Message:  "When a User Data Folder is configured, Steam launch options must pass it to 7 Days to Die.",
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:      udfSettingID,
		Name:    "7 Days to Die User Data Folder",
		Scope:   "game",
		Message: "Optional absolute User Data Folder path. If unset, DMM follows Vortex's fallback game-root Mods path.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "7daystodie-prefix-offset",
		Name:    "Prefix Offset Assign",
		Scope:   "profile",
		Kind:    "load-order",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex exposes prefix offset actions. DMM applies deterministic profile-priority prefixes but does not yet expose a user offset action.",
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "7daystodie-settings-state",
		Name:    "7 Days to Die extension settings",
		Scope:   "profile",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex stores User Data Folder and previous load-order state in extension settings.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modsTargetRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	if udf, ok, err := configuredUDF(input.ExtensionSettings); err != nil {
		return sdk.TargetRootResult{}, err
	} else if ok {
		return sdk.TargetRootResult{Path: filepath.Join(udf, modsRoot), Source: "Vortex 7 Days to Die User Data Folder setting"}, nil
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.TargetRootResult{}, errors.New("game path is required to resolve 7 Days to Die Mods folder")
	}
	return sdk.TargetRootResult{Path: filepath.Join(gamePath, modsRoot), Source: "Vortex fallback game-root Mods path"}, nil
}

func udfLaunchOptionRequirement(ctx context.Context, input sdk.LaunchOptionInput) (sdk.LaunchOptionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.LaunchOptionResult{}, err
	}
	udf, ok, err := configuredUDF(input.ExtensionSettings)
	if err != nil || !ok {
		return sdk.LaunchOptionResult{}, err
	}
	return sdk.LaunchOptionResult{
		Required:  true,
		Arguments: []string{`-UserDataFolder="` + filepath.ToSlash(udf) + `"`},
		Details:   []string{"7 Days to Die User Data Folder launch argument is derived from the extension setting."},
		Source:    "Vortex User Data Folder setting",
	}, nil
}

func configuredUDF(settings map[string]map[string]json.RawMessage) (string, bool, error) {
	extensionSettings := settings[strings.ToLower(VortexGameID)]
	if len(extensionSettings) == 0 {
		return "", false, nil
	}
	raw := extensionSettings[strings.ToLower(udfSettingID)]
	if len(raw) == 0 || string(raw) == "null" {
		return "", false, nil
	}
	var pathValue string
	if err := json.Unmarshal(raw, &pathValue); err != nil {
		var object struct {
			Path string `json:"path"`
		}
		if objectErr := json.Unmarshal(raw, &object); objectErr != nil {
			return "", false, err
		}
		pathValue = object.Path
	}
	clean := filepath.Clean(strings.TrimSpace(pathValue))
	if clean == "" || clean == "." {
		return "", false, nil
	}
	if !filepath.IsAbs(clean) {
		return "", false, errors.New("7 Days to Die User Data Folder must be an absolute path")
	}
	if strings.Contains(strings.ToLower(filepath.ToSlash(clean)), "vortex") {
		return "", false, errors.New("7 Days to Die User Data Folder must not be inside Vortex directories")
	}
	if strings.EqualFold(filepath.Base(clean), modsRoot) {
		clean = filepath.Dir(clean)
	}
	return clean, true, nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-7daystodie extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-7daystodie/src",
	}}
}
