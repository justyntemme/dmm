package neverwinter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/neverwinter"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestNWNExtensionsRegisterVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(neverwinter.Extensions()[1])}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.ModTypes) != 2 || len(summary.Capabilities.TargetRoots) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestNWNStructuredArchivePreservesKnownFolderStructure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "modules", "adventure.mod"), "module")
	writeFile(t, filepath.Join(root, "wrapped", "hak", "tileset.hak"), "hak")
	writeFile(t, filepath.Join(root, "readme.txt"), "ignored")

	plan, err := build("704450", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:nwnee:structured" || plan.ModType != "nwnee-structured" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "modules/adventure.mod")
	assertTarget(t, plan, "hak/tileset.hak")
	assertNoTarget(t, plan, "readme.txt")
}

func TestNWNLooseArchiveMapsByExtensionAndPreservesOverrideFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "portrait.tga"), "portrait")
	writeFile(t, filepath.Join(root, "custom", "override", "item.uti"), "item")
	writeFile(t, filepath.Join(root, "unsupported.json"), "{}")

	plan, err := build("704450", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:nwnee:loose" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan, "portraits/portrait.tga")
	assertTarget(t, plan, "custom/override/item.uti")
	assertNoTarget(t, plan, "unsupported.json")
}

func TestNWN2ModuleInstallerTargetsDocumentsModules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "module", "adventure.mod"), "module")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	plan, err := build(neverwinter.NWN2ID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:neverwinter2:module" || plan.ModType != "nwn2-module" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "modules/adventure.mod")
	assertNoTarget(t, plan, "readme.txt")
}

func build(appID, root string) (installplan.Plan, error) {
	extensions := neverwinter.Extensions()
	compiled := make([]gameext.Extension, 0, len(extensions))
	for _, extension := range extensions {
		compiled = append(compiled, gameext.MustCompileExtension(extension))
	}
	return gameext.NewRegistry(compiled).BuildInstallPlan(appID, root)
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

func assertNoTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, plan.Instructions)
		}
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
