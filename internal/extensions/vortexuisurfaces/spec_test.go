package vortexuisurfaces

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexUISurfaceMetadata(t *testing.T) {
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
	if len(summary.Capabilities.ExtensionAPIs) != 10 {
		t.Fatalf("extension APIs = %+v", summary.Capabilities.ExtensionAPIs)
	}
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "open-directory-action", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "mod-content-classifier", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-stylesheet", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-notification", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-dialog", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-directory-picker", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-file-picker", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "run-executable", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "mod-meta-lookup-save", sdk.CapabilityStatusReady)
	assertStatus(t, "extension API", summary.Capabilities.ExtensionAPIs, "ui-locale-highlight-outdated", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dynamic divider", summary.Capabilities.ExtensionDynamicDividers, "mod-highlight-state-divider", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dynamic divider", summary.Capabilities.ExtensionDynamicDividers, "feedback-state-divider", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dynamic divider", summary.Capabilities.ExtensionDynamicDividers, "titlebar-launcher-main-toolbar", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dynamic divider", summary.Capabilities.ExtensionDynamicDividers, "titlebar-launcher-tools-controls", sdk.CapabilityStatusReady)
	assertStatus(t, "extension action", summary.Capabilities.ExtensionActions, "mod-report", sdk.CapabilityStatusReady)
	assertStatus(t, "extension setting", summary.Capabilities.ExtensionSettings, "interface-theme", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dashlet", summary.Capabilities.ExtensionDashlets, "changelog-dashlet", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dashlet", summary.Capabilities.ExtensionDashlets, "extension-dashlet", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dashlet", summary.Capabilities.ExtensionDashlets, "issue-tracker", sdk.CapabilityStatusReady)
	assertStatus(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "feedback-responder", sdk.CapabilityStatusReady)
	assertStatus(t, "state reducer", summary.Capabilities.StateReducers, "interface-theme-settings", sdk.CapabilityStatusReady)
	assertStatus(t, "state reducer", summary.Capabilities.StateReducers, "changelog-cache", sdk.CapabilityStatusReady)
	assertStatus(t, "state reducer", summary.Capabilities.StateReducers, "issues-persistent", sdk.CapabilityStatusReady)
	assertStatus(t, "state reducer", summary.Capabilities.StateReducers, "issues-session", sdk.CapabilityStatusReady)
	if len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("post-MVP import TODOs must not be advertised as runtime capabilities: %+v", summary.Capabilities.ExtensionToDos)
	}
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
