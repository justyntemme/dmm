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
	registerFNIS(r)
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
		blockedAPI("unfulfilled-rules", "Resolve unfulfilled dependency rules"),
		readyAPI("registerGameInfoProvider", "Register generic game info provider"),
	} {
		r.RegisterExtensionAPI(api)
	}
}

func registerFNIS(r sdk.Registrar) {
	r.RegisterStateReducer(blockedReducer("fnis-settings", "FNIS settings state", "settings/fnis"))
	r.RegisterExtensionSetting(blockedSetting("fnis-integration", "FNIS integration settings", "settings"))
	r.RegisterExtensionToDo(blockedToDo("fnis-install", "Install FNIS integration", "gamemode-activated"))
	r.RegisterExtensionAction(blockedAction("fnis-generate", "Generate FNIS for Users", "mod-icons", "tool"))
	r.RegisterExtensionTest(blockedTest("fnis-integration", "FNIS integration check", "gamemode-activated"))
}

func registerGamebryoSystems(r sdk.Registrar) {
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "gamebryo-incompatible-mod-archives", Name: "Gamebryo incompatible archive check", Trigger: "plugins-changed", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "oblivion-fonts", Name: "Oblivion font settings check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "gamebryo-plugin-index-lock", Name: "Gamebryo plugin index lock state", Scope: "profile_plugin_activations.locked_index", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTableAttribute(blockedTableAttribute("gamebryo-plugin-index-lock", "Gamebryo plugin index lock table attribute", "gamebryo-plugins"))
	r.RegisterStateReducer(blockedReducer("gamebryo-save-session", "Gamebryo savegame session state", "session/saves"))
	r.RegisterStateReducer(blockedReducer("gamebryo-save-settings", "Gamebryo savegame settings state", "settings/saves"))
	r.RegisterExtensionAction(blockedAction("gamebryo-save-transfer", "Transfer save games", "savegames-icons", "savegame"))
	r.RegisterExtensionAction(blockedAction("gamebryo-save-refresh", "Refresh save games", "savegames-icons", "refresh"))
	r.RegisterExtensionAction(blockedAction("gamebryo-save-open", "Open save games", "savegames-icons", "open"))
	r.RegisterExtensionMainPage(blockedMainPage("gamebryo-savegames", "Save games", "savegame"))
	r.RegisterProfileFeature(sdk.ProfileFeatureSpec{ID: "gamebryo-savegames", Name: "Gamebryo savegame profile feature", Status: sdk.CapabilityStatusBlocked, Message: blockedMessage})
	r.RegisterExtensionAPI(readyAPI("oblivion-font-repair", "Oblivion font settings automatic repair"))
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "skyrim-fonts", Name: "Skyrim font settings check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionMainPage(blockedMainPage("morrowind-plugins", "Morrowind plugins", "plugins"))
	r.RegisterHistoryStack(sdk.HistoryStackSpec{
		ID:      "plugins",
		Name:    "Gamebryo plugin history stack",
		Scope:   "gamebryo-plugins",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
}

func registerDependencyManager(r sdk.Registrar) {
	r.RegisterStateReducer(blockedReducer("dependency-workarounds", "Dependency workaround settings", "settings/workarounds"))
	r.RegisterStateReducer(blockedReducer("dependency-session", "Dependency connection state", "session/dependencies"))
	r.RegisterExtensionTableAttribute(blockedTableAttribute("dependency-load-order", "Dependency load-order table attribute", "mods"))
	r.RegisterExtensionTableAttribute(blockedTableAttribute("dependency-rules", "Dependency rules table attribute", "mods"))
	r.RegisterExtensionAction(blockedAction("dependency-manage-rules", "Manage dependency rules", "mod-icons", "rules"))
	r.RegisterExtensionDialog(blockedDialog("dependency-connector", "Dependency connector dialog", "dependencies"))
	r.RegisterExtensionDialog(blockedDialog("dependency-editor", "Dependency editor dialog", "dependencies"))
	r.RegisterExtensionDialog(blockedDialog("conflict-editor", "Conflict editor dialog", "dependencies"))
	r.RegisterExtensionDialog(blockedDialog("dependency-cycle-graph", "Dependency cycle graph dialog", "dependencies"))
	r.RegisterExtensionDialog(blockedDialog("file-override-editor", "File override editor dialog", "dependencies"))
	r.RegisterExtensionControlWrapper(sdk.ExtensionControlWrapperSpec{
		ID:       "dependency-mod-name-wrapper",
		Name:     "Dependency mod-name control wrapper",
		Target:   "mods-name",
		Priority: 100,
		Status:   sdk.CapabilityStatusBlocked,
		Message:  blockedMessage,
	})
	r.RegisterExtensionSetting(blockedSetting("dependency-workarounds", "Dependency workarounds settings", "settings"))
	r.RegisterExtensionTest(blockedTest("dependency-unsolved-conflicts", "Unsolved dependency conflicts check", "gamemode-activated"))
	r.RegisterStartHook(sdk.StartHookSpec{
		ID:       "dependency-check-unsolved-conflicts",
		Name:     "Check unsolved dependency conflicts",
		Trigger:  "startup",
		Priority: 50,
		Status:   sdk.CapabilityStatusBlocked,
		Message:  blockedMessage,
	})
}

func registerLocalGameSettings(r sdk.Registrar) {
	r.RegisterProfileFeature(sdk.ProfileFeatureSpec{ID: "local-game-settings", Name: "Local game settings profile feature", Status: sdk.CapabilityStatusBlocked, Message: blockedMessage})
	r.RegisterExtensionTest(blockedTest("local-game-settings-global-files", "Global local game settings check", "gamemode-activated"))
}

func registerNewFileMonitor(r sdk.Registrar) {
	r.RegisterExtensionAPI(blockedAPI("new-file-adoption", "Adopt newly created game files into a managed mod"))
}

func registerVortexTests(r sdk.Registrar) {
	r.RegisterStateReducer(blockedReducer("test-gameversion-state", "Vortex game-version test state", "persistent/gameMode"))
	r.RegisterExtensionTest(blockedTest("game-version-gamemode", "Game version check on game mode activation", "gamemode-activated"))
	r.RegisterExtensionTest(blockedTest("game-version-mod-installed", "Game version check after mod install", "mod-installed"))
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
	r.RegisterExtensionTest(blockedTest("test-setup-uninstall-entry", "Vortex setup uninstall-entry test", "startup"))
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
