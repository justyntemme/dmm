package extensions

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestFirstPartyCoversVortexRegistrationSurfaces(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex extension calls such as
	// context.registerGame, context.registerInstaller, context.registerAPI, etc.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	counts := map[string]int{}
	for _, summary := range summaries {
		caps := summary.Capabilities
		addFeatures(counts, "registerGame", featureSlice(caps.GameRegistration))
		addFeatures(counts, "registerInstaller", caps.Installers)
		addFeatures(counts, "registerModType", caps.ModTypes)
		addFeatures(counts, "registerAction", caps.ExtensionActions)
		addFeatures(counts, "registerTest", caps.ExtensionTests)
		addFeatures(counts, "registerReducer", caps.StateReducers)
		addFeatures(counts, "registerMigration", caps.StateMigrations)
		addFeatures(counts, "registerAPI", caps.ExtensionAPIs)
		addFeatures(counts, "registerGameStub", supportModFeature(summary.SupportModID))
		addFeatures(counts, "registerLoadOrder", caps.LoadOrders)
		addFeatures(counts, "registerDialog", caps.ExtensionDialogs)
		addFeatures(counts, "registerDashlet", caps.ExtensionDashlets)
		addFeatures(counts, "registerSettings", caps.ExtensionSettings)
		addFeatures(counts, "registerMerge", caps.Merges)
		addFeatures(counts, "registerMainPage", caps.ExtensionMainPages)
		addFeatures(counts, "registerInterpreter", caps.Interpreters)
		addFeatures(counts, "registerProfileFeature", caps.ProfileFeatures)
		addFeatures(counts, "registerGameStore", caps.GameStores)
		addFeatures(counts, "registerArchiveType", caps.ArchiveTypes)
		addFeatures(counts, "registerTableAttribute", caps.ExtensionTableAttrs)
		addFeatures(counts, "registerPersistor", caps.StatePersistors)
		addFeatures(counts, "registerLoadOrderPage", caps.ExtensionLoadOrderPages)
		addFeatures(counts, "registerProfileFile", caps.ProfileFiles)
		addFeatures(counts, "registerGameInfoProvider", caps.GameInfoProviders)
		addFeatures(counts, "registerAttributeExtractor", caps.AttributeExtractors)
		addFeatures(counts, "registerActionCheck", caps.ExtensionActionChecks)
		addFeatures(counts, "registerDynDiv", caps.ExtensionDynamicDividers)
		addFeatures(counts, "registerControlWrapper", caps.ExtensionControlWrappers)
		addFeatures(counts, "registerHistoryStack", caps.HistoryStacks)
		addFeatures(counts, "registerHealthCheck", caps.HealthChecks)
		addFeatures(counts, "registerStartHook", caps.StartHooks)
		addFeatures(counts, "registerEventHandler", caps.EventHandlers)
	}

	required := []string{
		"registerGame",
		"registerInstaller",
		"registerModType",
		"registerAction",
		"registerTest",
		"registerReducer",
		"registerMigration",
		"registerAPI",
		"registerGameStub",
		"registerLoadOrder",
		"registerDialog",
		"registerDashlet",
		"registerSettings",
		"registerMerge",
		"registerMainPage",
		"registerInterpreter",
		"registerProfileFeature",
		"registerGameStore",
		"registerArchiveType",
		"registerTableAttribute",
		"registerPersistor",
		"registerLoadOrderPage",
		"registerProfileFile",
		"registerGameInfoProvider",
		"registerAttributeExtractor",
		"registerActionCheck",
		"registerDynDiv",
		"registerControlWrapper",
		"registerHistoryStack",
		"registerHealthCheck",
		"registerStartHook",
		"registerEventHandler",
	}
	for _, surface := range required {
		if counts[surface] == 0 {
			t.Fatalf("missing DMM runtime capability for Vortex %s surface", surface)
		}
	}
	for _, summary := range summaries {
		if len(summary.Capabilities.ExtensionToDos) != 0 {
			t.Fatalf("%s advertises TODO surfaces instead of runtime capability: %+v", summary.ID, summary.Capabilities.ExtensionToDos)
		}
	}
}

func TestFirstPartyCoversVortexAPIMethodInventory(t *testing.T) {
	// Source inventory verified from actual api.* call sites in the checked-out
	// Nexus-Mods/Vortex extension tree. False positives from filenames/native
	// module objects are intentionally omitted; every listed method must map to
	// a ready DMM runtime surface.
	required := map[string][]string{
		"api.addMetaServer":         {"mod-meta-lookup-save"},
		"api.awaitUI":               {"ui-await"},
		"api.clearStylesheet":       {"ui-stylesheet"},
		"api.dismissNotification":   {"ui-notification"},
		"api.emitAndAwait":          {"deploy-mods", "purge-mods-in-path", "start-download", "start-install-download"},
		"api.ext.addToHistory":      {"addToHistory"},
		"api.ext.bepinexAddGame":    {"register-bepinex-unity-game"},
		"api.ext.ensureLoggedIn":    {"ensureLoggedIn"},
		"api.ext.nexusGetModFiles":  {"nexusGetModFiles"},
		"api.ext.showHistory":       {"showHistory"},
		"api.ext.ummAddGame":        {"ummAddGame"},
		"api.genMd5Hash":            {"native-system-access"},
		"api.getState":              {"state-store-access"},
		"api.lookupModMeta":         {"mod-meta-lookup-save"},
		"api.saveModMeta":           {"mod-meta-lookup-save"},
		"api.onAsync":               {"deploy-mods", "autosort-plugins"},
		"api.onStateChange":         {"state-store-access"},
		"api.runExecutable":         {"run-executable"},
		"api.saveFile":              {"ui-file-picker"},
		"api.selectDir":             {"ui-directory-picker"},
		"api.selectFile":            {"ui-file-picker"},
		"api.sendNotification":      {"ui-notification"},
		"api.setStylesheet":         {"ui-stylesheet"},
		"api.showDialog":            {"ui-dialog"},
		"api.showErrorNotification": {"ui-notification"},
		"api.suppressNotification":  {"ui-notification"},
		"api.translate":             {"ui-locale-highlight-outdated"},
		"api.GetDiskFreeSpaceEx":    {"native-system-access"},
		"api.GetProcessList":        {"native-system-access"},
		"api.GetVolumePathName":     {"native-system-access"},
		"api.RegEnumKeys":           {"native-system-access"},
		"api.RegEnumValues":         {"native-system-access"},
		"api.RegGetValue":           {"native-system-access"},
		"api.RegSetKeyValue":        {"native-system-access"},
		"api.WithRegOpen":           {"native-system-access"},
	}

	features := map[string]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		for _, feature := range allFeatureSummaries(summary.Capabilities) {
			features[feature.ID] = feature
		}
	}
	for method, ids := range required {
		found := false
		for _, id := range ids {
			feature, ok := features[id]
			if !ok || feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				continue
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("missing ready DMM runtime surface for Vortex %s; candidate ids: %v", method, ids)
		}
	}
}

func TestFirstPartyCoversVortexToDoSurfacesWithRuntimeActions(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerToDo calls:
	// fnis-integration, bsa-redirection, and import-nmm.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	features := map[string]gameext.FeatureSummary{}
	for _, summary := range summaries {
		for _, feature := range allFeatureSummaries(summary.Capabilities) {
			features[feature.ID] = feature
		}
	}
	for _, id := range []string{"fnis-integration", "oblivion-font-repair", "import-from-nmm"} {
		if _, ok := features[id]; !ok {
			t.Fatalf("missing runtime counterpart for Vortex registerToDo surface %q", id)
		}
	}
}

func TestFirstPartyCoversVortexLauncherRequirements(t *testing.T) {
	// Source inventory verified from live requiresLauncher registrations in
	// Nexus-Mods/Vortex extensions/games. Commented-out requiresLauncher
	// bindings, such as Morrowind and Prison Architect in the current upstream
	// tree, are intentionally omitted.
	required := []string{
		"7daystodie-steam-launcher",
		"bladeandsorcery-steam-launcher",
		"bloodstainedrotn-epic-launcher",
		"darksouls-steam-launcher",
		"darkestdungeon-epic-launcher",
		"dragonage-steam-launcher",
		"fallout3-epic-launcher",
		"fallout3-xbox-launcher",
		"fallout4-epic-launcher",
		"fallout4-xbox-launcher",
		"falloutnv-epic-launcher",
		"falloutnv-xbox-launcher",
		"halo-mcc-steam-launcher",
		"halo-mcc-xbox-launcher",
		"kenshi-steam-launcher",
		"kingdomcomedeliverance-epic-launcher",
		"kingdomcomedeliverance-xbox-launcher",
		"kotor2-steam-launcher",
		"msfs-xbox-launcher",
		"nomanssky-xbox-launcher",
		"poe2-xbox-launcher",
		"rimworld-steam-launcher",
		"skyrimse-epic-launcher",
		"skyrimse-xbox-launcher",
		"starbound-xbox-launcher",
		"torchlight2-steam-launcher",
		"totalwarthreekingdoms-epic-launcher",
		"untitledgoosegame-epic-launcher",
		"vampirebloodlines-steam-launcher",
	}

	features := map[string]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		for _, feature := range summary.Capabilities.LauncherRequirements {
			features[feature.ID] = feature
		}
	}
	for _, id := range required {
		feature, ok := features[id]
		if !ok {
			t.Fatalf("missing DMM launcher requirement for Vortex requiresLauncher surface %q", id)
		}
		if feature.Status != "" && feature.Status != sdk.CapabilityStatusReady {
			t.Fatalf("launcher requirement %s is not ready: %+v", id, feature)
		}
		if strings.TrimSpace(feature.Message) == "" {
			t.Fatalf("launcher requirement %s must document its Vortex-backed behavior", id)
		}
	}
}

func TestFirstPartyCoversVortexDashletSurfaces(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerDashlet calls:
	// mtframework-arc-support, modtype-umm, extension-dashlet,
	// changelog-dashlet, issue-tracker, modtype-bepinex, quickbms-support.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	features := map[string]gameext.FeatureSummary{}
	for _, summary := range summaries {
		for _, feature := range summary.Capabilities.ExtensionDashlets {
			features[feature.ID] = feature
		}
	}
	required := []string{
		"mtframework-arc-support",
		"extension-dashlet",
		"changelog-dashlet",
		"issue-tracker",
		"bepinex-support",
		"quickbms-support",
	}
	for _, id := range required {
		feature, ok := features[id]
		if !ok {
			t.Fatalf("missing runtime counterpart for Vortex registerDashlet surface %q", id)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("dashlet %s = %+v", id, feature)
		}
	}

	foundUMM := false
	for id, feature := range features {
		if strings.HasSuffix(id, "-umm-support-dashlet") && feature.Status == sdk.CapabilityStatusReady && feature.Message != "" {
			foundUMM = true
			break
		}
	}
	if !foundUMM {
		t.Fatalf("missing runtime counterpart for Vortex modtype-umm registerDashlet surface")
	}
}

func TestFirstPartyCoversVortexSettingsSurfaces(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerSettings calls:
	// gamebryo-plugin-management Workarounds, gamebryo-archive-invalidation
	// Workarounds, theme-switcher Theme, fnis-integration Interface, and
	// Stardew Valley Mods settings.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	features := map[string]gameext.FeatureSummary{}
	for _, summary := range summaries {
		for _, feature := range summary.Capabilities.ExtensionSettings {
			features[feature.ID] = feature
		}
	}
	for _, id := range []string{
		"dependency-workarounds",
		"gamebryo-archive-invalidation-workarounds",
		"interface-theme",
		"stardew_merge_configs",
	} {
		feature, ok := features[id]
		if !ok {
			t.Fatalf("missing runtime counterpart for Vortex registerSettings surface %q", id)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("setting %s = %+v", id, feature)
		}
	}

	foundFNIS := false
	for _, feature := range features {
		if feature.ID == "fnis_auto_run" && feature.Status == sdk.CapabilityStatusReady && feature.Message != "" {
			foundFNIS = true
			break
		}
	}
	if !foundFNIS {
		t.Fatalf("missing runtime counterpart for Vortex fnis-integration registerSettings surface")
	}
}

func TestFirstPartyCoversVortexActionAndPageSurfaces(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerAction,
	// registerMainPage, and registerLoadOrderPage calls. Repeated open-folder
	// actions are represented by DMM's generic open-directory API plus
	// extension-declared safe path targets.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	features := map[string]gameext.FeatureSummary{}
	for _, summary := range summaries {
		for _, feature := range allFeatureSummaries(summary.Capabilities) {
			features[feature.ID] = feature
		}
	}
	for _, id := range []string{
		"gamebryo-save-transfer",
		"gamebryo-save-refresh",
		"gamebryo-save-open",
		"gamebryo-savegames",
		"morrowind-plugins",
		"gamebryo-plugin-manage-rules",
		"gamebryo-plugin-manage-groups",
		"gamebryo-plugin-history",
		"gamebryo-plugin-reset-rules",
		"import-from-nmm",
		"import-from-mo",
		"mod-report",
		"fnis-configure-patches",
		"stardew-smapi-log",
		"stardew_merge_configs",
		"view-mod-metadata",
		"open-directory-action",
		"witcher3-install-script-merger",
		"witcher3-open-documents",
		"witcher3-load-order-page",
		"bladeandsorcery-loadorder-page",
		"conanexiles-load-order-page",
		"kingdomcomedeliverance-load-order-page",
		"poe2-load-order-page",
		"msfs-load-order-page",
		"xcom2-load-order-page",
	} {
		feature, ok := features[id]
		if !ok {
			t.Fatalf("missing runtime counterpart for Vortex action/page surface %q", id)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("action/page %s = %+v", id, feature)
		}
	}
}

func TestFirstPartyCoversVortexMergeAndLoadOrderInventory(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerMerge and
	// registerLoadOrder calls. DMM may expose richer load-order support for
	// additional games, but each Vortex-backed merge/load-order registration
	// must have a concrete extension-owned runtime surface.
	requiredMerges := []string{
		"wolcen-xml-mtl-merge",
		"msfs-aircraft-cfg",
		"dragonage-addins-xml",
		"witcher3-xml-menu-merge",
	}
	requiredLoadOrders := []string{
		"codevein-pak-load-order",
		"xcom2-default-mod-options",
		"spyro-pak-load-order",
		"bloodstainedrotn-pak-load-order",
		"morrowind-ini-load-order",
		"witcher3-mods-settings",
		"conanexiles-modlist",
		"msfs-community-load-order",
		"bladeandsorcery-loadorder-json",
	}

	merges := map[string]gameext.FeatureSummary{}
	loadOrders := map[string]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		for _, merge := range summary.Capabilities.Merges {
			merges[merge.ID] = merge
		}
		for _, loadOrder := range summary.Capabilities.LoadOrders {
			loadOrders[loadOrder.ID] = loadOrder
		}
	}
	for _, id := range requiredMerges {
		feature, ok := merges[id]
		if !ok {
			t.Fatalf("missing runtime counterpart for Vortex registerMerge surface %q", id)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("merge %s = %+v", id, feature)
		}
	}
	for _, id := range requiredLoadOrders {
		feature, ok := loadOrders[id]
		if !ok {
			t.Fatalf("missing runtime counterpart for Vortex registerLoadOrder surface %q", id)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
			t.Fatalf("load order %s = %+v", id, feature)
		}
	}
}

func TestFirstPartyCoversVortexRuntimeSupportInventory(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex calls for runtime
	// support surfaces not already covered by installer/load-order tests:
	// common-interpreters, gameinfo-steam, test-gameversion,
	// gamebryo-plugin-management, game-xrebirth, game-pillarsofeternity2,
	// and game-stardewvalley.
	required := map[string][]string{
		"interpreter": {
			"jar",
			"python",
			"vbs",
			"cmd",
			"bat",
		},
		"game info provider": {
			"game-version",
			"steam",
		},
		"game version provider": {
			"hash-version-check",
		},
		"profile file": {
			"plugins-file",
			"loadorder-file",
		},
		"state persistor": {
			"gamebryo-plugins-load-order-persistor",
			"gamebryo-plugins-userlist-persistor",
			"gamebryo-plugins-masterlist-persistor",
		},
		"action check": {
			"gamebryo-management-enabled-sync",
			"gamebryo-userlist-duplicate-rule-check",
		},
		"history stack": {
			"plugins",
		},
		"health check": {
			"xrebirth-content-xml-metadata",
			"xrebirth-mod-has-files",
			"xrebirth-mod-shape-recognised",
		},
		"attribute extractor": {
			"poe2-manifest-version",
			"stardew-manifest-attribute-extractor",
		},
	}

	features := map[string]map[string]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		addFeatureMap(features, "interpreter", summary.Capabilities.Interpreters)
		addFeatureMap(features, "game info provider", summary.Capabilities.GameInfoProviders)
		addFeatureMap(features, "game version provider", summary.Capabilities.GameVersions)
		addFeatureMap(features, "profile file", summary.Capabilities.ProfileFiles)
		addFeatureMap(features, "state persistor", summary.Capabilities.StatePersistors)
		addFeatureMap(features, "action check", summary.Capabilities.ExtensionActionChecks)
		addFeatureMap(features, "history stack", summary.Capabilities.HistoryStacks)
		addFeatureMap(features, "health check", summary.Capabilities.HealthChecks)
		addFeatureMap(features, "attribute extractor", summary.Capabilities.AttributeExtractors)
	}

	for kind, ids := range required {
		for _, id := range ids {
			feature, ok := features[kind][id]
			if !ok {
				if kind == "profile file" || kind == "state persistor" {
					if feature, ok = findFeatureWithSuffix(features[kind], "-"+id); ok {
						goto checkFeature
					}
				}
				t.Fatalf("missing runtime counterpart for Vortex %s surface %q", kind, id)
			}
		checkFeature:
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("%s %s = %+v", kind, id, feature)
			}
		}
	}
}

func TestFirstPartyCoversVortexLifecycleEventInventory(t *testing.T) {
	// Source inventory verified from Vortex context.once handlers using
	// context.api.events.on and context.api.onAsync. These hooks carry runtime
	// behavior that is not visible from register* calls, so DMM must expose the
	// equivalent lifecycle event in the game extension itself.
	required := map[string][]string{
		"battletech": {
			sdk.EventAddedFiles,
		},
		"bladeandsorcery": {
			sdk.EventWillDeploy,
			sdk.EventDidDeploy,
		},
		"divinityoriginalsin2": {
			sdk.EventWillDeploy,
			sdk.EventDidDeploy,
		},
		"fallout4vr": {
			sdk.EventDidDeploy,
		},
		"galacticcivilizations3": {
			sdk.EventWillDeploy,
			sdk.EventDidDeploy,
		},
		"greedfall": {
			sdk.EventDidDeploy,
		},
		"kingdomcomedeliverance": {
			sdk.EventModEnabled,
			sdk.EventDidPurge,
			sdk.EventDidDeploy,
		},
		"halothemasterchiefcollection": {
			sdk.EventDidDeploy,
			sdk.EventDidPurge,
		},
		"morrowind": {
			sdk.EventDidInstallMod,
		},
		"pillarsofeternity2": {
			sdk.EventGamemodeActivated,
		},
		"skyrimvr": {
			sdk.EventGamemodeActivated,
			sdk.EventDidDeploy,
		},
		"stardewvalley": {
			sdk.EventAddedFiles,
			sdk.EventWillEnableMods,
			sdk.EventDidDeploy,
			sdk.EventDidPurge,
			sdk.EventDidInstallMod,
			sdk.EventGamemodeActivated,
		},
		"witcher3": {
			sdk.EventGamemodeActivated,
			sdk.EventProfileWillChange,
			sdk.EventModsEnabled,
			sdk.EventWillDeploy,
			sdk.EventDidDeploy,
			sdk.EventDidPurge,
			sdk.EventDidRemoveMod,
		},
	}

	byID := map[string]gameext.ExtensionSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		byID[strings.ToLower(summary.ID)] = summary
	}
	for extensionID, events := range required {
		summary, ok := byID[extensionID]
		if !ok {
			t.Fatalf("missing first-party extension %q for Vortex lifecycle inventory", extensionID)
		}
		features := map[string]gameext.FeatureSummary{}
		for _, feature := range summary.Capabilities.EventHandlers {
			features[strings.ToLower(feature.ID)] = feature
		}
		for _, event := range events {
			feature, ok := features[strings.ToLower(event)]
			if !ok {
				t.Fatalf("%s missing runtime counterpart for Vortex lifecycle event %q", extensionID, event)
			}
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("%s lifecycle event %s = %+v", extensionID, event, feature)
			}
		}
	}
}

func TestFirstPartyCoversVortexStateChangeInventory(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex context.api.onStateChange calls
	// in game-witcher3, game-baldursgate3, gamebryo-plugin-indexlock,
	// gamebryo-plugin-management, and gamebryo-savegame-management.
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	watchersByID := map[string]gameext.FeatureSummary{}
	for _, summary := range summaries {
		for _, watcher := range summary.Capabilities.StateChangeWatchers {
			watchersByID[watcher.ID] = watcher
		}
	}
	for _, required := range []struct {
		id   string
		path string
	}{
		{"witcher3-settings-change", "settings.witcher3"},
		{"bg3-tools-running-load-order-refresh", "session.base.toolsRunning"},
		{"gamebryo-index-lock-load-order", "loadOrder"},
		{"gamebryo-index-lock-plugin-info", "session.plugins.pluginInfo"},
		{"gamebryo-index-lock-persistent-indices", "persistent.plugins.lockedIndices"},
		{"gamebryo-plugin-management-load-order", "loadOrder"},
		{"gamebryo-plugin-management-discovery", "settings.gameMode.discovered"},
		{"gamebryo-plugin-management-main-page", "session.base.mainPage"},
		{"gamebryo-plugin-management-profiles", "persistent.profiles"},
		{"gamebryo-savegame-profile-feature", "persistent.profiles"},
		{"gamebryo-savegame-discovery", "settings.gameMode.discovered"},
	} {
		watcher, ok := watchersByID[required.id]
		if !ok {
			t.Fatalf("missing DMM state-change watcher for Vortex onStateChange surface %q", required.id)
		}
		if watcher.Trigger != required.path || watcher.Status != sdk.CapabilityStatusReady || watcher.Message == "" {
			t.Fatalf("state-change watcher %s = %+v", required.id, watcher)
		}
	}
}

func TestFirstPartyCoversVortexSupportLifecycleEventInventory(t *testing.T) {
	// Source inventory verified from Vortex framework/support extensions outside
	// extensions/games. These are shared extension-runtime hooks that game
	// extensions rely on for BepInEx, UMM, and QuickBMS behavior.
	required := map[string][]string{
		"modtype-bepinex": {
			sdk.EventDidInstallMod,
			sdk.EventProfileWillChange,
			sdk.EventGamemodeActivated,
			sdk.EventWillDeploy,
			sdk.EventCheckModsVersion,
		},
		"quickbms-support": {
			sdk.EventGamemodeActivated,
			"quickbms-operation",
		},
	}

	byID := map[string]gameext.ExtensionSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		byID[strings.ToLower(summary.ID)] = summary
	}
	for extensionID, events := range required {
		summary, ok := byID[extensionID]
		if !ok {
			t.Fatalf("missing first-party support extension %q for Vortex lifecycle inventory", extensionID)
		}
		features := map[string]gameext.FeatureSummary{}
		for _, feature := range summary.Capabilities.EventHandlers {
			features[strings.ToLower(feature.ID)] = feature
		}
		for _, event := range events {
			feature, ok := features[strings.ToLower(event)]
			if !ok {
				t.Fatalf("%s missing runtime counterpart for Vortex support lifecycle event %q", extensionID, event)
			}
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("%s support lifecycle event %s = %+v", extensionID, event, feature)
			}
		}
	}

	ummEventCoverage := map[string]map[string]bool{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		for _, feature := range summary.Capabilities.ExtensionAPIs {
			if feature.ID != "ummAddGame" {
				continue
			}
			events := map[string]bool{}
			for _, handler := range summary.Capabilities.EventHandlers {
				events[strings.ToLower(handler.ID)] = true
			}
			ummEventCoverage[summary.ID] = events
		}
	}
	if len(ummEventCoverage) == 0 {
		t.Fatalf("missing first-party UMM support extension registrations")
	}
	for extensionID, events := range ummEventCoverage {
		for _, event := range []string{sdk.EventGamemodeActivated, sdk.EventCheckModsVersion} {
			if !events[strings.ToLower(event)] {
				t.Fatalf("%s missing runtime counterpart for Vortex UMM lifecycle event %q", extensionID, event)
			}
		}
	}
}

func TestFirstPartyCoversVortexStateProfileAndStoreInventory(t *testing.T) {
	// Source inventory verified from Vortex registerGameStore,
	// registerProfileFeature, optional.registerCollectionFeature,
	// registerReducer, and registerMigration calls. Per-game migration IDs may
	// be DMM-owned, but each upstream migration surface must have a concrete
	// state migration with executable commands in that extension.
	required := map[string][]string{
		"game store": {
			"gog",
			"origin",
			"uplay",
			"xbox",
		},
		"profile feature": {
			"local_game_settings",
			"gamebryo-savegames",
			"local_loot_rules",
			"local_merges",
		},
		"collection feature": {
			"morrowind-collection-data",
			"witcher3-collection-data",
			"kingdomcomedeliverance-collection-data",
		},
		"state reducer": {
			"test-gameversion-state",
			"external-manager-import-session",
			"issues-persistent",
			"issues-session",
			"changelog-cache",
			"gamebryo-save-session",
			"gamebryo-save-settings",
			"gamebryo-plugin-index-lock",
			"dependency-workarounds",
			"dependency-session",
			"fnis-settings",
			"interface-theme-settings",
			"managed-mod-metadata",
			"stardew-settings-reducer",
			"witcher3-settings-reducer",
		},
	}

	features := map[string]map[string]gameext.FeatureSummary{}
	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	for _, summary := range summaries {
		addFeatureMap(features, "game store", summary.Capabilities.GameStores)
		addFeatureMap(features, "profile feature", summary.Capabilities.ProfileFeatures)
		addFeatureMap(features, "collection feature", summary.Capabilities.CollectionFeatures)
		addFeatureMap(features, "state reducer", summary.Capabilities.StateReducers)
	}
	for kind, ids := range required {
		for _, id := range ids {
			feature, ok := features[kind][id]
			if !ok {
				t.Fatalf("missing runtime counterpart for Vortex %s surface %q", kind, id)
			}
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("%s %s = %+v", kind, id, feature)
			}
		}
	}

	requiredMigrationExtensions := map[string]int{
		"dragonage2":                   1,
		"dragonsdogma":                 1,
		"codevein":                     1,
		"nomanssky":                    1,
		"bloodstainedritualofthenight": 1,
		"spyroreignitedtrilogy":        1,
		"untitledgoosegame":            1,
		"witcher3":                     1,
		"morrowind":                    1,
		"thesims4":                     1,
		"x4foundations":                1,
		"bladeandsorcery":              2,
	}
	byExtension := map[string]int{}
	for _, summary := range summaries {
		for _, migration := range summary.Capabilities.StateMigrations {
			if migration.Status != sdk.CapabilityStatusReady || migration.Message == "" || len(migration.Commands) == 0 {
				t.Fatalf("state migration %s = %+v", migration.ID, migration)
			}
			byExtension[strings.ToLower(summary.ID)]++
		}
	}
	for extensionID, count := range requiredMigrationExtensions {
		if byExtension[extensionID] < count {
			t.Fatalf("missing runtime counterpart for Vortex migration extension %q: got %d want >= %d", extensionID, byExtension[extensionID], count)
		}
	}
}

func TestFirstPartyCoversVortexStoreAppIDs(t *testing.T) {
	// Source inventory verified from Vortex game extension findGame/details
	// literals and gamestore registrations. These are extension-owned because
	// they decide which store manifests identify each game.
	required := map[string]map[string][]string{
		"baldursgate3":                  {"gog": {"1456460669"}},
		"bloodstainedritualofthenight":  {"epic": {"a2ac59c83b704e40b4ab3a9e963fef52"}},
		"darkestdungeon":                {"gog": {"1450711444"}, "epic": {"36cbf259e631478eaac6ea244e55a709"}},
		"fallout3":                      {"gog": {"1454315831"}, "epic": {"adeae8bbfc94427db57c7dfecce3f1d4"}, "xbox": {"BethesdaSoftworks.Fallout3"}},
		"fallout4":                      {"gog": {"1998527297"}, "epic": {"61d52ce4d09d41e48800c22784d13ae8"}, "xbox": {"BethesdaSoftworks.Fallout4-PC"}},
		"falloutnv":                     {"gog": {"1454587428"}, "epic": {"5daeb974a22a435988892319b3a4f476"}, "xbox": {"BethesdaSoftworks.FalloutNewVegas"}},
		"kingdomcomedeliverance":        {"epic": {"Eel"}, "xbox": {"DeepSilver.KingdomComeDeliverance"}},
		"morrowind":                     {"gog": {"1435828767"}, "xbox": {"BethesdaSoftworks.TESMorrowind-PC"}},
		"nomanssky":                     {"xbox": {"HelloGames.NoMansSky"}},
		"oblivion":                      {"gog": {"1458058109"}, "xbox": {"BethesdaSoftworks.TESOblivion-PC"}},
		"pathfinderwrathoftherighteous": {"gog": {"1207187357"}},
		"pillarsofeternity2":            {"xbox": {"VersusEvil.PillarsofEternity2-PC"}},
		"skyrimse":                      {"gog": {"1711230643"}, "epic": {"ac82db5035584c7f8a2c548d98c86b2c"}, "xbox": {"BethesdaSoftworks.SkyrimSE-PC"}},
		"starbound":                     {"xbox": {"Chucklefish.StarboundWindows10Edition"}},
		"torchlight2":                   {"gog": {"1958228073"}},
		"totalwarthreekingdoms":         {"gog": {"1717887914"}, "epic": {"769f2fee68e9477180da900ccccbbcf0"}},
		"vampirebloodlines":             {"gog": {"1207659240"}},
		"witcher3":                      {"gog": {"1495134320", "1207664663", "1207664643", "1640424747"}, "epic": {"725a22e15ed74735bb0d6a19f3cc82d0"}},
		"x4foundations":                 {"gog": {"1395669635"}},
		"xcom2":                         {"gog": {"1482002159"}, "epic": {"3be3c4d681bc46b3b8b26c5df3ae0a18"}},
	}

	summaries := map[string]gameext.ExtensionSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		summaries[summary.ID] = summary
	}
	for extensionID, stores := range required {
		summary, ok := summaries[extensionID]
		if !ok || summary.Capabilities.GameRegistration == nil {
			t.Fatalf("missing game registration for Vortex store app ID parity target %s", extensionID)
		}
		for store, appIDs := range stores {
			got := summary.Capabilities.GameRegistration.StoreAppIDs[store]
			for _, appID := range appIDs {
				if !containsString(got, appID) {
					t.Fatalf("%s missing %s store app ID %q from Vortex source; got %+v", extensionID, store, appID, got)
				}
			}
		}
	}
}

func TestFirstPartyCoversVortexSetupRegistrations(t *testing.T) {
	// Source inventory verified from active registerGame({ setup: ... }) calls
	// in Nexus-Mods/Vortex extensions/games. Every listed game must expose a
	// DMM game setup surface, either directly or through a shared extension
	// helper such as UMM or Gamebryo.
	required := []string{
		"7daystodie",
		"ahatintime",
		"baldursgate3",
		"battletech",
		"bladeandsorcery",
		"bloodstainedritualofthenight",
		"codevein",
		"darkestdungeon",
		"dawnofman",
		"divinityoriginalsin2",
		"dragonage",
		"dragonage2",
		"dragons-dogma",
		"elex",
		"factorio",
		"fallout3",
		"fallout4vr",
		"gardenpaws",
		"greedfall",
		"grimdawn",
		"grimrock",
		"kenshi",
		"kingdomcome-deliverance",
		"monster-hunter-world",
		"neverwinter-nights",
		"neverwinter-nights2",
		"nomanssky",
		"oblivion",
		"oni",
		"pathfinderkingmaker",
		"pillarsofeternity2",
		"sekiro",
		"shadowrunreturns",
		"sims3",
		"sims4",
		"skyrimvr",
		"spyroreignitedtrilogy",
		"starbound",
		"stardewvalley",
		"survivingmars",
		"sw-kotor",
		"torchlight2",
		"totalwarthreekingdoms",
		"untitledgoose",
		"vtmbloodlines",
		"warthunder",
		"witcher",
		"witcher2",
		"witcher3",
		"wolcen",
		"x4foundations",
		"xcom2",
	}
	aliases := map[string][]string{
		"7daystodie":              {"sevendaystodie"},
		"dragons-dogma":           {"dragonsdogma"},
		"kingdomcome-deliverance": {"kingdomcomedeliverance"},
		"monster-hunter-world":    {"monsterhunterworld"},
		"neverwinter-nights":      {"neverwinter"},
		"neverwinter-nights2":     {"neverwinter"},
		"oni":                     {"oxygennotincluded"},
		"sims3":                   {"thesims3"},
		"sims4":                   {"thesims4"},
		"sw-kotor":                {"kotor", "kotor2"},
		"untitledgoose":           {"untitledgoosegame"},
		"vtmbloodlines":           {"vampirebloodlines"},
		"witcher":                 {"witcherlegacy"},
		"witcher2":                {"witcherlegacy"},
		"wolcen":                  {"wolcenlordsofmayhem"},
	}

	setupByID := map[string][]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		if len(summary.Capabilities.GameSetups) == 0 {
			continue
		}
		keys := []string{summary.ID, summary.VortexGameID}
		keys = append(keys, summary.NexusDomains...)
		for _, key := range keys {
			if key == "" {
				continue
			}
			setupByID[key] = append(setupByID[key], summary.Capabilities.GameSetups...)
		}
	}
	for _, id := range required {
		setups := setupByID[id]
		for _, alias := range aliases[id] {
			setups = append(setups, setupByID[alias]...)
		}
		if len(setups) == 0 {
			t.Fatalf("missing DMM setup surface for Vortex game setup %q", id)
		}
		for _, setup := range setups {
			if setup.Status != sdk.CapabilityStatusReady || len(setup.SetupActions) == 0 {
				t.Fatalf("setup surface for %s is not ready: %+v", id, setup)
			}
		}
	}
}

func TestFirstPartyCoversVortexAPITestDialogAndTableInventory(t *testing.T) {
	// Source inventory verified from Vortex registerAPI, registerTest,
	// registerDialog, and registerTableAttribute calls. Repeated trigger
	// registrations map to one DMM extension test when the test body is the
	// same runtime check exposed through DMM's event/test runner.
	required := map[string][]string{
		"extension API": {
			"create-mod",
			"deploy-mods",
			"register-bepinex-unity-game",
			"ummAddGame",
			"getHashVersion",
			"lootSortAsync",
			"isBlueprintPlugin",
			"qbmsRegisterGame",
			"qbmsList",
			"qbmsExtract",
			"qbmsWrite",
			"qbmsReimport",
			"purge-mods",
			"remove-mod",
			"start-download",
			"start-install-download",
			"install-extension",
			"start-quick-discovery",
			"enable-download-watch",
			"did-import-downloads",
			"preview-files",
			"analytics-track-click-event",
			"deploy-mods-event",
			"purge-mods-event",
			"download-script-extender",
			"will-remove-mods",
			"will-purge",
			"did-install-collection",
			"collection-postprocess-complete",
			"did-update-masterlist",
			"navigate-knowledgebase",
			"open-knowledge-base",
			"mod-content-changed",
			"request-own-issues",
			"display-report",
			"report-feedback",
			"report-log-error",
			"submit-feedback",
			"select-theme",
			"apply-settings",
			"update-conflicts-and-rules",
			"install-dependencies",
			"will-install-dependencies",
			"check-file-override-redundancies",
			"edit-mod-cycle",
			"recalculate-modtype-conflicts",
			"ui-file-picker",
			"run-executable",
		},
		"extension test": {
			"gamebryo-incompatible-mod-archives",
			"gamebryo-plugins-locked",
			"gamebryo-missing-masters",
			"gamebryo-blueprint-master",
			"dependency-unsolved-conflicts",
			"gamebryo-invalid-userlist",
			"gamebryo-missing-groups",
			"gamebryo-exceeded-plugin-limit",
			"oblivion-fonts",
			"skyrim-fonts",
			"bepinex-config-test",
			"doorstop-config-test",
			"script-extender-missing",
			"misconfigured-script-extender",
			"fnis-integration",
			"local-game-settings-global-files",
			"game-version-gamemode",
			"game-version-mod-installed",
			"test-setup-uninstall-entry",
			"sdv-incompatible-mods",
			"mcc-ce-mp-test",
		},
		"extension dialog": {
			"meta-editor-dialog",
			"dependency-connector",
			"dependency-editor",
			"dependency-group-editor",
			"nmm-import",
			"mo-import",
			"feedback-responder",
		},
		"table attribute": {
			"script-extender-errors",
			"sdv-compatibility",
			"gameType",
			"gamebryo-plugin-index-lock",
		},
	}

	features := map[string]map[string]gameext.FeatureSummary{}
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		addFeatureMap(features, "extension API", summary.Capabilities.ExtensionAPIs)
		addFeatureMap(features, "extension test", summary.Capabilities.ExtensionTests)
		addFeatureMap(features, "extension dialog", summary.Capabilities.ExtensionDialogs)
		addFeatureMap(features, "table attribute", summary.Capabilities.ExtensionTableAttrs)
	}
	for kind, ids := range required {
		for _, id := range ids {
			feature, ok := features[kind][id]
			if !ok {
				t.Fatalf("missing runtime counterpart for Vortex %s surface %q", kind, id)
			}
			if feature.Status != sdk.CapabilityStatusReady || feature.Message == "" {
				t.Fatalf("%s %s = %+v", kind, id, feature)
			}
		}
	}
}

func TestFirstPartyCoversVortexInstallerIDInventory(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerInstaller calls.
	// DMM expands shared Vortex installers per game, so this test checks that
	// every Vortex installer ID has at least one DMM installer with the same
	// VortexInstallerID, plus explicit mappings for upstream constant IDs.
	required := []string{
		"ahatintime-mod",
		"bas-mulledk19-mod",
		"bas-official-mod",
		"bepinex-root",
		"bepis-injector-extensible",
		"bloodstainedrotn-mod",
		"codevein-mod",
		"dazipInner",
		"dazipOuter",
		"dd-noproject-mod",
		"dd-project-mod",
		"dddainvalidmod",
		"dfmodmultiplatform",
		"dinput",
		"dom-mod",
		"dom-scene-installer",
		"elex-mod",
		"fallout4vr-esl-enabler",
		"falloutnv-4gb-patch",
		"galciv3installer",
		"gedosato",
		"greedfall-mod",
		"kenshi-mod",
		"kotor-override-mod",
		"kotor-root-mod",
		"kotor-tslpatcher",
		"kotor-tslpatcher-mod",
		"masterchiefinstaller",
		"masterchiefmodconfiginstaller",
		"mcc-plug-and-play-installer",
		"mhwreshadeinstaller",
		"moduleinstaller",
		"monster-hunter-mod",
		"mount-and-blade-mod",
		"msfs-pack",
		"msfs-replacer",
		"nwn-mod",
		"rimworld-steam-mod",
		"script-extender-installer",
		"scriptmergerdummy",
		"sek-loose-files",
		"sek-root-mod",
		"sims4mixed",
		"skyvr-esl-enabler",
		"smapi-installer",
		"sdvrootfolder",
		"stardew-valley-installer",
		"spyroreignitedtrilogy-mod",
		"survivingmars-mod",
		"teamfortress2-mod",
		"torchlight2-mod",
		"tw3kingdoms-mod",
		"umm-installer",
		"witcher3content",
		"witcher3dlcmod",
		"witcher3menumodroot",
		"witcher3mixed",
		"witcher3tl",
		"xcom2-installer",
		"xrebirth",
	}
	covered := map[string]gameext.FeatureSummary{}
	for _, extension := range FirstParty() {
		for _, installer := range extension.InstallPlan.Installers {
			if installer.VortexInstallerID != "" {
				covered[installer.VortexInstallerID] = gameext.FeatureSummary{
					ID:      installer.ID,
					Status:  installer.Status,
					Message: installer.Message,
				}
			}
		}
	}
	for _, id := range required {
		if _, ok := covered[id]; !ok {
			t.Fatalf("missing DMM installer coverage for Vortex registerInstaller ID %q", id)
		}
	}
	if _, ok := covered["enb"]; ok {
		t.Fatalf("DMM must not implement Vortex enb installer; upstream registerInstaller is commented out")
	}
}

func TestFirstPartyCoversLiteralVortexInstallerIDs(t *testing.T) {
	upstreamIDs := literalVortexInstallerIDs(t, "/tmp/dmm-vortex-upstream/extensions")
	if len(upstreamIDs) == 0 {
		t.Skip("local Vortex source checkout is unavailable")
	}
	covered := map[string]bool{}
	for _, extension := range FirstParty() {
		for _, installer := range extension.InstallPlan.Installers {
			if strings.TrimSpace(installer.VortexInstallerID) != "" {
				covered[installer.VortexInstallerID] = true
			}
		}
	}
	for _, id := range upstreamIDs {
		if id == "enb" {
			continue
		}
		if !covered[id] {
			t.Fatalf("missing DMM installer coverage for literal Vortex registerInstaller ID %q", id)
		}
	}
}

func literalVortexInstallerIDs(t *testing.T, root string) []string {
	t.Helper()
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	rgx := regexp.MustCompile(`registerInstaller\(\s*['"]([^'"]+)['"]`)
	seen := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".js", ".ts", ".tsx":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range rgx.FindAllStringSubmatch(string(data), -1) {
			seen[match[1]] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func TestFirstPartyCoversVortexModTypeAndArchiveInventory(t *testing.T) {
	// Source inventory verified from Nexus-Mods/Vortex registerModType and
	// registerArchiveType calls. Some shared Vortex mod types are expanded into
	// game-scoped DMM mod types; those are asserted through representative IDs.
	requiredModTypes := []string{
		"bas-legacy-modtype",
		"bas-official-modtype",
		"bepinex-patcher",
		"dazip",
		"dinput",
		"dom-scene-modtype",
		"enb",
		"galciv3crusade",
		"gedosato",
		"kotor-root",
		"mhwreshade",
		"mhwstrackermodloader",
		"msfs-pack",
		"nwn2-override-mod",
		"sims4mixed",
		"umm",
		"vtmb-up-modtype",
		"w3modlimitpatcher",
		"warthunder-audio-modtype",
		"witcher2user",
		"witcher3dlc",
		"witcher3menumodroot",
		"witcher3tl",
		"witcheruser",
		"x4-documents-modtype",
	}
	requiredArchives := []string{"arc", "ba2", "bsa"}

	modTypes := map[string]bool{}
	archiveTypes := map[string]bool{}
	for _, extension := range FirstParty() {
		for _, modType := range extension.InstallPlan.ModTypes {
			modTypes[modType.ID] = true
		}
		for _, archiveType := range extension.ArchiveTypes {
			archiveTypes[archiveType.ID] = true
		}
	}
	for _, id := range requiredModTypes {
		if !modTypes[id] {
			t.Fatalf("missing DMM mod type coverage for Vortex registerModType ID %q", id)
		}
	}
	if !hasSuffixModType(modTypes, "-bepinex-injector") {
		t.Fatalf("missing DMM runtime mod type coverage for Vortex BepInEx injector mod type")
	}
	if !hasSuffixModType(modTypes, "-bepinex-root") {
		t.Fatalf("missing DMM runtime mod type coverage for Vortex BepInEx root mod type")
	}
	if !hasSuffixModType(modTypes, "-bepinex-plugin") {
		t.Fatalf("missing DMM runtime mod type coverage for Vortex BepInEx plugin mod type")
	}
	for _, id := range requiredArchives {
		if !archiveTypes[id] {
			t.Fatalf("missing DMM archive type coverage for Vortex registerArchiveType ID %q", id)
		}
	}
}

func TestFirstPartyExtensionsAdvertiseNoUnresolvedParitySurfaces(t *testing.T) {
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
		if summary.VortexGameID != "" && summary.Coverage == gameext.CoverageMetadataOnly {
			t.Fatalf("%s remains metadata-only; bundled Vortex game extensions must expose runtime installer behavior", summary.ID)
		}
		if summary.VortexGameID != "" && summary.VortexStub {
			t.Fatalf("%s remains a Vortex support-mod shell; bundled Vortex game extensions must be expanded into DMM runtime behavior", summary.ID)
		}
		if len(summary.ParityGaps) != 0 {
			t.Fatalf("%s advertises unresolved parity gaps: %+v", summary.ID, summary.ParityGaps)
		}
		for _, feature := range allFeatureSummaries(summary.Capabilities) {
			switch feature.Status {
			case sdk.CapabilityStatusBlocked, sdk.CapabilityStatusMetadata:
				t.Fatalf("%s advertises unresolved %s capability %s: %+v", summary.ID, feature.Status, feature.ID, feature)
			}
			message := strings.ToLower(feature.Message)
			for _, forbidden := range []string{"tracked separately", "manual install-path registration", "manual registration", "outside the steam deck runtime boundary"} {
				if strings.Contains(message, forbidden) {
					t.Fatalf("%s advertises incomplete parity wording %q in %s: %+v", summary.ID, forbidden, feature.ID, feature)
				}
			}
		}
	}
}

func addFeatures(counts map[string]int, name string, features []gameext.FeatureSummary) {
	counts[name] += len(features)
}

func allFeatureSummaries(caps gameext.ExtensionCapabilities) []gameext.FeatureSummary {
	features := []gameext.FeatureSummary{}
	features = append(features, caps.ModTypes...)
	features = append(features, caps.Installers...)
	features = append(features, caps.UnsupportedInstallers...)
	features = append(features, caps.InstallerChoices...)
	features = append(features, caps.RuntimeRequirements...)
	features = append(features, caps.LaunchTools...)
	features = append(features, caps.LaunchOptionRequirements...)
	features = append(features, caps.SupportedTools...)
	features = append(features, caps.LauncherRequirements...)
	features = append(features, caps.InstallPlatforms...)
	features = append(features, caps.GameVersions...)
	features = append(features, caps.GameInfoProviders...)
	features = append(features, caps.PluginActivations...)
	features = append(features, caps.UnmanagedMarkers...)
	features = append(features, caps.ExternalModAdoptions...)
	features = append(features, caps.ConflictIgnores...)
	features = append(features, caps.DeployIgnores...)
	features = append(features, caps.PackedArchiveMutations...)
	features = append(features, caps.TargetRoots...)
	features = append(features, caps.Merges...)
	features = append(features, caps.LoadOrders...)
	features = append(features, caps.ArchiveTypes...)
	features = append(features, caps.Interpreters...)
	features = append(features, caps.GameStores...)
	features = append(features, caps.GameSetups...)
	features = append(features, caps.ExtensionActions...)
	features = append(features, caps.ExtensionSettings...)
	features = append(features, caps.ExtensionTests...)
	features = append(features, caps.ExtensionToDos...)
	features = append(features, caps.ExtensionDialogs...)
	features = append(features, caps.ExtensionDashlets...)
	features = append(features, caps.ExtensionDynamicDividers...)
	features = append(features, caps.ExtensionMainPages...)
	features = append(features, caps.ExtensionTableAttrs...)
	features = append(features, caps.ExtensionLoadOrderPages...)
	features = append(features, caps.ExtensionActionChecks...)
	features = append(features, caps.ExtensionControlWrappers...)
	features = append(features, caps.ExtensionAPIs...)
	features = append(features, caps.ProfileFeatures...)
	features = append(features, caps.ProfileFiles...)
	features = append(features, caps.SavegameManagement...)
	features = append(features, caps.CollectionFeatures...)
	features = append(features, caps.StateReducers...)
	features = append(features, caps.StatePersistors...)
	features = append(features, caps.StateStores...)
	features = append(features, caps.StateMigrations...)
	features = append(features, caps.HistoryStacks...)
	features = append(features, caps.HealthChecks...)
	features = append(features, caps.AttributeExtractors...)
	features = append(features, caps.StartHooks...)
	features = append(features, caps.EventHandlers...)
	features = append(features, caps.StateChangeWatchers...)
	if caps.SteamWorkshop != nil {
		features = append(features, caps.SteamWorkshop.Actions...)
	}
	return features
}

func featureSlice(summary *gameext.GameRegistrationSummary) []gameext.FeatureSummary {
	if summary == nil {
		return nil
	}
	return []gameext.FeatureSummary{{ID: "game-registration"}}
}

func supportModFeature(supportModID string) []gameext.FeatureSummary {
	if supportModID == "" {
		return nil
	}
	return []gameext.FeatureSummary{{ID: "support-mod-" + supportModID}}
}

func hasSuffixModType(modTypes map[string]bool, suffix string) bool {
	for id := range modTypes {
		if strings.HasSuffix(id, suffix) {
			return true
		}
	}
	return false
}

func addFeatureMap(dst map[string]map[string]gameext.FeatureSummary, kind string, features []gameext.FeatureSummary) {
	if dst[kind] == nil {
		dst[kind] = map[string]gameext.FeatureSummary{}
	}
	for _, feature := range features {
		dst[kind][feature.ID] = feature
	}
}

func findFeatureWithSuffix(features map[string]gameext.FeatureSummary, suffix string) (gameext.FeatureSummary, bool) {
	for id, feature := range features {
		if strings.HasSuffix(id, suffix) {
			return feature, true
		}
	}
	return gameext.FeatureSummary{}, false
}
