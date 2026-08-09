package quickbmssupport

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedQuickBMSAPIMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, api := range summary.Capabilities.ExtensionAPIs {
		byID[api.ID] = api
	}
	for _, id := range []string{"qbmsRegisterGame", "qbmsList", "qbmsExtract", "qbmsWrite", "qbmsReimport"} {
		if byID[id].Status != sdk.CapabilityStatusBlocked || byID[id].Message == "" {
			t.Fatalf("%s capability = %+v", id, byID[id])
		}
	}
}
