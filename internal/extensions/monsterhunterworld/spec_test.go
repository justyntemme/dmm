package monsterhunterworld

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestNativePCArchiveStripsThroughNativePC(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "nativePC", "wp", "sword.mod3"), "mesh")
	writeFile(t, filepath.Join(root, "wrapped", "nativePC", "readme.txt"), "readme")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != nativePCModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"nativePC/readme.txt",
		"nativePC/wp/sword.mod3",
	})
}

func TestReshadeArchiveCopiesIniToGameRootAndWarnsWhenReshadeMissing(t *testing.T) {
	root := t.TempDir()
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(root, "Preset.ini"), "preset")

	plan, err := registry().BuildInstallPlanWithGamePath(SteamAppID, root, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != reshadeModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"Preset.ini"})
	if len(plan.Warnings) != 1 {
		t.Fatalf("warnings = %+v", plan.Warnings)
	}
}

func TestStrackerArchiveCopiesRootFilesToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "loader.dll"), "dll")
	writeFile(t, filepath.Join(root, "loader-config.json"), "{}")
	writeFile(t, filepath.Join(root, "nativePC", "plugins", "plugin.dll"), "plugin")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != strackerModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"loader-config.json",
		"loader.dll",
		"nativePC/plugins/plugin.dll",
	})
}

func TestExtensionSummaryRecordsToolsAndRuntimeRequirement(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != nativePCRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.SupportedTools) != 3 {
		t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
	}
	if len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", summary.Capabilities.RuntimeRequirements)
	}
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertTargets(t *testing.T, plan installplan.Plan, targets []string) {
	t.Helper()
	found := map[string]bool{}
	for _, target := range targets {
		found[target] = false
	}
	for _, instruction := range plan.Instructions {
		if _, ok := found[instruction.TargetRelative]; ok {
			found[instruction.TargetRelative] = true
		}
	}
	for target, ok := range found {
		if !ok {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}
