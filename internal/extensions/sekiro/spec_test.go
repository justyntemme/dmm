package sekiro_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sekiro"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(sekiro.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestLoosePartsInstallerTargetsModsParts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "c0000.partsbnd.dcx"), "parts")
	writeFile(t, filepath.Join(root, "readme.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:sekiro:loose-parts" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan, "mods/parts/c0000.partsbnd.dcx")
	if len(plan.Instructions) != 1 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestRootInstallerPreservesRootFoldersFromPartsLevel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "parts", "c0000.partsbnd.dcx"), "parts")
	writeFile(t, filepath.Join(root, "chr", "c0000.anibnd.dcx"), "chr")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:sekiro:root-mod" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan, "mods/parts/c0000.partsbnd.dcx")
	assertTarget(t, plan, "mods/chr/c0000.anibnd.dcx")
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(sekiro.Extension())}).BuildInstallPlan(sekiro.SteamAppID, root)
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
