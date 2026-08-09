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
	if arc.ID != "arc" || arc.Engine != ID || !arc.SupportsWrite || arc.Status != sdk.CapabilityStatusBlocked || arc.Message == "" {
		t.Fatalf("arc capability = %+v", arc)
	}
}
