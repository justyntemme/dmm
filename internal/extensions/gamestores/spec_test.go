package gamestores

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedGameStoreMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, store := range summary.Capabilities.GameStores {
		byID[store.ID] = store
	}
	for _, id := range []string{"gog", "origin", "uplay", "epic", "xbox"} {
		if byID[id].Status != "ready" || byID[id].Message == "" {
			t.Fatalf("%s store = %+v", id, byID[id])
		}
	}
}
