package vortexuisurfaces

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersBlockedVortexUISurfaceMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	assertBlocked(t, "extension API", summary.Capabilities.ExtensionAPIs, "registerAction", "registerDialog", "registerReducer", "registerStartHook")
	assertBlocked(t, "dialog", summary.Capabilities.ExtensionDialogs, "registerDialog")
	assertBlocked(t, "dashlet", summary.Capabilities.ExtensionDashlets, "registerDashlet")
	assertBlocked(t, "main page", summary.Capabilities.ExtensionMainPages, "registerMainPage")
	assertBlocked(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "registerTableAttribute")
	assertBlocked(t, "action check", summary.Capabilities.ExtensionActionChecks, "registerActionCheck")
	assertBlocked(t, "profile file", summary.Capabilities.ProfileFiles, "registerProfileFile")
	assertBlocked(t, "state reducer", summary.Capabilities.StateReducers, "registerReducer")
	assertBlocked(t, "state persistor", summary.Capabilities.StatePersistors, "registerPersistor")
	assertBlocked(t, "start hook", summary.Capabilities.StartHooks, "registerStartHook")
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
