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

	gameExecutable        = "7DaysToDie.exe"
	modsRoot              = "Mods"
	modsRootID            = "7daystodie-mods-root"
	udfSettingID          = "7daystodie-udf"
	prefixOffsetSettingID = "7daystodie-prefix-offset"
	modInfoName           = "modinfo.xml"

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
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM evaluates Vortex's Steam launcher behavior against the discovered Steam app and reports it through launcher diagnostics.",
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
		ID:          udfSettingID,
		Name:        "7 Days to Die User Data Folder",
		Scope:       "game",
		ValueType:   sdk.ExtensionSettingValuePath,
		Placeholder: "/home/deck/.local/share/7DaysToDie",
		Message:     "Optional absolute User Data Folder path. If unset, DMM follows Vortex's fallback game-root Mods path.",
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:           prefixOffsetSettingID,
		Name:         "7 Days to Die Prefix Offset",
		Scope:        "profile",
		ValueType:    sdk.ExtensionSettingValueNumber,
		DefaultValue: json.RawMessage("0"),
		Placeholder:  "0",
		Message:      "Profile-specific numeric offset for generated modlet folder prefixes. Vortex prompts for AAA-ZZZ; DMM stores the equivalent numeric offset and applies makePrefix(priority + offset).",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "7daystodie-prefix-offset-reset",
		Name:    "Reset Prefix Offset",
		Scope:   VortexGameID,
		Kind:    sdk.ExtensionActionKindSetSetting,
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's Prefix Offset Reset action by resetting the active profile's generated modlet prefix offset to zero.",
		SetSetting: &sdk.SetExtensionSettingActionSpec{
			SettingID: prefixOffsetSettingID,
			Value:     json.RawMessage("0"),
		},
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "7daystodie-settings-state",
		Name:    "7 Days to Die extension settings",
		Scope:   "profile",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM stores Vortex-equivalent profile prefix offset state through profile-scoped extension settings.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "7daystodie-0.2.0-reinstall-warning",
		Name:        "7 Days to Die v17 reinstall warning",
		FromVersion: "0.0.0",
		ToVersion:   "0.2.0",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:      "warn-installed-mods",
			Name:    "Warn when historical 7 Days to Die mods exist",
			Command: sdk.StateMigrationCommandWarnInstalled,
			Message: "7 Days to Die version 17 changed the mod install layout; historical mods should be reinstalled.",
		}},
		Message: "Mirrors Vortex 0.2.0 migration by warning when any historical 7 Days to Die mods exist and need reinstall after the v17 layout change.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "7daystodie-1.0.0-load-order-migration",
		Name:        "7 Days to Die load-order file migration",
		FromVersion: "0.2.0",
		ToVersion:   "1.0.0",
		Commands: []sdk.StateMigrationCommandSpec{
			{
				ID:           "serialize-existing-load-order",
				Name:         "Serialize existing 7 Days to Die load order",
				Command:      sdk.StateMigrationCommandSerializeState,
				MetadataKind: "load-order",
			},
			{
				ID:           "purge-old-mods-root",
				Name:         "Purge old 7 Days to Die Mods deployment",
				Command:      sdk.StateMigrationCommandPurgeModsInPath,
				TargetRootID: modsRootID,
			},
			{
				ID:      "redeploy-active-profile",
				Name:    "Redeploy active 7 Days to Die profile",
				Command: sdk.StateMigrationCommandDeployProfile,
			},
		},
		Message: "Source-backed Vortex migration serializes profile load-order state, purges the old Mods deployment, and marks deployment necessary. DMM represents this with the generic purge/redeploy migration commands for imported Vortex state.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "7daystodie-1.0.11-load-order-location-migration",
		Name:        "7 Days to Die old load-order file cleanup",
		FromVersion: "1.0.0",
		ToVersion:   "1.0.11",
		Commands: []sdk.StateMigrationCommandSpec{
			{
				ID:           "serialize-existing-load-order",
				Name:         "Serialize existing 7 Days to Die load order",
				Command:      sdk.StateMigrationCommandSerializeState,
				MetadataKind: "load-order",
			},
			{
				ID:           "purge-old-mods-root",
				Name:         "Purge old 7 Days to Die Mods deployment",
				Command:      sdk.StateMigrationCommandPurgeModsInPath,
				TargetRootID: modsRootID,
			},
			{
				ID:      "redeploy-active-profile",
				Name:    "Redeploy active 7 Days to Die profile",
				Command: sdk.StateMigrationCommandDeployProfile,
			},
		},
		Message: "Source-backed Vortex migration moves old per-profile load-order files, removes the obsolete game-root JSON files, purges the old Mods deployment, and redeploys. DMM-created state already stores profile order internally; Vortex import must preserve the cleanup obligation.",
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
