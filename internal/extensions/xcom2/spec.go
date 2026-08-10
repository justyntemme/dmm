package xcom2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "268500"
	VortexGameID = "xcom2"
	WOTCGameID   = "xcom2-wotc"
	Name         = "XCOM 2"

	xcom2Mods   = "XComGame/Mods"
	xcom2Config = "XComGame/Config"
	wotcMods    = "XCom2-WarOfTheChosen/XComGame/Mods"
	wotcConfig  = "XCom2-WarOfTheChosen/XComGame/Config"

	modExt     = ".XComMod"
	optionsINI = "DefaultModOptions.ini"

	xcom2ModType = "xcom2-mod"
	wotcModType  = "xcom2-wotc-mod"
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
		ExecutableRelative:  "Binaries/Win64/XCom2.exe",
		RequiredFiles:       []string{"XComGame", "XComGame/CookedPCConsole/3DUIBP.upk", "XComGame/CharacterPool/Importable/Demos&Replays.bin"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		CompatibleDownloads: []string{WOTCGameID},
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: xcom2ModType, TargetRoot: xcom2Mods})
	r.RegisterModType(installplan.ModTypeSpec{ID: wotcModType, TargetRoot: wotcMods})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:xcom2:xcommod",
		VortexInstallerID: "xcom2-installer",
		Priority:          25,
		ModType:           xcom2ModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchXComArchive,
		CustomBuild:       buildXComArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "xcom2-default-mod-options", Name: "XCOM 2 DefaultModOptions.ini"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Generate XCOM 2 DefaultModOptions.ini",
		Handler: willDeploy,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "xcom2-prepare-mods-folder",
		Name:    "Prepare XCOM 2 mods folder",
		Actions: sdk.EnsureGameDirectories(xcom2Mods, wotcMods, xcom2Config, wotcConfig),
	})
	registerSupportedTools(r)
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "xcom2-load-order-page",
		Name:    "XCOM 2 Load Order",
		Scope:   "game",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex exposes separate load-order pages for base XCOM 2 and War of the Chosen. DMM generates DefaultModOptions.ini from profile priority today; generic multi-variant load-order UI remains to be implemented.",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "xcom2-multi-variant-selector",
		Name:    "XCOM 2 base/WOTC variant selector",
		Trigger: "game-selection",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex registers xcom2 and xcom2-wotc as separate logical games for Steam app 268500. DMM currently detects WOTC when present and needs a generic multi-logical-game-per-app selector for exact UI parity.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func registerSupportedTools(r sdk.Registrar) {
	for _, tool := range []sdk.SupportedToolSpec{
		{
			ID:                 "xcom2-launcher",
			Name:               "XCOM 2 Launcher",
			ExecutableRelative: "Launcher/launcher.exe",
			RequiredFiles:      []string{"Launcher/launcher.exe"},
			Relative:           true,
		},
		{
			ID:                 "xcom2-devtools",
			Name:               "XCOM 2 ModBuddy",
			ExecutableRelative: "Binaries/Win32/ModBuddy/XCOM ModBuddy.exe",
			RequiredFiles:      []string{"Binaries/Win32/ModBuddy/XCOM ModBuddy.exe"},
			Status:             sdk.CapabilityStatusMetadata,
			Message:            "Vortex discovers XCOM 2 SDK through Steam app 299990. DMM records the tool metadata until external tool discovery is executable.",
		},
		{
			ID:                 "xcom2-wotc-launcher",
			Name:               "XCOM 2: War of the Chosen Launcher",
			ExecutableRelative: "Launcher/launcher.exe",
			RequiredFiles:      []string{"Launcher/launcher.exe"},
			Relative:           true,
		},
		{
			ID:                 "xcom2-wotc-devtools",
			Name:               "XCOM 2: War of the Chosen ModBuddy",
			ExecutableRelative: "Binaries/Win32/ModBuddy/XCOM ModBuddy.exe",
			RequiredFiles:      []string{"Binaries/Win32/ModBuddy/XCOM ModBuddy.exe"},
			Status:             sdk.CapabilityStatusMetadata,
			Message:            "Vortex discovers WOTC SDK through Steam app 602410. DMM records the tool metadata until external tool discovery is executable.",
		},
	} {
		r.RegisterSupportedTool(tool)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-xcom2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-xcom2/src",
	}}
}
