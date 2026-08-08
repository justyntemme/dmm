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

func TestExtensionRegistersExternalFileInstallers(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	coverage, _ := gameext.ExtensionCoverage(ext)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
	if len(ext.InstallPlan.Installers) != 3 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 2 {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
}

func TestExternalFileLoaderPlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "loader", "dinput8.dll"), "dll")
	writeFile(t, filepath.Join(root, "loader", "hook.ini"), "ini")
	writeFile(t, filepath.Join(root, "loader", "modules", "ff10-file-loader.dll"), "dll")
	writeFile(t, filepath.Join(root, "loader", "readme.txt"), "readme")

	plan, err := buildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != loaderModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := instructionTargets(plan)
	for _, want := range []string{"dinput8.dll", "hook.ini", "modules/ff10-file-loader.dll"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["readme.txt"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestExternalFileModPlansToDataMods(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ReRemaster", "ffx_data", "gamedata", "ps3data", "fonts", "d3d11", "tuffy.fgen.phyre"), "font")
	writeFile(t, filepath.Join(root, "ReRemaster", "readme.txt"), "readme")

	plan, err := buildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != externalFileModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := instructionTargets(plan)
	want := "data/mods/ffx_data/gamedata/ps3data/fonts/d3d11/tuffy.fgen.phyre"
	if !targets[want] {
		t.Fatalf("targets = %+v, missing %s", targets, want)
	}
}

func TestUnclassifiedFFXArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "UnX.exe"), "tool")

	_, err := buildPlan(root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Final Fantasy X/X-2") {
		t.Fatalf("unsupported error = %v", err)
	}
}

func TestRequiredFilesAndLoaderChecks(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	writeFile(t, filepath.Join(root, "dinput8.dll"), "dll")
	writeFile(t, filepath.Join(root, "modules", "ff10-file-loader.dll"), "dll")
	if got := checkRequiredGameFiles(context.Background(), root); len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
	if got := checkExternalFileLoader(context.Background(), root); len(got) != 2 {
		t.Fatalf("loader details = %+v", got)
	}
}

func buildPlan(root string) (installplan.Plan, error) {
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	return registry.Build(SteamAppID, root)
}

func instructionTargets(plan installplan.Plan) map[string]bool {
	out := map[string]bool{}
	for _, instruction := range plan.Instructions {
		out[instruction.TargetRelative] = true
	}
	return out
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
