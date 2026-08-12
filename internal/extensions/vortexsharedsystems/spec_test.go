package vortexsharedsystems

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
	assertReadyWithMessage(t, "extension API", summary.Capabilities.ExtensionAPIs, "isBlueprintPlugin")
	assertReadyWithMessage(t, "extension API", summary.Capabilities.ExtensionAPIs, "new-file-ambiguous-adoption")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "dependency-manage-rules")
	assertReadyWithMessage(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "dependency-connector")
	assertReadyWithMessage(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "dependency-editor")
	assertReadyWithMessage(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "conflict-editor")
	assertReadyWithMessage(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "file-override-editor")
	assertReadyWithMessage(t, "extension dialog", summary.Capabilities.ExtensionDialogs, "dependency-cycle-graph")
	assertReadyWithMessage(t, "control wrapper", summary.Capabilities.ExtensionControlWrappers, "dependency-mod-name-wrapper")
	assertReadyWithMessage(t, "extension setting", summary.Capabilities.ExtensionSettings, "dependency-workarounds")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "game-version-gamemode")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "game-version-mod-installed")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "local-game-settings-global-files")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "test-setup-uninstall-entry")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "gamebryo-invalid-userlist")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "gamebryo-missing-groups")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "gamebryo-plugin-index-lock")
	assertReadyWithMessage(t, "action check", summary.Capabilities.ExtensionActionChecks, "gamebryo-userlist-duplicate-rule-check")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "dependency-rules")
	assertReadyWithMessage(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "dependency-load-order")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "gamebryo-savegames")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "local_game_settings")
	assertReadyWithMessage(t, "profile feature", summary.Capabilities.ProfileFeatures, "local_saves")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-plugin-index-lock")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "dependency-workarounds")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "dependency-session")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-save-session")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "gamebryo-save-settings")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-transfer")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-refresh")
	assertReadyWithMessage(t, "extension action", summary.Capabilities.ExtensionActions, "gamebryo-save-open")
	assertReadyWithMessage(t, "extension main page", summary.Capabilities.ExtensionMainPages, "gamebryo-savegames")
	assertReadyWithMessage(t, "extension main page", summary.Capabilities.ExtensionMainPages, "morrowind-plugins")
	assertReadyWithMessage(t, "state reducer", summary.Capabilities.StateReducers, "test-gameversion-state")
	assertReady(t, "extension test", summary.Capabilities.ExtensionTests, "gamebryo-incompatible-mod-archives", "oblivion-fonts", "skyrim-fonts")
	assertReadyWithMessage(t, "extension test", summary.Capabilities.ExtensionTests, "dependency-unsolved-conflicts")
	assertReady(t, "game info provider", summary.Capabilities.GameInfoProviders, "game-version")
	assertStatusWithMessage(t, "start hook", summary.Capabilities.StartHooks, "dependency-check-unsolved-conflicts", sdk.CapabilityStatusReady)
	assertReadyWithMessage(t, "history stack", summary.Capabilities.HistoryStacks, "plugins")
	assertReadyWithMessage(t, "game info provider", summary.Capabilities.GameInfoProviders, "steam")
}

func TestSteamGameInfoProviderReadsSteamAppDetails(t *testing.T) {
	var gotAppIDs string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAppIDs = r.URL.Query().Get("appids")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"413150": {
				"success": true,
				"data": {
					"release_date": {"date": "Feb 26, 2016"},
					"website": "https://www.stardewvalley.net/",
					"metacritic": {"score": "89"}
				}
			}
		}`))
	}))
	defer server.Close()

	previousURL := steamAppDetailsURL
	steamAppDetailsURL = server.URL
	t.Cleanup(func() { steamAppDetailsURL = previousURL })

	result, err := steamGameInfoProvider(context.Background(), sdk.GameInfoInput{AppID: "413150"})
	if err != nil {
		t.Fatalf("steamGameInfoProvider returned error: %v", err)
	}
	if gotAppIDs != "413150" {
		t.Fatalf("appids query = %q", gotAppIDs)
	}
	got := map[string]sdk.GameInfoDetail{}
	for _, detail := range result.Details {
		got[detail.ID] = detail
	}
	for _, id := range []string{"release_date", "website", "metacritic_score"} {
		if got[id].ID == "" {
			t.Fatalf("missing %s in %+v", id, result.Details)
		}
		if got[id].Source != "steam" {
			t.Fatalf("%s source = %q", id, got[id].Source)
		}
	}
	if got["metacritic_score"].Value != 89 {
		t.Fatalf("metacritic value = %#v", got["metacritic_score"].Value)
	}
	if !strings.Contains(got["website"].Value.(string), "stardewvalley") {
		t.Fatalf("website detail = %+v", got["website"])
	}
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
