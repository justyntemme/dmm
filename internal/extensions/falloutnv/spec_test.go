package falloutnv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestDataArchiveInstallsToDataRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapped", "Meshes", "armor.nif"), "mesh")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dataRootModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"Data/Meshes/armor.nif"})
}

func TestFourGBPatchRoutesThroughDInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "FNVpatch.exe"), "exe")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dinputModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"FNVpatch.exe", "readme.txt"})
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "is4GBPatcher" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestSummaryRecordsPluginActivationAndLaunchTool(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if len(summary.Capabilities.PluginActivations) != 1 {
		t.Fatalf("plugin activations = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.LaunchTools) != 1 || summary.Capabilities.LaunchTools[0].ID != "nvse" {
		t.Fatalf("launch tools = %+v", summary.Capabilities.LaunchTools)
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
