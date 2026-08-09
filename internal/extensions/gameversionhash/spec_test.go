package gameversionhash

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedHashVersionMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.GameVersions) != 1 || summary.Capabilities.GameVersions[0].Status != sdk.CapabilityStatusBlocked {
		t.Fatalf("game versions = %+v", summary.Capabilities.GameVersions)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 1 || summary.Capabilities.ExtensionAPIs[0].ID != "getHashVersion" || summary.Capabilities.ExtensionAPIs[0].Message == "" {
		t.Fatalf("extension apis = %+v", summary.Capabilities.ExtensionAPIs)
	}
}
