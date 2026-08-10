package kingdomcomedeliverance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersKCDCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != modsRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.LoadOrders) != 1 ||
		!hasFeature(summary.Capabilities.EventHandlers, sdk.EventWillDeploy) ||
		!hasFeature(summary.Capabilities.EventHandlers, sdk.EventDidPurge) {
		t.Fatalf("load order/event handlers = %+v / %+v", summary.Capabilities.LoadOrders, summary.Capabilities.EventHandlers)
	}
	if len(summary.Capabilities.CollectionFeatures) != 1 || summary.Capabilities.CollectionFeatures[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("collection features = %+v", summary.Capabilities.CollectionFeatures)
	}
}

func TestDefaultInstallerTargetsModsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolKCDMod", "Data", "example.pak"), "payload")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:kingdomcomedeliverance:mods" || plan.ModType != modType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Mods/Data/example.pak" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestWillDeployRewritesFoldersAndGeneratesModOrder(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	workDir := filepath.Join(root, "work")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(modOrderFile)), "Manual-Mod\nold20\n")

	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "Mods/Data/late.pak", Priority: 20},
			{InstalledModID: 10, TargetRelative: "Mods/Data/early.pak", Priority: 10},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: modType, Priority: 20},
			{ID: 10, Name: "Early", ModType: modType, Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 3 {
		t.Fatalf("result = %+v", result)
	}
	assertMapping(t, result.Mappings, "Mods/10/Data/early.pak")
	assertMapping(t, result.Mappings, "Mods/20/Data/late.pak")
	order := readFile(t, filepath.Join(workDir, "kingdomcomedeliverance-load-order", "mod_order.txt"))
	if strings.TrimSpace(order) != "10\n20\nManualMod\nold20" {
		t.Fatalf("mod_order.txt = %q", order)
	}
}

func TestDidPurgePreservesOnlyManualLiveOrderEntries(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	writeFile(t, filepath.Join(gamePath, "Mods", "ManualMod", "manual.pak"), "manual")
	writeFile(t, filepath.Join(gamePath, "Mods", "ReadmeOnly", "readme.txt"), "not a kcd mod")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(modOrderFile)), "ManualMod\n10\nMissingManual\nManualMod\nReadmeOnly\n")

	result, err := didPurge(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		ManagedFiles: []deploy.AppliedFile{
			{TargetPath: filepath.Join(gamePath, "Mods", "10", "Data", "managed.pak")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) == 0 {
		t.Fatalf("expected purge preservation message")
	}
	order := readFile(t, filepath.Join(gamePath, filepath.FromSlash(modOrderFile)))
	if strings.TrimSpace(order) != "ManualMod" {
		t.Fatalf("mod_order.txt = %q", order)
	}
}

func assertMapping(t *testing.T, mappings []deploy.FileMapping, target string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, mappings)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func hasFeature(features []gameext.FeatureSummary, id string) bool {
	for _, feature := range features {
		if feature.ID == id {
			return true
		}
	}
	return false
}
