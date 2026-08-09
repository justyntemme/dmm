package commoninterpreters

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexCommonInterpreters(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework || summary.Coverage != gameext.CoverageFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Interpreters) != 5 {
		t.Fatalf("interpreters = %+v", summary.Capabilities.Interpreters)
	}
	byID := map[string]gameext.FeatureSummary{}
	for _, interpreter := range summary.Capabilities.Interpreters {
		byID[interpreter.ID] = interpreter
	}
	for _, id := range []string{"jar", "python", "vbs", "cmd", "bat"} {
		if byID[id].ID == "" {
			t.Fatalf("missing interpreter %s in %+v", id, summary.Capabilities.Interpreters)
		}
	}
	if got := byID["cmd"].Platforms; len(got) != 1 || got[0] != "windows" {
		t.Fatalf("cmd platforms = %+v", got)
	}
	if got := byID["jar"].Platforms; len(got) != 2 || got[0] != "linux" || got[1] != "windows" {
		t.Fatalf("jar platforms = %+v", got)
	}
}
