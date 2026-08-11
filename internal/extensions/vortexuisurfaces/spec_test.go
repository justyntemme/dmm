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
	if len(summary.Sources) == 0 {
		t.Fatalf("summary should retain source references: %+v", summary)
	}
	if len(summary.Capabilities.StartHooks) != 0 {
		t.Fatalf("generic UI surface extension should not advertise startup hooks after source-backed hook runtime moved to vortexsharedsystems: %+v", summary.Capabilities.StartHooks)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 2 {
		t.Fatalf("extension APIs = %+v", summary.Capabilities.ExtensionAPIs)
	}
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "open-directory-action", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "mod-content-classifier", sdk.CapabilityStatusReady)
	assertStatus(t, "extension action", summary.Capabilities.ExtensionActions, "mo-import", sdk.CapabilityStatusNotApplicable)
	assertStatus(t, "extension action", summary.Capabilities.ExtensionActions, "nmm-import", sdk.CapabilityStatusNotApplicable)
	assertStatus(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "mo-import", sdk.CapabilityStatusNotApplicable)
	assertStatus(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "nmm-import", sdk.CapabilityStatusNotApplicable)
	assertStatus(t, "extension todo", summary.Capabilities.ExtensionToDos, "import-nmm", sdk.CapabilityStatusNotApplicable)
	assertStatus(t, "state reducer", summary.Capabilities.StateReducers, "nmm-import-session", sdk.CapabilityStatusNotApplicable)
}

func assertStatus(t *testing.T, kind string, features []gameext.FeatureSummary, id, status string) {
	t.Helper()
	for _, feature := range features {
		if feature.ID != id {
			continue
		}
		if feature.Status != status || feature.Message == "" {
			t.Fatalf("%s %s = %+v", kind, id, feature)
		}
		return
	}
	t.Fatalf("%s %s missing from %+v", kind, id, features)
}
