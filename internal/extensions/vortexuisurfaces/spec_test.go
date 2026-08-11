package vortexuisurfaces

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersNonApplicableVortexUISurfaceMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Sources) == 0 {
		t.Fatalf("summary should retain source references: %+v", summary)
	}
	if len(summary.ParityGaps) != 0 {
		t.Fatalf("source-only UI surface extension should not advertise runtime gaps: %+v", summary.ParityGaps)
	}
	if len(summary.Capabilities.StartHooks) != 0 {
		t.Fatalf("generic UI surface extension should not advertise startup hooks after source-backed hook runtime moved to vortexsharedsystems: %+v", summary.Capabilities.StartHooks)
	}
}
