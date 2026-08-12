package mtframeworkarc

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedArcMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.ArchiveTypes) != 1 {
		t.Fatalf("archive types = %+v", summary.Capabilities.ArchiveTypes)
	}
	arc := summary.Capabilities.ArchiveTypes[0]
	if arc.ID != "arc" || arc.Engine != ID || !arc.SupportsWrite || arc.Status != sdk.CapabilityStatusReady || arc.Message == "" {
		t.Fatalf("arc capability = %+v", arc)
	}
	if len(summary.Capabilities.ExtensionDashlets) != 1 {
		t.Fatalf("dashlets = %+v", summary.Capabilities.ExtensionDashlets)
	}
	dashlet := summary.Capabilities.ExtensionDashlets[0]
	if dashlet.ID != "mtframework-arc-support" || dashlet.Status != sdk.CapabilityStatusReady || dashlet.Message == "" {
		t.Fatalf("dashlet = %+v", dashlet)
	}
}
