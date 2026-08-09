package mirrorsedge_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/mirrorsedge"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersInstallerCoverage(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(mirrorsedge.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.UnsupportedInstallers) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestTdGameCookedPCArchiveTargetsCookedPC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "TdGame", "CookedPC", "Characters", "CH_Faith_Cinematic.upk"), "upk")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "TdGame/CookedPC/Characters/CH_Faith_Cinematic.upk")
}

func TestBareCookedPCArchiveTargetsCookedPC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CookedPC", "Characters", "CH_TKY_Crim_Fixer.upk"), "upk")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "TdGame/CookedPC/Characters/CH_TKY_Crim_Fixer.upk")
}

func TestCookedPCTopLevelArchiveTargetsCookedPC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Characters", "CH_TKY_Crim_Fixer_1P.upk"), "upk")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "TdGame/CookedPC/Characters/CH_TKY_Crim_Fixer_1P.upk")
}

func TestExecutableArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "installer.exe"), "exe")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "not classified") {
		t.Fatalf("err = %v", err)
	}
}

func build(root string) (installplan.Plan, error) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(mirrorsedge.Extension())})
	return registry.BuildInstallPlan(mirrorsedge.SteamAppID, root)
}

func assertTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
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
