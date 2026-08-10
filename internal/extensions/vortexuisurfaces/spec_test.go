package vortexuisurfaces

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersNonApplicableVortexUISurfaceMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	assertNonApplicable(t, "extension API", summary.Capabilities.ExtensionAPIs, "registerAction", "registerDialog", "registerReducer")
	assertNonApplicable(t, "dialog", summary.Capabilities.ExtensionDialogs, "registerDialog")
	assertNonApplicable(t, "dashlet", summary.Capabilities.ExtensionDashlets, "registerDashlet")
	assertNonApplicable(t, "main page", summary.Capabilities.ExtensionMainPages, "registerMainPage")
	assertNonApplicable(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "registerTableAttribute")
	assertNonApplicable(t, "action check", summary.Capabilities.ExtensionActionChecks, "registerActionCheck")
	assertNonApplicable(t, "control wrapper", summary.Capabilities.ExtensionControlWrappers, "registerControlWrapper")
	assertNonApplicable(t, "profile file", summary.Capabilities.ProfileFiles, "registerProfileFile")
	assertNonApplicable(t, "state reducer", summary.Capabilities.StateReducers, "registerReducer")
	assertNonApplicable(t, "state persistor", summary.Capabilities.StatePersistors, "registerPersistor")
	if len(summary.Capabilities.StartHooks) != 0 {
		t.Fatalf("generic UI surface extension should not advertise startup hooks after source-backed hook runtime moved to vortexsharedsystems: %+v", summary.Capabilities.StartHooks)
	}
}

func assertNonApplicable(t *testing.T, kind string, features []gameext.FeatureSummary, ids ...string) {
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
		if feature.Status != sdk.CapabilityStatusNotApplicable || feature.Message == "" {
			t.Fatalf("%s %s = %+v", kind, id, feature)
		}
	}
}
