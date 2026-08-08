package commandconquergenerals

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

func TestExtensionRegistersBigInstaller(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	coverage, _ := gameext.ExtensionCoverage(ext)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
	if len(ext.NexusDomains) != 1 || ext.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", ext.NexusDomains)
	}
	if len(ext.InstallPlan.Installers) != 2 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestBigArchivePlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MyMod", "CoolMod.big"), "big")
	writeFile(t, filepath.Join(root, "MyMod", "readme.txt"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bigModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "CoolMod.big" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestUnclassifiedGeneralsArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "GenLauncher.exe"), "tool")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Command & Conquer") {
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
