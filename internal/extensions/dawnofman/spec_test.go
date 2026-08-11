package dawnofman_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/dawnofman"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(dawnofman.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "Mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || len(summary.Capabilities.ModTypes) != 3 || len(summary.Capabilities.Installers) != 3 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.SupportedTools) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("UMM runtime capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.GameSetups) != 1 || len(summary.Capabilities.GameSetups[0].SetupActions) != 2 {
		t.Fatalf("game setup capabilities = %+v", summary.Capabilities.GameSetups)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 1 || len(summary.Capabilities.ExtensionDashlets) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("extension metadata = %+v", summary.Capabilities)
	}
}

func TestSceneInstallerTargetsDocumentsScenarioRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Scenario", "example.scn.xml"), "scene")
	writeFile(t, filepath.Join(root, "Scenario", "preview.png"), "preview")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := build(root, "Better Scenario.zip")
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:dawnofman:scene" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan, "BetterScenario/example.scn.xml", "dawnofman-scenarios")
	assertTarget(t, plan, "BetterScenario/preview.png", "dawnofman-scenarios")
	if len(plan.Instructions) != 2 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestUMMInstallerTargetsGameModsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mod", "Info.json"), "{}")
	writeFile(t, filepath.Join(root, "Mod", "Assemblies", "Mod.dll"), "dll")

	plan, err := build(root, "Cool UMM.zip")
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:dawnofman:umm" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan, "Mods/CoolUMM/Info.json", "")
	assertTarget(t, plan, "Mods/CoolUMM/Assemblies/Mod.dll", "")
}

func TestUMMRuntimeRequirementIsSatisfiedByManagedTool(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(dawnofman.Extension())})
	requirements := registry.RuntimeRequirements(context.Background(), dawnofman.SteamAppID, t.TempDir(), []gamehandler.RuntimeMod{{
		Enabled: true,
		ModType: "dawnofman-umm-mod",
	}})
	if len(requirements) != 1 || requirements[0].Status != gamehandler.RequirementMissing || requirements[0].Acquisition == nil {
		t.Fatalf("missing UMM requirement = %+v", requirements)
	}
	requirements = registry.RuntimeRequirements(context.Background(), dawnofman.SteamAppID, t.TempDir(), []gamehandler.RuntimeMod{
		{Enabled: true, ModType: "dawnofman-umm-mod"},
		{Enabled: true, ModType: "umm", Metadata: []gamehandler.ModMetadata{{Name: "Unity Mod Manager"}}},
	})
	if len(requirements) != 1 || requirements[0].Status != gamehandler.RequirementOK {
		t.Fatalf("satisfied UMM requirement = %+v", requirements)
	}
}

func TestScenarioRootUsesDocumentsFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(dawnofman.Extension())})
	result, ok, err := registry.ResolveTargetRoot(context.Background(), dawnofman.SteamAppID, "dawnofman-scenarios", gameext.TargetRootInput{})
	if err != nil || !ok {
		t.Fatalf("resolve target root ok=%v err=%v", ok, err)
	}
	want := filepath.Join(home, "Documents", "DawnOfMan", "Scenarios")
	if result.Path != want {
		t.Fatalf("target root = %q, want %q", result.Path, want)
	}
}

func build(root, archiveName string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(dawnofman.Extension())}).BuildInstallPlanWithGamePathArchiveAndSelections(dawnofman.SteamAppID, root, "", archiveName, nil)
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
