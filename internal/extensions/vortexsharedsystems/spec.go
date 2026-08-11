package vortexsharedsystems

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ID      = "vortex-shared-systems"
	Name    = "Vortex Shared System Extensions"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source defines this shared system behavior, but DMM has not implemented the reusable runtime for it yet."

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	registerCrossExtensionAPIs(r)
	registerGamebryoSystems(r)
	registerDependencyManager(r)
	registerLocalGameSettings(r)
	registerNewFileMonitor(r)
	registerVortexTests(r)
}

func registerCrossExtensionAPIs(r sdk.Registrar) {
	for _, api := range []sdk.ExtensionAPISpec{
		readyAPI("deploy-single-mod", "Deploy one mod through the deployment pipeline"),
		readyAPI("purge-mods-in-path", "Purge managed mods under a path"),
		readyAPI("browse-for-download", "Open a source-backed download browser"),
		readyAPI("nexus-download", "Queue a source-backed Nexus manager download"),
		readyAPI("discover-tools", "Discover extension-declared and DMM-managed external tools"),
		readyAPI("bake-settings", "Bake profile-local game settings"),
		readyAPI("unfulfilled-rules", "Resolve unfulfilled dependency rules"),
		readyAPI("registerGameInfoProvider", "Register generic game info provider"),
	} {
		r.RegisterExtensionAPI(api)
	}
}

func registerGamebryoSystems(r sdk.Registrar) {
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "gamebryo-incompatible-mod-archives", Name: "Gamebryo incompatible archive check", Trigger: "plugins-changed", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "oblivion-fonts", Name: "Oblivion font settings check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "gamebryo-plugin-index-lock", Name: "Gamebryo plugin index lock state", Scope: "profile_plugin_activations.locked_index", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "gamebryo-plugin-index-lock",
		Name:    "Gamebryo plugin index lock table attribute",
		Target:  "gamebryo-plugins",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes Vortex's lock-index state through profile plugin activation rows and applies it during generated load-order output; the exact Vortex table widget is represented by DMM's plugin activation update API/UI.",
	})
	r.RegisterProfileFeature(sdk.ProfileFeatureSpec{
		ID:      "local_saves",
		Name:    "Gamebryo local save paths",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM supports Vortex-style Gamebryo local save path redirection through extension-declared profile-file INI patches and the savegame management runtime.",
	})
	savegameMessage := "DMM exposes Vortex-equivalent Gamebryo savegame management through extension-declared savegame specs plus profile APIs for list, transfer/copy, delete, and restore-plugin-order operations."
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "gamebryo-save-session", Name: "Gamebryo savegame session state", Scope: "api/profile-savegames", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "gamebryo-save-settings", Name: "Gamebryo savegame settings state", Scope: "extension-savegame-spec", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-transfer", Name: "Transfer save games", Scope: "profile-savegames", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-refresh", Name: "Refresh save games", Scope: "profile-savegames", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-open", Name: "Open save games", Scope: "profile-savegames", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{ID: "gamebryo-savegames", Name: "Save games", Scope: "profile-savegames", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterProfileFeature(sdk.ProfileFeatureSpec{ID: "gamebryo-savegames", Name: "Gamebryo savegame profile feature", Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAPI(readyAPI("oblivion-font-repair", "Oblivion font settings automatic repair"))
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "skyrim-fonts", Name: "Skyrim font settings check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionMainPage(sdk.ExtensionMainPageSpec{
		ID:      "morrowind-plugins",
		Name:    "Morrowind plugins",
		Scope:   "plugins",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM represents Vortex's Morrowind plugin page through the Morrowind extension's ESP/ESM activation runtime, generated Morrowind.ini Game Files output, timestamp ordering, and generic profile plugin-order UI/API instead of copying Vortex's desktop page.",
	})
	r.RegisterHistoryStack(sdk.HistoryStackSpec{
		ID:      "plugins",
		Name:    "Gamebryo plugin history stack",
		Scope:   "gamebryo-plugins",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM represents Vortex's Gamebryo plugin history stack with profile plugin activation state plus deployment history. Plugin enable/disable/order changes are applied through the profile activation APIs, create normal deployment history entries, and can be previewed/restored through DMM's deployment history endpoints instead of Vortex's desktop undo stack.",
	})
}

func registerDependencyManager(r sdk.Registrar) {
	dependencyRuleMessage := "DMM supports Vortex-style before/after/conflicts profile mod rules through profile rule APIs, cycle rejection, and profile priority normalization during rule updates."
	dependencyDialogMessage := dependencyRuleMessage + " DMM maps Vortex's connector/editor dialogs to the phone/tablet profile rule editor and Decky/Action Center conflict prompts instead of embedding Vortex's desktop React dialogs."
	conflictDialogMessage := "DMM supports Vortex-style managed file conflict resolution through duplicate-target deploy blocking, profile-scoped file-winner APIs, startup conflict notices, and the profile deploy preview/read model."
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "dependency-workarounds", Name: "Dependency workaround settings", Scope: "profile-mod-rules", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "dependency-session", Name: "Dependency connection state", Scope: "profile-mod-rules", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{ID: "dependency-load-order", Name: "Dependency load-order table attribute", Target: "mods", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{ID: "dependency-rules", Name: "Dependency rules table attribute", Target: "mods", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "dependency-manage-rules", Name: "Manage dependency rules", Scope: "profile-mod-rules", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "dependency-connector", Name: "Dependency connector dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: dependencyDialogMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "dependency-editor", Name: "Dependency editor dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: dependencyDialogMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "conflict-editor", Name: "Conflict editor dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: conflictDialogMessage})
	r.RegisterExtensionDialog(blockedDialog("dependency-cycle-graph", "Dependency cycle graph dialog", "dependencies"))
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "file-override-editor", Name: "File override editor dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: conflictDialogMessage + " DMM maps Vortex's file-override modal to file-winner selection routes and advanced conflict review UI."})
	r.RegisterExtensionControlWrapper(sdk.ExtensionControlWrapperSpec{
		ID:       "dependency-mod-name-wrapper",
		Name:     "Dependency mod-name control wrapper",
		Target:   "mods-name",
		Priority: 100,
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM maps Vortex's mod-name dependency wrapper to profile mod rows that expose rule/conflict badges, unresolved dependency notices, and file-conflict actions from the same profile rule/conflict read models.",
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:      "dependency-workarounds",
		Name:    "Dependency workarounds settings",
		Scope:   "profile-mod-rules",
		Status:  sdk.CapabilityStatusReady,
		Message: dependencyRuleMessage + " DMM stores these as profile-scoped mod rules instead of a Vortex global settings page.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "dependency-unsolved-conflicts", Name: "Unsolved dependency conflicts check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterStartHook(sdk.StartHookSpec{
		ID:       "dependency-check-unsolved-conflicts",
		Name:     "Check unsolved dependency conflicts",
		Trigger:  sdk.StartHookTriggerStartup,
		Kind:     sdk.StartHookKindCheckUnresolvedConflicts,
		Priority: 50,
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM checks enabled profile deployment plans for unresolved duplicate managed file targets at startup and queues an Action Center notice when a file winner is required.",
	})
}

func registerLocalGameSettings(r sdk.Registrar) {
	r.RegisterProfileFeature(sdk.ProfileFeatureSpec{
		ID:      "local_game_settings",
		Name:    "Local game settings profile feature",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM supports Vortex-style profile-local game settings through extension-declared profile files, per-profile feature state, profile-switch sync, and the bake-settings lifecycle event.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "local-game-settings-global-files",
		Name:    "Global local game settings check",
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM reports missing required profile-local settings files from extension-declared profile-file metadata in game diagnostics.",
	})
}

func registerNewFileMonitor(r sdk.Registrar) {
	r.RegisterExtensionAPI(readyAPI("new-file-single-owner-adoption", "Adopt single-owner generated game files into a managed mod"))
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "new-file-ambiguous-adoption",
		Name:    "Resolve ambiguous generated file adoption",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM passes Vortex-style candidate owner lists to extension added-files/removed-files handlers and persists extension-selected adopted files into managed staging. A generic user-facing unmanaged adoption wizard remains a separate product UI feature.",
	})
}

func registerVortexTests(r sdk.Registrar) {
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "test-gameversion-state",
		Name:    "Vortex game-version test state",
		Scope:   "persistent/gameMode",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM persists the last observed game version separately from Steam discovery state and reports a Vortex-style warning when the installed game version changes.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "game-version-gamemode",
		Name:    "Game version check on game mode activation",
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM reports installed mods whose extension-extracted min/max game-version metadata is incompatible with the detected game version.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "game-version-mod-installed",
		Name:    "Game version check after mod install",
		Trigger: "mod-installed",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM queues an Action Center notice after mod install when extension-extracted min/max game-version metadata is incompatible with the detected game version.",
	})
	r.RegisterGameInfoProvider(sdk.GameInfoProviderSpec{
		ID:           "game-version",
		Name:         "Vortex game version info provider",
		Tags:         []string{"game_version"},
		CacheSeconds: 300,
		Priority:     15,
		Provider: func(_ context.Context, input sdk.GameInfoInput) (sdk.GameInfoResult, error) {
			version := strings.TrimSpace(input.GameVersion)
			if version == "" {
				return sdk.GameInfoResult{}, nil
			}
			return sdk.GameInfoResult{Details: []sdk.GameInfoDetail{{
				ID:     "game_version",
				Title:  "Installed Version",
				Value:  version,
				Source: "game-version",
			}}}, nil
		},
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "test-setup-uninstall-entry",
		Name:    "Vortex setup uninstall-entry test",
		Trigger: "startup",
		Status:  sdk.CapabilityStatusReady,
		Message: "Verified non-applicable: Vortex registers this test only for Windows installer registry state. DMM is delivered as a Decky plugin on SteamOS and has no Windows uninstall registry entry to validate.",
	})
}

func blockedAPI(id, name string) sdk.ExtensionAPISpec {
	return sdk.ExtensionAPISpec{ID: id, Name: name, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func readyAPI(id, name string) sdk.ExtensionAPISpec {
	return sdk.ExtensionAPISpec{ID: id, Name: name, Status: sdk.CapabilityStatusReady}
}

func blockedAction(id, name, scope, kind string) sdk.ExtensionActionSpec {
	return sdk.ExtensionActionSpec{ID: id, Name: name, Scope: scope, Kind: kind, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedSetting(id, name, scope string) sdk.ExtensionSettingSpec {
	return sdk.ExtensionSettingSpec{ID: id, Name: name, Scope: scope, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedTest(id, name, trigger string) sdk.ExtensionTestSpec {
	return sdk.ExtensionTestSpec{ID: id, Name: name, Trigger: trigger, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedToDo(id, name, trigger string) sdk.ExtensionToDoSpec {
	return sdk.ExtensionToDoSpec{ID: id, Name: name, Trigger: trigger, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedReducer(id, name, scope string) sdk.StateReducerSpec {
	return sdk.StateReducerSpec{ID: id, Name: name, Scope: scope, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedTableAttribute(id, name, target string) sdk.ExtensionTableAttributeSpec {
	return sdk.ExtensionTableAttributeSpec{ID: id, Name: name, Target: target, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedDialog(id, name, scope string) sdk.ExtensionDialogSpec {
	return sdk.ExtensionDialogSpec{ID: id, Name: name, Scope: scope, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedMainPage(id, name, scope string) sdk.ExtensionMainPageSpec {
	return sdk.ExtensionMainPageSpec{ID: id, Name: name, Scope: scope, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func blockedEventHandler(id, event, name string) sdk.EventHandlerSpec {
	return sdk.EventHandlerSpec{ID: id, Event: event, Name: name, Status: sdk.CapabilityStatusBlocked, Message: blockedMessage}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex FNIS integration source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/fnis-integration/src/index.ts"},
		{Name: "Vortex Gamebryo archive check source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-check/src/index.ts"},
		{Name: "Vortex Gamebryo plugin index lock source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-plugin-indexlock/src/index.tsx"},
		{Name: "Vortex Gamebryo savegame management source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-savegame-management/src/index.ts"},
		{Name: "Vortex Gamebryo test settings source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-test-settings/src/index.ts"},
		{Name: "Vortex local game settings source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/local-gamesettings/src/index.ts"},
		{Name: "Vortex mod dependency manager source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mod-dependency-manager/src/index.tsx"},
		{Name: "Vortex Morrowind plugin management source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/morrowind-plugin-management/src/index.ts"},
		{Name: "Vortex new-file monitor source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/new-file-monitor/src/index.ts"},
		{Name: "Vortex game-version test source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/test-gameversion/src/index.ts"},
		{Name: "Vortex setup test source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/test-setup/src/index.ts"},
	}
}
