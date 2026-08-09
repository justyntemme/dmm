package torchlight2_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/torchlight2"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(torchlight2.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != "torchlight2-documents-mods" {
		t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "vortex:torchlight2:mod" {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.LauncherRequirements) != 1 {
		t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
	}
}

func TestModInstallerCopiesOnlyModFilesIntoNamedFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Cool.MOD"), "mod")
	writeFile(t, filepath.Join(root, "nested", "Other.mod"), "mod")
	writeFile(t, filepath.Join(root, "readme.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:torchlight2:mod" || plan.ModType != "torchlight2-mod" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "Cool/Cool.MOD", "torchlight2-documents-mods")
	assertTarget(t, plan, "Other/Other.mod", "torchlight2-documents-mods")
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestDocumentsTargetRootUsesDeckUserDocuments(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(torchlight2.Extension())})
	result, ok, err := registry.ResolveTargetRoot(context.Background(), torchlight2.SteamAppID, "torchlight2-documents-mods", gameext.TargetRootInput{})
	if err != nil || !ok {
		t.Fatalf("resolve target root ok=%v err=%v", ok, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Documents", "My Games", "runic games", "torchlight 2", "mods")
	if result.Path != want {
		t.Fatalf("target root = %q, want %q", result.Path, want)
	}
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(torchlight2.Extension())}).BuildInstallPlan(torchlight2.SteamAppID, root)
}

func assertTarget(t *testing.T, plan installplan.Plan, target, targetRoot string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target && instruction.TargetRoot == targetRoot {
			return
		}
	}
	t.Fatalf("missing target %q root %q in %+v", target, targetRoot, plan.Instructions)
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
