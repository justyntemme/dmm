package manifestblocked

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersBlockedManifestShape(t *testing.T) {
	spec := Spec{
		ID:                "examplegame",
		Name:              "Example Game",
		SteamAppIDs:       []string{"123"},
		NexusDomains:      []string{"examplegame"},
		VortexGameID:      "examplegame",
		UnsupportedReason: "source package has not been inspected",
		RequiredFiles:     []string{"Example.exe", "Data"},
		Sources: []sdk.SourceRef{{
			Name: "source",
			URL:  "https://example.test",
		}},
	}
	extension := gameext.MustCompileExtension(Extension(spec))
	if extension.ID != spec.ID || extension.Name != spec.Name {
		t.Fatalf("extension = %+v", extension)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionUnsupported {
		t.Fatalf("installer = %+v", installer)
	}
	if len(extension.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", extension.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestBlockedInstallerReturnsConfiguredReason(t *testing.T) {
	const reason = "blocked until Vortex package source is reviewed"
	extension := gameext.MustCompileExtension(Extension(Spec{
		ID:                "examplegame",
		Name:              "Example Game",
		SteamAppIDs:       []string{"123"},
		NexusDomains:      []string{"examplegame"},
		VortexGameID:      "examplegame",
		UnsupportedReason: reason,
	}))
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "payload.txt"), "payload")
	registry := installplan.NewRegistry([]installplan.GameSpec{extension.InstallPlan})
	_, err := registry.Build("123", root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), reason) {
		t.Fatalf("error = %T %v", err, err)
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
