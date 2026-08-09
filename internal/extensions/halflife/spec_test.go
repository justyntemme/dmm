package halflife_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/halflife"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersInstallerCoverage(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(halflife.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.UnsupportedInstallers) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if !registry.SupportsSteamApp(halflife.SteamAppID) {
		t.Fatal("registry does not support Half-Life")
	}
}

func TestValveArchivePreservesValveRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "valve", "maps", "subtitles.bsp"), "map")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "valve/maps/subtitles.bsp")
}

func TestLooseMapArchiveTargetsValveMaps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Sandbox.bsp"), "map")
	writeFile(t, filepath.Join(root, "readme.md"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "valve/maps/Sandbox.bsp")
	if hasTarget(plan, "valve/maps/readme.md") {
		t.Fatalf("unexpected readme target in %+v", plan.Instructions)
	}
}

func TestStandaloneGoldSrcModFolderIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sm_valve", "liblist.gam"), "game")
	writeFile(t, filepath.Join(root, "sm_valve", "maps", "intro.bsp"), "map")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "not classified") {
		t.Fatalf("err = %v", err)
	}
}

func build(root string) (installplan.Plan, error) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(halflife.Extension())})
	return registry.BuildInstallPlan(halflife.SteamAppID, root)
}

func assertTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	if hasTarget(plan, target) {
		return
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
}

func hasTarget(plan installplan.Plan, target string) bool {
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return true
		}
	}
	return false
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
