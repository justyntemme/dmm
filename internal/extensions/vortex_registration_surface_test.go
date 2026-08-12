package extensions

import (
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

func TestFirstPartyExtensionsAdvertiseNoUnresolvedParitySurfaces(t *testing.T) {
	for _, summary := range gameext.NewRegistry(FirstParty()).ExtensionSummaries() {
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
