package sharedmodtypes

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersBlockedSharedModTypeMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	modTypes := map[string]gameext.FeatureSummary{}
	for _, modType := range summary.Capabilities.ModTypes {
		modTypes[modType.ID] = modType
	}
	for _, id := range []string{"dinput", "enb", "gedosato"} {
		if modTypes[id].Status != sdk.CapabilityStatusBlocked || modTypes[id].Message == "" {
			t.Fatalf("%s mod type = %+v", id, modTypes[id])
		}
	}
	if len(summary.Capabilities.UnsupportedInstallers) != 2 {
		t.Fatalf("unsupported installers = %+v", summary.Capabilities.UnsupportedInstallers)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 0 {
		t.Fatalf("extension apis = %+v", summary.Capabilities.ExtensionAPIs)
	}
}
