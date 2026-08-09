package survivingmars_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/survivingmars"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(survivingmars.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || len(summary.Capabilities.Installers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestModContentInstallerUsesMarkerFolderAsModName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MarsMod", "ModContent.hpk"), "hpk")
	writeFile(t, filepath.Join(root, "MarsMod", "metadata.lua"), "lua")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := build(root, "ignored.zip")
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "MarsMod/ModContent.hpk", "survivingmars-appdata-mods")
	assertTarget(t, plan, "MarsMod/metadata.lua", "survivingmars-appdata-mods")
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestRootModContentInstallerUsesArchiveNameAsModName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "modcontent.hpk"), "hpk")

	plan, err := build(root, "Root Mars Mod.zip")
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "Root Mars Mod/modcontent.hpk", "survivingmars-appdata-mods")
}

func TestAppDataRootUsesSteamCompatdata(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(survivingmars.Extension())})
	library := filepath.Join(t.TempDir(), "steam-library")
	result, ok, err := registry.ResolveTargetRoot(context.Background(), survivingmars.SteamAppID, "survivingmars-appdata-mods", gameext.TargetRootInput{
		LibraryPath: library,
	})
	if err != nil || !ok {
		t.Fatalf("resolve target root ok=%v err=%v", ok, err)
	}
	want := filepath.Join(library, "steamapps", "compatdata", survivingmars.SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Roaming", "Surviving Mars", "mods")
	if result.Path != want {
		t.Fatalf("target root = %q, want %q", result.Path, want)
	}
}

func build(root, archiveName string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(survivingmars.Extension())}).BuildInstallPlanWithGamePathArchiveAndSelections(survivingmars.SteamAppID, root, "", archiveName, nil)
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
