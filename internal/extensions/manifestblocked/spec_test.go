package manifestblocked

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersBlockedManifestShape(t *testing.T) {
	spec := Spec{
		ID:             "examplegame",
		Name:           "Example Game",
		SteamAppIDs:    []string{"123"},
		NexusDomains:   []string{"examplegame"},
		VortexGameID:   "examplegame",
		ResearchReason: "source package has not been inspected",
		RequiredFiles:  []string{"Example.exe", "Data"},
		Sources: []sdk.SourceRef{{
			Name: "source",
			URL:  "https://example.test",
		}},
	}
	extension := gameext.MustCompileExtension(Extension(spec))
	if extension.ID != spec.ID || extension.Name != spec.Name {
		t.Fatalf("extension = %+v", extension)
	}
	if len(extension.InstallPlan.Installers) != 0 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	if len(extension.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", extension.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestResearchOnlyExtensionDoesNotClaimArchiveSupport(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension(Spec{
		ID:             "examplegame",
		Name:           "Example Game",
		SteamAppIDs:    []string{"123"},
		NexusDomains:   []string{"examplegame"},
		VortexGameID:   "examplegame",
		ResearchReason: "pending Vortex package source review",
	}))
	if len(extension.InstallPlan.Installers) != 0 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
}

func TestRequiredFilesCheckAcceptsFilesAndDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example.exe"), "game")
	if err := os.Mkdir(filepath.Join(root, "Data"), 0o755); err != nil {
		t.Fatal(err)
	}
	check := RequiredFilesCheck([]string{"Example.exe", "Data"})
	got := check(context.Background(), root)
	if len(got) != 2 || !strings.HasSuffix(got[1], "/") {
		t.Fatalf("details = %+v", got)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
