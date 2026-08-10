package sevendaystodie

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "251570"
	VortexGameID = "7daystodie"
	Name         = "7 Days to Die"

	gameExecutable = "7DaysToDie.exe"
	modsRoot       = "Mods"
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
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: modletModType, TargetRoot: modsRoot})
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
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "7daystodie-folder-prefix-order", Name: "7 Days to Die folder prefix load order"})
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
		Actions: sdk.EnsureGameDirectories("Mods"),
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:      "7daystodie-udf",
		Name:    "7 Days to Die User Data Folder",
		Scope:   "game",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex can prompt for a custom User Data Folder and write launchersettings.json. DMM currently follows Vortex's fallback Mods path until a generic extension settings/runtime setup flow exists.",
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

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-7daystodie extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-7daystodie/src",
	}}
}
