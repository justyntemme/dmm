package ahatintime_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/ahatintime"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(ahatintime.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "HatInTimeGame/Mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.SupportedTools) != 1 || summary.Capabilities.SupportedTools[0].ID != "HatinTimeEditor" {
		t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
	}
}

func TestModInfoInstallerUsesManifestFolderAsModName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod", "modinfo.ini"), "[ModInfo]")
	writeFile(t, filepath.Join(root, "CoolMod", "Content", "asset.u"), "asset")
	writeFile(t, filepath.Join(root, "ignored", "outside.txt"), "ignored")

	plan, err := build(root, "ignored.zip")
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "HatInTimeGame/Mods/CoolMod/modinfo.ini")
	assertTarget(t, plan, "HatInTimeGame/Mods/CoolMod/Content/asset.u")
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestRootModInfoInstallerUsesArchiveNameAsModName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "modinfo.ini"), "[ModInfo]")
	writeFile(t, filepath.Join(root, "Content", "asset.u"), "asset")

	plan, err := build(root, "Root Mod.zip")
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "HatInTimeGame/Mods/Root Mod/modinfo.ini")
	assertTarget(t, plan, "HatInTimeGame/Mods/Root Mod/Content/asset.u")
}

func build(root, archiveName string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(ahatintime.Extension())}).BuildInstallPlanWithGamePathArchiveAndSelections(ahatintime.SteamAppID, root, "", archiveName, nil)
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
