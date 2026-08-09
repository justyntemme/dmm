package nomanssky_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/nomanssky"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexModTypes(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(nomanssky.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "GAMEDATA/MODS" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ModTypes) != 3 || len(summary.Capabilities.Installers) != 3 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestPakArchiveTargetsDeprecatedPCBANKSMods(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "MOD.pak"), "pak")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(nomanssky.Extension())}).BuildInstallPlan(nomanssky.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:nomanssky:deprecated-pak" || plan.ModType != "nomanssky-deprecated-pak" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "GAMEDATA/PCBANKS/MODS/wrapped/MOD.pak" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestDLLArchiveTargetsBinaries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "injector", "dxgi.dll"), "dll")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(nomanssky.Extension())}).BuildInstallPlan(nomanssky.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:nomanssky:binaries" || plan.ModType != "nomanssky-binaries" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Binaries/injector/dxgi.dll" {
		t.Fatalf("instructions = %+v", plan.Instructions)
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
