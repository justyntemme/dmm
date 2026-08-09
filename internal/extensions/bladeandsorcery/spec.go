package bladeandsorcery

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/loadorderjson"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sharedmodtypes"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "629730"
	VortexGameID = "bladeandsorcery"
	Name         = "Blade & Sorcery"

	executableRelative = "BladeAndSorcery.exe"
	streamingAssets    = "BladeAndSorcery_Data/StreamingAssets"
	officialRoot       = streamingAssets + "/Mods"
	legacyRoot         = streamingAssets
	loadOrderFile      = officialRoot + "/loadorder.json"
	manifestFile       = "manifest.json"
	mulleManifestFile  = "mod.json"

	officialModType = "bas-official-modtype"
	legacyModType   = "bas-legacy-modtype"
	dinputModType   = "dinput"
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
		ExecutableRelative: executableRelative,
		RequiredFiles:      []string{executableRelative},
		QueryModPath:       officialRoot,
		MergeMode:          sdk.GameMergeModeAll,
		RequiresCleanup:    true,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, tool := range supportedTools() {
		r.RegisterSupportedTool(tool)
	}
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "bladeandsorcery-steam-launcher",
		Name:     "Blade & Sorcery Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex declares a Steam launcher requirement when Steam owns the discovered install path.",
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: officialModType, TargetRoot: officialRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: legacyModType, TargetRoot: legacyRoot, Status: sdk.CapabilityStatusMetadata, Message: "Vortex keeps this legacy mod type for pre-8.4 installs and migrations."})
	r.RegisterModType(sharedmodtypes.DInputModTypeSpec())
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bladeandsorcery:mulle-blocked",
		VortexInstallerID: "bas-mulledk19-mod",
		Priority:          20,
		ModType:           legacyModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMulleArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Vortex blocks MulleDK19 Blade & Sorcery mod.json archives because that loader is incompatible with Blade & Sorcery 6.0 and newer.",
		Status:            sdk.CapabilityStatusBlocked,
		Message:           "Vortex blocks MulleDK19 mod.json archives for modern Blade & Sorcery versions.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bladeandsorcery:official",
		VortexInstallerID: "bas-official-mod",
		Priority:          25,
		ModType:           officialModType,
		NameSource:        installplan.NameSourceManifestDisplay,
		CustomMatch:       matchOfficialArchive,
		CustomBuild:       buildOfficialArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(sharedmodtypes.DInputInstaller("vortex:bladeandsorcery:dinput", 50))
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bladeandsorcery-loadorder-json", Name: "Blade & Sorcery loadorder.json"})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "bladeandsorcery-loadorder-page",
		Name:    "Blade & Sorcery load order page",
		Scope:   VortexGameID,
		Status:  sdk.CapabilityStatusBlocked,
		Message: "DMM generates StreamingAssets/Mods/loadorder.json for managed mods, but the Vortex drag/drop load-order page and unmanaged external mod refresh remain blocked until DMM has a generic load-order page runtime.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "bladeandsorcery-view-loadorder-file",
		Name:    "View Blade & Sorcery load order file",
		Scope:   VortexGameID,
		Kind:    "open-directory",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex opens the StreamingAssets/Mods directory from a load-order action; DMM needs the Deck-safe open-directory action runtime.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "bladeandsorcery-prepare-for-modding",
		Name:           "Prepare Blade & Sorcery mod folders",
		GeneratedFiles: []string{streamingAssets, officialRoot, streamingAssets + "/Default"},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: sdk.EventWillDeploy,
		Name:  "Generate Blade & Sorcery loadorder.json",
		Handler: loadorderjson.Handler(loadorderjson.Options{
			ID:                     "bladeandsorcery-loadorder-json",
			TargetRelative:         loadOrderFile,
			EntryRoot:              officialRoot,
			Key:                    "modNames",
			ModTypes:               []string{officialModType},
			ManifestFileName:       manifestFile,
			ManifestParentModTypes: []string{dinputModType},
			ExcludedNames:          []string{"default", "aa", "steamvr"},
			EmptyMessage:           "Generated empty Blade & Sorcery load order because no managed official mods are enabled.",
			SuccessMessage:         "Generated Blade & Sorcery loadorder.json from enabled DMM-managed mods.",
			AlreadyPresentMessage:  "Blade & Sorcery loadorder.json is already up to date.",
		}),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "bladeandsorcery-version",
		Name:    "Blade & Sorcery game/minimum mod version",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex derives game and min-mod versions from the executable or Game.json extracted from bas.jsondb. DMM needs source-backed bas.jsondb extraction/version validation before enforcing this.",
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "bladeandsorcery-version-deploy-validation",
		Name:    "Blade & Sorcery version/mod-type deploy validation",
		Trigger: sdk.EventWillDeploy,
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex blocks deployment when the installed game version changes mod type expectations. DMM needs generic game-version validation before executing this hook.",
	})
	for _, migration := range migrations() {
		r.RegisterStateMigration(migration)
	}
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func supportedTools() []sdk.SupportedToolSpec {
	return []sdk.SupportedToolSpec{
		{
			ID:                 "SteamVR",
			Name:               "Blade and Sorcery (SteamVR)",
			ExecutableRelative: executableRelative,
			RequiredFiles:      []string{executableRelative},
			Arguments:          []string{"-vrmode", "openvr"},
			Relative:           true,
		},
		{
			ID:                 "OculusVR",
			Name:               "Blade and Sorcery (OculusVR)",
			ExecutableRelative: executableRelative,
			RequiredFiles:      []string{executableRelative},
			Arguments:          []string{"-vrmode", "oculus"},
			Relative:           true,
		},
	}
}

func migrations() []sdk.StateMigrationSpec {
	return []sdk.StateMigrationSpec{
		{ID: "bladeandsorcery-migration-0.1.0", Name: "Blade & Sorcery migration guard", FromVersion: "0.0.0", ToVersion: "0.1.0", Status: sdk.CapabilityStatusBlocked, Message: "Vortex suppresses later staged-mod migration for users already on correctly installed pre-0.1.x state; DMM has no released pre-MVP state to migrate."},
		{ID: "bladeandsorcery-migration-0.2.0", Name: "Blade & Sorcery load-order folder migration", FromVersion: "0.1.0", ToVersion: "0.2.0", Status: sdk.CapabilityStatusBlocked, Message: "Vortex migrates staged official mods into per-mod folders for the load-order system. DMM has no released pre-MVP state to migrate."},
		{ID: "bladeandsorcery-migration-0.2.12", Name: "Blade & Sorcery official mod-type migration", FromVersion: "0.2.0", ToVersion: "0.2.12", Status: sdk.CapabilityStatusBlocked, Message: "Vortex purges legacy deployment and retypes old mods when the game version requires official mods. DMM needs generic version-aware mod-type migration runtime before executing this."},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-bladeandsorcery extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-bladeandsorcery/src",
	}}
}
