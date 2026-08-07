package totalwarrome2

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

func TestExtensionRegistersResearchBlockedRomeII(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	if ext.ID != VortexGameID {
		t.Fatalf("extension id = %q", ext.ID)
	}
	if len(ext.NexusDomains) != 1 || ext.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", ext.NexusDomains)
	}
	if ext.SteamWorkshop.AllowCoexistence {
		t.Fatalf("Rome II should not advertise Workshop support without verified Steam Workshop category")
	}
	if len(ext.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	installer := ext.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionUnsupported || !strings.Contains(installer.UnsupportedReason, "representative Nexus archives") {
		t.Fatalf("installer = %+v", installer)
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 1 || ext.RuntimeRequirements.RuntimeRequirements[0].ID != "totalwarrome2-required-files" {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestRomeIIArchivesAreBlockedUntilClassified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "some-mod.pack"), "pack")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Total War: ROME II") {
		t.Fatalf("unsupported error = %v", err)
	}
}

func TestRequiredFilesCheck(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
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
