package prototype2

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

func TestExtensionRegistersASIInstaller(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(extension.InstallPlan.Installers) != 2 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	coverage, _ := gameext.ExtensionCoverage(extension)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
}

func TestASIArchivePlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "prototype_fix.asi"), "asi")
	writeFile(t, filepath.Join(root, "prototype_fix.ini"), "ini")
	writeFile(t, filepath.Join(root, "dinput8.dll"), "loader")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"prototype_fix.asi", "prototype_fix.ini", "dinput8.dll"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["readme.txt"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestArchiveIsBlockedWithResearchReason(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Texmod.exe"), "tool")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Prototype 2 archive layout") {
		t.Fatalf("error = %T %v", err, err)
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
