package commandconquergeneralszerohour

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersWorkshopAndBigInstaller(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	coverage, _ := gameext.ExtensionCoverage(ext)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
	if len(ext.NexusDomains) != 0 {
		t.Fatalf("Zero Hour must not declare unverified Nexus domains: %+v", ext.NexusDomains)
	}
	if !ext.SteamWorkshop.AllowCoexistence || len(ext.SteamWorkshop.Actions) != 5 {
		t.Fatalf("workshop = %+v", ext.SteamWorkshop)
	}
	if len(ext.InstallPlan.Installers) != 1 || ext.InstallPlan.Installers[0].ModType != bigModType {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestBigArchivePlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ZeroHourMod", "CoolZH.big"), "big")
	writeFile(t, filepath.Join(root, "ZeroHourMod", "readme.txt"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bigModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "CoolZH.big" {
		t.Fatalf("instructions = %+v", plan.Instructions)
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
