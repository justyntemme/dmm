package witcher3_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/witcher3"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansTopLevelModsArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mods", "modExample", "content", "scripts", "example.ws"), "script")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3tl" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Mods/modExample/content/scripts/example.ws")
}

func TestExtensionPlansTopLevelDLCArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "DLC", "DLCExample", "content", "example.bundle"), "bundle")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3dlc" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "DLC/DLCExample/content/example.bundle")
}

func TestExtensionBlocksScriptMergerModArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "WitcherScriptMerger.exe"), "tool")

	_, err := build(root)
	if err == nil {
		t.Fatal("expected unsupported script merger archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "tool, not a mod") {
		t.Fatalf("error = %v", err)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(witcher3.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("witcher3", root)
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
