package spelunky_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/spelunky"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersInstallerCoverage(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(spelunky.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.UnsupportedInstallers) != 0 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if !registry.SupportsSteamApp(spelunky.SteamAppID) {
		t.Fatal("registry does not support Spelunky")
	}
}

func TestDataWrapperArchiveTargetsDataFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Textures", "alltex.wad"), "wad")
	writeFile(t, filepath.Join(root, "Data", "Textures", "alltex.wad.wix"), "wix")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "Data/Textures/alltex.wad")
	assertTarget(t, plan, "Data/Textures/alltex.wad.wix")
}

func TestLooseDataSubfolderTargetsDataFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Localization", "strings.pct"), "strings")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "Data/Localization/strings.pct")
}

func build(root string) (installplan.Plan, error) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(spelunky.Extension())})
	return registry.BuildInstallPlan(spelunky.SteamAppID, root)
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
