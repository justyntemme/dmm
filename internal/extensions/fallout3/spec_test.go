package fallout3

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexStoreLocaleSetup(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil {
		t.Fatal("missing game registration")
	}
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "gog", gogAppID)
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "epic", epicAppID)
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "xbox", xboxAppID)
	for key, want := range map[string]string{"SteamAPPId": SteamAppIDGOTY, "GogAPPId": gogAppID, "EpicAPPId": epicAppID, "XboxAPPId": xboxAppID} {
		if got := summary.Capabilities.GameRegistration.Environment[key]; got != want {
			t.Fatalf("environment[%s] = %q, want %q", key, got, want)
		}
	}
	setup := featureByID(summary.Capabilities.GameSetups, "fallout3-store-locale-paths")
	if setup == nil || len(setup.SetupActions) != 2 {
		t.Fatalf("store locale setup = %+v", summary.Capabilities.GameSetups)
	}
	for _, action := range setup.SetupActions {
		if action.Kind != sdk.GameSetupActionSelectStoreLocalePath || action.RelativePath != "Fallout 3 GOTY English" || len(action.CandidatePaths) != 5 {
			t.Fatalf("setup action = %+v", action)
		}
	}
}

func assertStoreIdentity(t *testing.T, summary *gameext.GameRegistrationSummary, store, appID string) {
	t.Helper()
	for _, value := range summary.StoreAppIDs[store] {
		if value == appID {
			return
		}
	}
	t.Fatalf("store app ids[%s] = %+v, want %q", store, summary.StoreAppIDs[store], appID)
}

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for _, feature := range features {
		if feature.ID == id {
			return &feature
		}
	}
	return nil
}
