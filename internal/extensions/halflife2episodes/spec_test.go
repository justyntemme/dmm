package halflife2episodes

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestEpisodeExtensionsAreResearchBlocked(t *testing.T) {
	extensions := Extensions()
	if len(extensions) != 3 {
		t.Fatalf("extensions = %+v", extensions)
	}
	for _, extensionSpec := range extensions {
		extension := gameext.MustCompileExtension(extensionSpec)
		summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
		if summary.Coverage != gameext.CoverageResearchBlocked {
			t.Fatalf("%s coverage = %+v", extension.ID, summary)
		}
		if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != nexusDomain {
			t.Fatalf("%s nexus domains = %+v", extension.ID, extension.NexusDomains)
		}
		if len(extension.InstallPlan.Installers) != 1 || extension.InstallPlan.Installers[0].InstructionMode != installplan.InstructionUnsupported {
			t.Fatalf("%s installers = %+v", extension.ID, extension.InstallPlan.Installers)
		}
	}
}
