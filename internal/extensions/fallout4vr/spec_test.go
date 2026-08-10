package fallout4vr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestESLEnablerRoutesFilesUnderData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "Daytripper4.dll"), "dll")
	writeFile(t, filepath.Join(root, "Wrapper", "Data", "F4SE", "plugin.dll"), "plugin")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dinputModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"Data/Daytripper4.dll", "Data/Data/F4SE/plugin.dll"})
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "eslEnabler" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestSummaryRecordsConditionalDynamicESLSupport(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if len(summary.Capabilities.PluginActivations) != 1 {
		t.Fatalf("plugin activations = %+v", summary.Capabilities.PluginActivations)
	}
	activation, ok := registry.PluginActivationForSteamApp(SteamAppID)
	if !ok {
		t.Fatal("missing plugin activation")
	}
	if activation.SupportsLightPlugins {
		t.Fatalf("Fallout 4 VR light plugins should be conditional, not static: %+v", activation)
	}
	if activation.LightPluginsCondition == nil ||
		activation.LightPluginsCondition.MetadataKind != "vortex-attribute" ||
		activation.LightPluginsCondition.MetadataName != "eslEnabler" ||
		activation.LightPluginsCondition.MetadataUniqueID != "true" {
		t.Fatalf("light plugin condition = %+v", activation.LightPluginsCondition)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 0 {
		t.Fatalf("extension APIs = %+v", summary.Capabilities.ExtensionAPIs)
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
