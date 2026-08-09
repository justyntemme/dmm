package rometotalwar_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/rometotalwar"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersRomeAndAlexanderAppIDs(t *testing.T) {
	extension := gameext.MustCompileExtension(rometotalwar.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.UnsupportedInstallers) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	want := map[string]bool{
		rometotalwar.SteamAppID:          false,
		rometotalwar.AlexanderSteamAppID: false,
	}
	for _, appID := range summary.SteamAppIDs {
		if _, ok := want[appID]; ok {
			want[appID] = true
		}
	}
	for appID, found := range want {
		if !found {
			t.Fatalf("missing app id %s in %+v", appID, summary.SteamAppIDs)
		}
		if !registry.SupportsSteamApp(appID) {
			t.Fatalf("registry does not support %s", appID)
		}
	}
}

func TestRomeDataInstallerTargetsVanillaDataFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "export_descr_buildings.txt"), "buildings")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(rometotalwar.Extension())})
	plan, err := registry.BuildInstallPlan(rometotalwar.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "data/export_descr_buildings.txt")
}

func TestRomeDataInstallerTargetsAlexanderDataFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "data", "descr_fmv.txt"), "fmv")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(rometotalwar.Extension())})
	plan, err := registry.BuildInstallPlan(rometotalwar.AlexanderSteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "alexander/data/descr_fmv.txt")
}

func TestRomeUnclassifiedArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "launcher.exe"), "exe")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(rometotalwar.Extension())})
	_, err := registry.BuildInstallPlan(rometotalwar.SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "not classified") {
		t.Fatalf("err = %v", err)
	}
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
