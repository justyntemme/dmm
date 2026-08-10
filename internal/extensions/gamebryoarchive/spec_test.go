package gamebryoarchive

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedArchiveMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework || summary.Coverage != gameext.CoverageFramework {
		t.Fatalf("summary = %+v", summary)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, archiveType := range summary.Capabilities.ArchiveTypes {
		byID[archiveType.ID] = archiveType
	}
	if byID["ba2"].Engine != ID || byID["ba2"].Status != sdk.CapabilityStatusReady || byID["ba2"].Message != "" {
		t.Fatalf("ba2 capability = %+v", byID["ba2"])
	}
	if byID["bsa"].Engine != ID || !byID["bsa"].SupportsWrite || byID["bsa"].Status != sdk.CapabilityStatusReady {
		t.Fatalf("bsa capability = %+v", byID["bsa"])
	}
	if byID["bsa"].Message != "" {
		t.Fatalf("bsa message = %q", byID["bsa"].Message)
	}
}
