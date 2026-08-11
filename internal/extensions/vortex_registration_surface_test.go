package extensions

import (
	"testing"

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

func addFeatures(counts map[string]int, name string, features []gameext.FeatureSummary) {
	counts[name] += len(features)
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
