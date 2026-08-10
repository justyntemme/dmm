package vortexsharedsystems

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersBlockedSharedSystemMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	assertReady(t, "extension API", summary.Capabilities.ExtensionAPIs, "deploy-single-mod", "purge-mods-in-path", "browse-for-download", "nexus-download", "discover-tools", "unfulfilled-rules", "registerGameInfoProvider", "new-file-single-owner-adoption")
	assertReady(t, "extension API", summary.Capabilities.ExtensionAPIs, "oblivion-font-repair")
	assertReadyWithMessage(t, "extension API", summary.Capabilities.ExtensionAPIs, "new-file-ambiguous-adoption")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "dependency-manage-rules")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "game-version-gamemode")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "game-version-mod-installed")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "local-game-settings-global-files")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "test-setup-uninstall-entry")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "gamebryo-plugin-index-lock")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "dependency-rules")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "dependency-load-order")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "gamebryo-savegames")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "local_game_settings")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "local_saves")
	assertReady(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-plugin-index-lock")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "dependency-workarounds")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "dependency-session")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-save-session")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-save-settings")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-transfer")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-refresh")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-open")
	assertReadyWithMessage(t, "extension main page", summary.Capabilities.ExtensionMainPages, "gamebryo-savegames")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "test-gameversion-state")
	assertReady(t, "extension test", summary.Capabilities.ExtensionTests, "gamebryo-incompatible-mod-archives", "oblivion-fonts", "skyrim-fonts")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "dependency-unsolved-conflicts")
	assertReady(t, "game info provider", summary.Capabilities.GameInfoProviders, "game-version")
	assertStatusWithMessage(t, "start hook", summary.Capabilities.StartHooks, "dependency-check-unsolved-conflicts", sdk.CapabilityStatusReady)
	assertReadyWithMessage(t, "history stack", summary.Capabilities.HistoryStacks, "plugins")
}

func assertBlocked(t *testing.T, kind string, features []gameext.FeatureSummary, ids ...string) {
	t.Helper()
	featuresByID := map[string]gameext.FeatureSummary{}
	for _, feature := range features {
		featuresByID[feature.ID] = feature
	}
	for _, id := range ids {
		feature, ok := featuresByID[id]
		if !ok {
			t.Fatalf("%s %s missing from %+v", kind, id, features)
		}
		if feature.Status != sdk.CapabilityStatusBlocked || feature.Message == "" {
			t.Fatalf("%s %s = %+v", kind, id, feature)
		}
	}
}

func assertReady(t *testing.T, kind string, features []gameext.FeatureSummary, ids ...string) {
	t.Helper()
	featuresByID := map[string]gameext.FeatureSummary{}
	for _, feature := range features {
		featuresByID[feature.ID] = feature
	}
	for _, id := range ids {
		feature, ok := featuresByID[id]
		if !ok {
			t.Fatalf("%s %s missing from %+v", kind, id, features)
		}
		if feature.Status != sdk.CapabilityStatusReady || feature.Message != "" {
			t.Fatalf("%s %s = %+v", kind, id, feature)
		}
	}
}

func assertReadyWithMessage(t *testing.T, kind string, features []gameext.FeatureSummary, id string) {
	t.Helper()
	assertStatusWithMessage(t, kind, features, id, sdk.CapabilityStatusReady)
}

func assertStatusWithMessage(t *testing.T, kind string, features []gameext.FeatureSummary, id, status string) {
	t.Helper()
	featuresByID := map[string]gameext.FeatureSummary{}
	for _, feature := range features {
		featuresByID[feature.ID] = feature
	}
	feature, ok := featuresByID[id]
	if !ok {
		t.Fatalf("%s %s missing from %+v", kind, id, features)
	}
	if feature.Status != status || feature.Message == "" {
		t.Fatalf("%s %s = %+v", kind, id, feature)
	}
}
