package finalfantasyxx2hdremaster

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersResearchBlockedNexusDomain(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionUnsupported || installer.ModType != researchModType {
		t.Fatalf("installer = %+v", installer)
	}
	if !strings.Contains(installer.UnsupportedReason, "no verified Vortex extension") {
		t.Fatalf("unsupported reason = %q", installer.UnsupportedReason)
	}
}

func TestArchiveInstallIsBlockedUntilPatternsAreVerified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "payload.bin"), "payload")
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "representative archives") {
		t.Fatalf("error = %T %v", err, err)
	}
}

func TestRequiredFilesChecks(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "payload")
	}
	if got := checkRequiredGameFiles(context.Background(), root); len(got) != len(requiredGameFiles) {
		t.Fatalf("required files = %+v", got)
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
