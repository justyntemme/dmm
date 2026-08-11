package kotor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/kotor"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionsRegisterVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(kotor.Extensions()[0])}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "override" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 4 || len(summary.Capabilities.UnsupportedInstallers) != 0 || len(summary.Capabilities.ModTypes) != 4 || len(summary.Capabilities.LaunchTools) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestRootInstallerCopiesFromRecognizedGameFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "modules", "example.mod"), "module")
	writeFile(t, filepath.Join(root, "wrapped", "override", "item.uti"), "override")
	writeFile(t, filepath.Join(root, "readme.txt"), "ignored")

	plan, err := build("32370", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:kotor:root" || plan.ModType != "kotor-root" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "modules/example.mod")
	assertTarget(t, plan, "override/item.uti")
	assertNoTarget(t, plan, "readme.txt")
}

func TestOverrideInstallerCopiesPayloadUnderOverrideRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "folder", "item.uti"), "item")
	writeFile(t, filepath.Join(root, "dialog.dlg"), "dialog")

	plan, err := build("208580", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:kotor2:override" || plan.ModType != "kotor2-override" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "override/dialog.dlg")
	assertTarget(t, plan, "override/folder/item.uti")
}

func TestTSLPatcherArchivesStageManagedPatcherPayload(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "TSLPatcher.exe"), "exe")
	writeFile(t, filepath.Join(root, "tslpatchdata", "changes.ini"), "patcher")
	writeFile(t, filepath.Join(root, "override", "item.uti"), "item")

	plan, err := build("32370", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:kotor:tslpatcher-tool" && plan.PlannerID != "vortex:kotor:tslpatcher-mod" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	if plan.ModType != "kotor-tslpatcher-tool" && plan.ModType != "kotor-tslpatcher-patch" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan, "DMM/TSLPatcher/TSLPatcher.exe")
}

func TestTSLPatcherModWithoutBundledExeStagesPatchData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "tslpatchdata", "changes.ini"), "patcher")

	plan, err := build("32370", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:kotor:tslpatcher-mod" || plan.ModType != "kotor-tslpatcher-patch" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "DMM/TSLPatcher/tslpatchdata/changes.ini")
}

func build(appID, root string) (installplan.Plan, error) {
	extensions := kotor.Extensions()
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
