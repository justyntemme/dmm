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
	assertBlocked(t, "extension API", summary.Capabilities.ExtensionAPIs, "deploy-single-mod", "purge-mods-in-path", "registerGameInfoProvider")
	assertBlocked(t, "extension action", summary.Capabilities.ExtensionActions, "fnis-generate", "dependency-manage-rules")
	assertBlocked(t, "extension test", summary.Capabilities.ExtensionTests, "fnis-integration", "gamebryo-incompatible-mod-archives", "game-version-gamemode")
	assertBlocked(t, "table attribute", summary.Capabilities.ExtensionTableAttrs, "gamebryo-plugin-index-lock", "dependency-rules")
	assertBlocked(t, "profile feature", summary.Capabilities.ProfileFeatures, "gamebryo-savegames", "local-game-settings")
	assertBlocked(t, "game info provider", summary.Capabilities.GameInfoProviders, "game-version")
	assertBlocked(t, "start hook", summary.Capabilities.StartHooks, "dependency-check-unsolved-conflicts")
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
