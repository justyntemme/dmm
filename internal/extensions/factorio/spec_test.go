package factorio_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/factorio"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexLinuxModRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	compiled := gameext.MustCompileExtension(factorio.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{compiled}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != "factorio-user-mods" {
		t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
	}

	root, err := compiled.TargetRoots[0].Resolver(t.Context(), sdk.TargetRootInput{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".factorio", "mods")
	if root.Path != want {
		t.Fatalf("root = %q, want %q", root.Path, want)
	}
}

func TestDefaultInstallerTargetsUserModsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod_1.0.0", "info.json"), "{}")
	writeFile(t, filepath.Join(root, "CoolMod_1.0.0", "control.lua"), "script")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(factorio.Extension())}).BuildInstallPlan(factorio.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:factorio:default" || plan.ModType != "factorio-mod" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	for _, instruction := range plan.Instructions {
		if instruction.TargetRoot != "factorio-user-mods" {
			t.Fatalf("target root = %q", instruction.TargetRoot)
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
