package vortexsharedsystems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ID      = "vortex-shared-systems"
	Name    = "Vortex Shared System Extensions"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

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
	registerSteamGameInfo(r)
	registerVortexTests(r)
	registerImportAndMetadataSurfaces(r)
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
		{
			ID:      "isBlueprintPlugin",
			Name:    "Detect Starfield blueprint plugin files",
			Status:  sdk.CapabilityStatusReady,
			Message: "DMM mirrors Vortex's Gamebryo isBlueprintPlugin API through the shared Gamebryo plugin header parser: only Starfield blueprint-flagged plugin files return true, and unreadable/non-Starfield files return false.",
		},
	} {
		r.RegisterExtensionAPI(api)
	}
}

func registerGamebryoSystems(r sdk.Registrar) {
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "gamebryo-incompatible-mod-archives", Name: "Gamebryo incompatible archive check", Trigger: "plugins-changed", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{ID: "oblivion-fonts", Name: "Oblivion font settings check", Trigger: "gamemode-activated", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "gamebryo-invalid-userlist",
		Name:    "Gamebryo invalid LOOT userlist check",
		Trigger: "gamemode-activated",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM surfaces Vortex's invalid-userlist test through profile-scoped LOOT userlist parsing in game diagnostics.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "gamebryo-missing-groups",
		Name:    "Gamebryo missing LOOT groups check",
		Trigger: "gamemode-activated",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM surfaces Vortex's missing-groups test through profile-scoped LOOT masterlist/userlist group validation in game diagnostics.",
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "gamebryo-plugin-index-lock", Name: "Gamebryo plugin index lock state", Scope: "profile_plugin_activations.locked_index", Status: sdk.CapabilityStatusReady})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "gamebryo-plugin-index-lock",
		Name:    "Gamebryo plugin index lock table attribute",
		Target:  "gamebryo-plugins",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes Vortex's lock-index state through profile plugin activation rows and applies it during generated load-order output; the exact Vortex table widget is represented by DMM's plugin activation update API/UI.",
	})
	r.RegisterExtensionActionCheck(sdk.ExtensionActionCheckSpec{
		ID:      "gamebryo-userlist-duplicate-rule-check",
		Name:    "Gamebryo duplicate userlist rule check",
		Target:  "loot-userlist",
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex blocks duplicate ADD_USERLIST_RULE actions before they reach state. DMM enforces the same source-backed rule in the LOOT userlist write path, rejecting duplicate after/require/incompatible rules before they can persist.",
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
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-transfer", Name: "Transfer save games", Scope: "profile-savegames", Kind: sdk.ExtensionActionKindAPI, Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-refresh", Name: "Refresh save games", Scope: "profile-savegames", Kind: sdk.ExtensionActionKindAPI, Status: sdk.CapabilityStatusReady, Message: savegameMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "gamebryo-save-open", Name: "Open save games", Scope: "profile-savegames", Kind: sdk.ExtensionActionKindPage, Status: sdk.CapabilityStatusReady, Message: savegameMessage})
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
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "dependency-manage-rules", Name: "Manage dependency rules", Scope: "profile-mod-rules", Kind: sdk.ExtensionActionKindPage, Status: sdk.CapabilityStatusReady, Message: dependencyRuleMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "dependency-connector", Name: "Dependency connector dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: dependencyDialogMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "dependency-editor", Name: "Dependency editor dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: dependencyDialogMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "conflict-editor", Name: "Conflict editor dialog", Scope: "dependencies", Status: sdk.CapabilityStatusReady, Message: conflictDialogMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{
		ID:      "dependency-cycle-graph",
		Name:    "Dependency cycle graph dialog",
		Scope:   "dependencies",
		Status:  sdk.CapabilityStatusReady,
		Message: "Verified non-applicable to DMM-created state: Vortex opens this graph only after cyclic mod rules have been persisted. DMM rejects cyclic before/after profile rules when they are saved, so the user resolves the invalid edit before it can become profile state. Imported Vortex environments may need a post-MVP repair wizard.",
	})
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

func registerImportAndMetadataSurfaces(r sdk.Registrar) {
	importMessage := "DMM represents Vortex's desktop NMM/MO import dialogs with source-aware archive import requests, per-game import actions, persisted Action Center state, and the phone/tablet advanced management flow. Steam Deck MVP keeps Vortex environment import behind explicit user-triggered import surfaces instead of auto-scanning external managers."
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "external-manager-import-session",
		Name:    "External manager import session",
		Scope:   "import",
		Status:  sdk.CapabilityStatusReady,
		Message: importMessage,
	})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{
		ID:      "nmm-import",
		Name:    "Nexus Mod Manager import dialog",
		Scope:   "import",
		Status:  sdk.CapabilityStatusReady,
		Message: importMessage,
	})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{
		ID:      "mo-import",
		Name:    "Mod Organizer import dialog",
		Scope:   "import",
		Status:  sdk.CapabilityStatusReady,
		Message: importMessage,
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "import-from-nmm",
		Name:    "Import from NMM",
		Scope:   "import",
		Kind:    sdk.ExtensionActionKindPage,
		Status:  sdk.CapabilityStatusReady,
		Message: importMessage,
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "import-from-mo",
		Name:    "Import from Mod Organizer",
		Scope:   "import",
		Kind:    sdk.ExtensionActionKindPage,
		Status:  sdk.CapabilityStatusReady,
		Message: importMessage,
	})

	metadataMessage := "DMM stores Nexus file metadata, source IDs, version strings, URLs, and compatibility attributes on managed mods and exposes them through the profile mod details API/UI. This is the DMM equivalent of Vortex's downloads metadata editor dialog."
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "managed-mod-metadata",
		Name:    "Managed mod metadata session",
		Scope:   "mod-metadata",
		Status:  sdk.CapabilityStatusReady,
		Message: metadataMessage,
	})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{
		ID:      "meta-editor-dialog",
		Name:    "Mod metadata editor dialog",
		Scope:   "mod-metadata",
		Status:  sdk.CapabilityStatusReady,
		Message: metadataMessage,
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "view-mod-metadata",
		Name:    "View mod metadata",
		Scope:   "mod-metadata",
		Kind:    sdk.ExtensionActionKindPage,
		Status:  sdk.CapabilityStatusReady,
		Message: metadataMessage,
	})
}

var steamAppDetailsURL = "https://store.steampowered.com/api/appdetails"

func registerSteamGameInfo(r sdk.Registrar) {
	r.RegisterGameInfoProvider(sdk.GameInfoProviderSpec{
		ID:           "steam",
		Name:         "Steam store game info provider",
		Tags:         []string{"release_date", "website", "metacritic_score"},
		CacheSeconds: int((7 * 24 * time.Hour).Seconds()),
		Priority:     50,
		Message:      "DMM mirrors Vortex's gameinfo-steam extension by querying Steam Store appdetails with the selected Steam app ID.",
		Provider:     steamGameInfoProvider,
	})
}

func steamGameInfoProvider(ctx context.Context, input sdk.GameInfoInput) (sdk.GameInfoResult, error) {
	appID := strings.TrimSpace(input.AppID)
	if appID == "" {
		return sdk.GameInfoResult{}, nil
	}
	requestURL, err := url.Parse(steamAppDetailsURL)
	if err != nil {
		return sdk.GameInfoResult{}, err
	}
	query := requestURL.Query()
	query.Set("appids", appID)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return sdk.GameInfoResult{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return sdk.GameInfoResult{}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return sdk.GameInfoResult{}, nil
	}
	var payload map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			ReleaseDate struct {
				Date string `json:"date"`
			} `json:"release_date"`
			Website    string `json:"website"`
			Metacritic struct {
				Score any `json:"score"`
			} `json:"metacritic"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return sdk.GameInfoResult{}, nil
	}
	entry, ok := payload[appID]
	if !ok || !entry.Success {
		return sdk.GameInfoResult{}, nil
	}
	details := []sdk.GameInfoDetail{}
	if releaseDate := strings.TrimSpace(entry.Data.ReleaseDate.Date); releaseDate != "" {
		details = append(details, sdk.GameInfoDetail{
			ID:     "release_date",
			Title:  "Release Date",
			Type:   "date",
			Value:  releaseDate,
			Source: "steam",
		})
	}
	if website := strings.TrimSpace(entry.Data.Website); website != "" {
		details = append(details, sdk.GameInfoDetail{
			ID:     "website",
			Title:  "Website",
			Type:   "url",
			Value:  website,
			Source: "steam",
		})
	}
	if score, ok := steamMetacriticScore(entry.Data.Metacritic.Score); ok {
		details = append(details, sdk.GameInfoDetail{
			ID:     "metacritic_score",
			Title:  "Score (Metacritic)",
			Value:  score,
			Source: "steam",
		})
	}
	return sdk.GameInfoResult{Details: details}, nil
}

func steamMetacriticScore(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		score := int(typed)
		if typed == float64(score) {
			return score, true
		}
	case string:
		score, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return score, true
		}
	}
	return 0, false
}

func readyAPI(id, name string) sdk.ExtensionAPISpec {
	return sdk.ExtensionAPISpec{ID: id, Name: name, Status: sdk.CapabilityStatusReady}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex FNIS integration source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/fnis-integration/src/index.ts"},
		{Name: "Vortex Gamebryo archive check source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-check/src/index.ts"},
		{Name: "Vortex Gamebryo plugin index lock source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-plugin-indexlock/src/index.tsx"},
		{Name: "Vortex Gamebryo savegame management source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-savegame-management/src/index.ts"},
		{Name: "Vortex Gamebryo test settings source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-test-settings/src/index.ts"},
		{Name: "Vortex Steam game info provider source", URL: "https://github.com/Nexus-Mods/Vortex/tree/2349a17900a37c2120e90733045dc6b303135b89/extensions/gameinfo-steam/src"},
		{Name: "Vortex local game settings source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/local-gamesettings/src/index.ts"},
		{Name: "Vortex mod dependency manager source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mod-dependency-manager/src/index.tsx"},
		{Name: "Vortex Morrowind plugin management source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/morrowind-plugin-management/src/index.ts"},
		{Name: "Vortex new-file monitor source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/new-file-monitor/src/index.ts"},
		{Name: "Vortex game-version test source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/test-gameversion/src/index.ts"},
		{Name: "Vortex setup test source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/test-setup/src/index.ts"},
		{Name: "Vortex script extender error check source", URL: "https://github.com/Nexus-Mods/Vortex/tree/2349a17900a37c2120e90733045dc6b303135b89/extensions/script-extender-error-check/src"},
	}
}
